package traefik

import (
	"context"
	"fmt"
	"slices"

	"github.com/CXeon/tiles/gateway"
)

type traefikClient struct {
	handler          *handler
	middlewares      []string // 全局默认中间件
	excludeAuthPaths []string // 排除身份验证的完整路径，如/Company/Project/Service/Public
	healthCheckPath  string
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
func WithDefaultMiddlewares(middlewares []string) ClientOption {
	return func(c *traefikClient) {
		c.middlewares = middlewares
	}
}

// WithHealthCheckPath 用户自定义健康检查路径
func WithHealthCheckPath(path string) ClientOption {
	return func(c *traefikClient) {
		c.healthCheckPath = path
	}
}

func NewClient(ctx context.Context, provider *Provider, opts ...ClientOption) (gateway.Client, error) {
	h, err := NewHandler(ctx, provider)
	if err != nil {
		return nil, err
	}
	c := &traefikClient{
		handler:     h,
		middlewares: []string{},
	}

	for _, opt := range opts {
		opt(c)
	}

	// 如果中间件不包括ForwardAuth,手动添加
	if !slices.Contains(c.middlewares, "ForwardAuth") {
		c.middlewares = append(c.middlewares, "ForwardAuth")
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

	// 协议转换: https -> http
	normalized := *endpoint
	if normalized.Protocol == gateway.ProtocolTypeHttps {
		normalized.Protocol = gateway.ProtocolTypeHttp
	}
	return normalized, nil
}

// buildHandlerOptions 构造 HandlerOptions
func (c *traefikClient) buildHandlerOptions() HandlerOptions {
	return HandlerOptions{
		Middlewares:      c.middlewares,
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
	return c.handler.Register(normalized, opts)
}

func (c *traefikClient) Deregister(ctx context.Context, endpoint *gateway.Endpoint) error {
	normalized, err := normalizeEndpoint(endpoint)
	if err != nil {
		return err
	}
	return c.handler.Deregister(normalized)
}

func (c *traefikClient) Update(ctx context.Context, endpoint *gateway.Endpoint) error {
	normalized, err := normalizeEndpoint(endpoint)
	if err != nil {
		return err
	}
	opts := c.buildHandlerOptions()
	return c.handler.Update(normalized, opts)
}

func (c *traefikClient) Close(ctx context.Context) error {
	return c.handler.Close()
}
