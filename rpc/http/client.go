package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"

	"github.com/CXeon/tiles/rpc"
)

// Config 是 HTTP 客户端的构造参数
type Config struct {
	// BaseURL 直接指定服务地址，与 Service 二选一，如 "http://localhost:8080"
	BaseURL string
	// Service 服务名，配合 Resolver 使用，与 BaseURL 二选一
	Service string
	// Timeout 默认请求超时，0 表示不限时；可被 WithTimeout 逐次覆盖
	Timeout time.Duration
	// ContentType 默认 Content-Type，不填则为 "application/json"
	ContentType string
	// TraceIDExtractor 从 ctx 自动提取 TraceID；可被 WithTraceID 逐次覆盖
	TraceIDExtractor func(ctx context.Context) string
	// 公司名称
	Company string
	// 项目名称
	Project string
	// 染色
	Color string
}

// HTTPError 表示服务端返回了 4xx/5xx 传输层错误
type HTTPError struct {
	Code    uint   // HTTP 状态码
	Message string // 响应体内容
	TraceID string // 响应头 X-Trace-ID
}

func (e *HTTPError) Error() string {
	if e.TraceID == "" {
		return fmt.Sprintf("http error %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("http error %d: %s (trace_id=%s)", e.Code, e.Message, e.TraceID)
}

type client struct {
	cfg        Config
	resolver   rpc.Resolver
	httpClient *http.Client
}

// New 创建 HTTP RPC 客户端
// resolver 为 nil 时使用 cfg.BaseURL；不为 nil 时用 cfg.Service 做服务发现
func New(cfg Config, resolver rpc.Resolver) (rpc.Client, error) {
	if cfg.BaseURL == "" && resolver == nil {
		return nil, errors.New("BaseURL and resolver cannot both be empty")
	}
	if resolver != nil && cfg.Service == "" {
		return nil, errors.New("service name is required when resolver is set")
	}
	if cfg.ContentType == "" {
		cfg.ContentType = "application/json"
	}
	return &client{
		cfg:        cfg,
		resolver:   resolver,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}, nil
}

func WithPath(path string) rpc.CallOption {
	return func(opts *rpc.CallOptions) {
		opts.Path = path
	}
}

func WithHeader(key, value string) rpc.CallOption {
	return func(opts *rpc.CallOptions) {
		if opts.Headers == nil {
			opts.Headers = make(map[string]string)
		}
		opts.Headers[key] = value
	}
}

func WithQuery(key, value string) rpc.CallOption {
	return func(opts *rpc.CallOptions) {
		if opts.Query == nil {
			opts.Query = make(map[string]string)
		}
		opts.Query[key] = value
	}
}

func WithTimeout(d time.Duration) rpc.CallOption {
	return func(opts *rpc.CallOptions) {
		opts.Timeout = d
	}
}

func WithTraceID(traceID string) rpc.CallOption {
	return func(opts *rpc.CallOptions) {
		opts.TraceID = traceID
	}
}

func (c *client) Invoke(ctx context.Context, method string, req, resp any, opts ...rpc.CallOption) error {
	// 1. 聚合所有 CallOption
	co := &rpc.CallOptions{}
	for _, o := range opts {
		o(co)
	}

	// 2. path 必填
	if co.Path == "" {
		return rpc.ErrPathRequired
	}

	// 3. 解析目标地址
	baseURL := c.cfg.BaseURL
	if c.resolver != nil {
		var err error
		baseURL, err = c.resolver.Resolve(ctx, c.cfg.Company, c.cfg.Project, c.cfg.Service, c.cfg.Color)
		if err != nil {
			return fmt.Errorf("%w: %w", rpc.ErrResolverFailed, err)
		}
	}

	// 4. 序列化请求体
	var bodyReader io.Reader
	if req != nil {
		data, err := sonic.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// 5. 逐次超时覆盖（WithTimeout 优先于 Config.Timeout）
	if co.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, co.Timeout)
		defer cancel()
	}

	// 6. 构造 HTTP Request
	httpReq, err := http.NewRequestWithContext(ctx, method, baseURL+co.Path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	// 7. 设置 Header
	httpReq.Header.Set("Content-Type", c.cfg.ContentType)
	for k, v := range co.Headers {
		httpReq.Header.Set(k, v)
	}

	// 8. 设置 X-Trace-ID：WithTraceID > TraceIDExtractor > 不设置
	traceID := co.TraceID
	if traceID == "" && c.cfg.TraceIDExtractor != nil {
		traceID = c.cfg.TraceIDExtractor(ctx)
	}
	if traceID != "" {
		httpReq.Header.Set("X-Trace-ID", traceID)
	}

	// 9. 设置 Query 参数
	if len(co.Query) > 0 {
		q := httpReq.URL.Query()
		for k, v := range co.Query {
			q.Set(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// 10. 执行请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// 11. 4xx/5xx 传输层错误
	if httpResp.StatusCode >= 400 {
		return &HTTPError{
			Code:    uint(httpResp.StatusCode),
			Message: string(body),
			TraceID: httpResp.Header.Get("X-Trace-ID"),
		}
	}

	// 12. 反序列化响应包络
	var envelope rpc.Response
	if err := sonic.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: %w", rpc.ErrInvalidResponse, err)
	}

	// 13. 业务错误
	if envelope.Code != 0 {
		return &rpc.ResponseError{
			Code:    envelope.Code,
			Message: envelope.Message,
			TraceID: envelope.TraceID,
		}
	}

	// 14. 反序列化 Data 到 resp
	if resp != nil && len(envelope.Data) > 0 {
		if err := sonic.Unmarshal(envelope.Data, resp); err != nil {
			return fmt.Errorf("%w: %w", rpc.ErrInvalidResponse, err)
		}
	}

	return nil
}

func (c *client) Close(_ context.Context) error {
	c.httpClient.CloseIdleConnections()
	return nil
}
