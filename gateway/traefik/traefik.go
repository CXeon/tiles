package traefik

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CXeon/tiles/gateway"
)

type traefikClient struct {
	handler *handler
	//	middlewares      []string // 全局默认中间件
	excludeAuthPaths []string // 排除身份验证的完整路径，如/Company/Project/Service/Public
	healthCheckPath  string
	autoRenew        bool                     // 是否自动续约（默认 true）
	renewGoroutines  map[string]chan struct{} // 续约协程管理，key 为 endpoint.ID()
	mu               sync.Mutex               // 保护 renewGoroutines
}

// ClientOption 定义客户端配置选项
type ClientOption func(*traefikClient)

// WithExcludeAuthPaths 设置不需要身份验证的路径
// 注意：paths 应为完整的 HTTP 路径（如 /company/project/service/public），不会被内部修改
func WithExcludeAuthPaths(paths []string) ClientOption {
	return func(c *traefikClient) {
		c.excludeAuthPaths = paths
	}
}

// WithDefaultMiddlewares 设置默认中间件
// func WithDefaultMiddlewares(middlewares []string) ClientOption {
// 	return func(c *traefikClient) {
// 		c.middlewares = middlewares
// 	}
// }

// WithHealthCheckPath 用户自定义健康检查路径
func WithHealthCheckPath(path string) ClientOption {
	return func(c *traefikClient) {
		c.healthCheckPath = path
	}
}

// WithAutoRenew 配置是否自动续约（默认 true）
func WithAutoRenew(enabled bool) ClientOption {
	return func(c *traefikClient) {
		c.autoRenew = enabled
	}
}

// NewClient 创建Traefik网关客户端
//
// TTL 机制说明：
//   - 当 Endpoint.TTL > 0 时，服务实例配置会在 TTL 秒后自动过期
//   - 默认启用自动续约（autoRenew=true），无需手动调用 KeepAlive
//   - 高级用户可通过 WithAutoRenew(false) 禁用自动续约，自行管理心跳
//
// 使用示例：
//
// 示例 1：默认自动续约（推荐）
//
//	client, _ := traefik.NewClient(ctx, provider)
//	endpoint := &gateway.Endpoint{
//	    // ... 其他字段 ...
//	    TTL: 15, // 15 秒 TTL
//	}
//	client.Register(ctx, endpoint)
//	// 后台自动续约，用户无需关心
//	defer client.Deregister(ctx, endpoint)
//
// 示例 2：手动续约（高级）
//
//	client, _ := traefik.NewClient(ctx, provider, traefik.WithAutoRenew(false))
//	endpoint := &gateway.Endpoint{
//	    // ... 其他字段 ...
//	    TTL: 15,
//	}
//	client.Register(ctx, endpoint)
//	// 用户自己管理心跳
//	go func() {
//	    ticker := time.NewTicker(10 * time.Second)
//	    defer ticker.Stop()
//	    for range ticker.C {
//	        client.KeepAlive(ctx, endpoint)
//	    }
//	}()
//	defer client.Deregister(ctx, endpoint)
func NewClient(ctx context.Context, provider *Provider, opts ...ClientOption) (gateway.Client, error) {
	h, err := NewHandler(ctx, provider)
	if err != nil {
		return nil, err
	}
	c := &traefikClient{
		handler:         h,
		autoRenew:       true, // 默认启用自动续约
		renewGoroutines: make(map[string]chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	// 如果用户没有设置健康检查路径，使用默认路径
	if len(c.healthCheckPath) == 0 {
		c.healthCheckPath = "/health"
	}

	return c, nil
}

// normalizeEndpoint 验证并归一化 endpoint
func normalizeEndpoint(endpoint *gateway.Endpoint) (gateway.Endpoint, error) {
	if endpoint == nil {
		return gateway.Endpoint{}, fmt.Errorf("endpoint is nil")
	}
	if err := endpoint.Protocol.Validate(); err != nil {
		return gateway.Endpoint{}, err
	}

	normalized := *endpoint

	if len(normalized.Color) == 0 {
		normalized.Color = "clear"
	}

	return normalized, nil
}

// buildHandlerOptions 构造 HandlerOptions
func (c *traefikClient) buildHandlerOptions() HandlerOptions {
	return HandlerOptions{
		ExcludeAuthPaths: c.excludeAuthPaths,
		HealthCheckPath:  c.healthCheckPath,
	}
}

func (c *traefikClient) Register(ctx context.Context, endpoint *gateway.Endpoint) error {
	normalized, err := normalizeEndpoint(endpoint)
	if err != nil {
		return err
	}
	opts := c.buildHandlerOptions()
	err = c.handler.Register(ctx, normalized, opts)
	if err != nil {
		return err
	}

	// 如果启用自动续约且 TTL > 0，启动续约协程
	if c.autoRenew && normalized.TTL > 0 {
		c.startAutoRenew(ctx, normalized, opts)
	}

	return nil
}

func (c *traefikClient) Deregister(ctx context.Context, endpoint *gateway.Endpoint) error {
	normalized, err := normalizeEndpoint(endpoint)
	if err != nil {
		return err
	}

	// 停止续约协程
	c.stopAutoRenew(normalized.ID())

	return c.handler.Deregister(ctx, normalized)
}

func (c *traefikClient) Update(ctx context.Context, endpoint *gateway.Endpoint) error {
	normalized, err := normalizeEndpoint(endpoint)
	if err != nil {
		return err
	}
	opts := c.buildHandlerOptions()
	return c.handler.Update(ctx, normalized, opts)
}

func (c *traefikClient) Close(ctx context.Context) error {
	// 停止所有续约协程
	c.mu.Lock()
	for endpointID, stopChan := range c.renewGoroutines {
		close(stopChan)
		delete(c.renewGoroutines, endpointID)
	}
	c.mu.Unlock()

	return c.handler.Close()
}

// KeepAlive 手动续约 TTL（供高级用户使用）
func (c *traefikClient) KeepAlive(ctx context.Context, endpoint *gateway.Endpoint) error {
	normalized, err := normalizeEndpoint(endpoint)
	if err != nil {
		return err
	}
	return c.handler.Refresh(ctx, normalized)
}

// startAutoRenew 启动自动续约协程
func (c *traefikClient) startAutoRenew(ctx context.Context, endpoint gateway.Endpoint, opts HandlerOptions) {
	endpointID := endpoint.ID()

	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果已经存在，先停止旧的
	if stopChan, exists := c.renewGoroutines[endpointID]; exists {
		close(stopChan)
	}

	stopChan := make(chan struct{})
	c.renewGoroutines[endpointID] = stopChan

	go func() {
		// 心跳间隔 = TTL / 3
		interval := time.Duration(endpoint.TTL) * time.Second / 3
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 尝试续约
				err := c.handler.Refresh(ctx, endpoint)
				if err != nil {
					// Refresh 失败，尝试重新 Register
					err = c.handler.Register(ctx, endpoint, opts)
					if err != nil {
						// Register 也失败，记录日志（这里简单忽略）
						// TODO: 添加日志
					}
				}
			}
		}
	}()
}

// stopAutoRenew 停止自动续约协程
func (c *traefikClient) stopAutoRenew(endpointID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if stopChan, exists := c.renewGoroutines[endpointID]; exists {
		close(stopChan)
		delete(c.renewGoroutines, endpointID)
	}
}
