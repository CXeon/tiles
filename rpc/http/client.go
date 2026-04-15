package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

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

// Invoke stub — 将在 Task 4 中替换为完整实现
func (c *client) Invoke(_ context.Context, _ string, _, _ any, _ ...rpc.CallOption) error {
	return errors.New("not implemented")
}

func (c *client) Close(_ context.Context) error {
	return nil
}
