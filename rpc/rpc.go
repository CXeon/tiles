package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrPathRequired    = errors.New("path is required, use WithPath()")
	ErrResolverFailed  = errors.New("resolver failed to resolve service address")
	ErrInvalidResponse = errors.New("failed to decode response")
)

// Resolver 将服务名解析为 base URL（如 "http://192.168.1.1:8080"）
// 调用方可用 registry.Client 实现此接口，rpc 包不直接依赖 registry
type Resolver interface {
	Resolve(ctx context.Context, service string) (string, error)
}

// CallOption 修改单次调用的选项
type CallOption func(*CallOptions)

// CallOptions 聚合单次 Invoke 调用的所有选项
type CallOptions struct {
	Path    string
	Headers map[string]string
	Query   map[string]string
	Timeout time.Duration
	TraceID string // 显式覆盖，优先级高于 Config.TraceIDExtractor
}

// Response 是服务间通信的统一响应包络
// Code == 0 表示成功；非零为业务错误
type Response struct {
	Code    uint            `json:"code"`
	Message string          `json:"message"`
	TraceID string          `json:"trace_id,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ResponseError 表示服务端返回了业务错误（HTTP 200 但 Code != 0）
type ResponseError struct {
	Code    uint
	Message string
	TraceID string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("rpc error %d: %s (trace_id=%s)", e.Code, e.Message, e.TraceID)
}

// Client 是通用 RPC 客户端接口
type Client interface {
	Invoke(ctx context.Context, method string, req, resp any, opts ...CallOption) error
	Close(ctx context.Context) error
}
