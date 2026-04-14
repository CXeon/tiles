# RPC 模块设计文档

日期：2026-04-15

## 概述

在 tiles 项目中新增 `rpc` 模块，提供通用 RPC 客户端接口，首期实现 HTTP 客户端。设计目标：模块间保持独立（rpc 不依赖 registry），接口具备协议扩展性（未来可接入 gRPC）。

## 包结构

```
rpc/
├── rpc.go        # 接口定义、类型约定、标准错误
└── http/
    └── client.go # HTTP 实现
```

## 核心设计

### 解耦 registry 依赖

`rpc` 包内定义 `Resolver` 接口，调用方自行决定如何实现（可用 registry，也可用静态地址），rpc 与 registry 互不 import。

### 接口风格

使用通用 `Invoke` 方法，`method` 为显式参数（HTTP 动词 / gRPC 方法名），`path` 为 HTTP 特有概念，下沉为 `CallOption`，保持接口协议无关。

### 客户端生命周期

客户端在应用启动时作为基础设施初始化（一个目标服务对应一个客户端），复用连接池，不在每次调用前创建。

---

## rpc/rpc.go

### Resolver

```go
type Resolver interface {
    Resolve(ctx context.Context, service string) (string, error)
}
```

每次 `Invoke` 时调用，返回目标服务的 base URL（如 `http://192.168.1.1:8080`），支持负载均衡（每次可能返回不同实例）。

### Client

```go
type Client interface {
    Invoke(ctx context.Context, method string, req, resp any, opts ...CallOption) error
    Close(ctx context.Context) error
}
```

- `method`：HTTP 动词（GET/POST/...）或 gRPC 方法名，始终显式传递
- `req`：请求体，nil 表示无 body
- `resp`：响应数据的反序列化目标指针，nil 表示忽略响应体
- `opts`：每次调用的附加选项

### CallOption

```go
type CallOption func(*CallOptions)

type CallOptions struct {
    Path    string
    Headers map[string]string
    Query   map[string]string
    Timeout time.Duration
    TraceID string // 链路追踪 ID，HTTP 实现转为 X-Trace-ID 请求头
}
```

### 标准响应包络

```go
type Response struct {
    Code    uint            `json:"code"`
    Message string          `json:"message"`
    TraceID string          `json:"trace_id,omitempty"`
    Data    json.RawMessage `json:"data,omitempty"`
}
```

约定：`Code == 0` 为成功，非零为业务错误。`Data` 使用 `json.RawMessage` 支持二次反序列化到用户结构体，避免 `any` 导致的 `map[string]interface{}` 问题。

### 错误类型

```go
// ResponseError：HTTP 200 但业务 Code != 0
type ResponseError struct {
    Code    uint
    Message string
    TraceID string
}

// 标准错误变量
var (
    ErrPathRequired    = errors.New("path is required, use WithPath()")
    ErrResolverFailed  = errors.New("resolver failed to resolve service address")
    ErrInvalidResponse = errors.New("failed to decode response")
)
```

---

## rpc/http/client.go

### Config

```go
type Config struct {
    BaseURL     string        // 直接指定地址，与 Service 二选一
    Service     string        // 服务名，配合 Resolver 使用，与 BaseURL 二选一
    Timeout     time.Duration // 默认请求超时，0 表示不限时；可被 WithTimeout 逐次覆盖
    ContentType string        // 默认 Content-Type，不填则为 "application/json"
}
```

### 构造函数

```go
func New(cfg Config, resolver rpc.Resolver) (rpc.Client, error)
```

校验规则：
- `BaseURL` 与 `resolver` 不能同时为空
- `resolver` 不为 nil 时，`Service` 必须非空
- `ContentType` 默认填充为 `"application/json"`

内部使用标准 `http.Client`，`Timeout` 设置在 transport 层。

### CallOption 函数

```go
func WithPath(path string) rpc.CallOption
func WithHeader(key, value string) rpc.CallOption
func WithQuery(key, value string) rpc.CallOption
func WithTimeout(d time.Duration) rpc.CallOption
func WithTraceID(traceID string) rpc.CallOption
```

### Invoke 执行流程

1. 聚合所有 `CallOption` 到 `CallOptions`
2. 校验 `Path` 非空，否则返回 `rpc.ErrPathRequired`
3. 解析 base URL：`resolver` 非 nil 则调用 `Resolve(ctx, service)`，否则用 `cfg.BaseURL`
4. 使用 `bytedance/sonic` 序列化 `req` 为 JSON body
5. 若 `co.Timeout > 0`，用其覆盖 context 超时（优先于 `Config.Timeout`）
6. 构造 `http.Request`，设置 `Content-Type` 及自定义 Header
7. 若 `co.TraceID != ""`，设置 `X-Trace-ID` 请求头
8. 设置 Query 参数
9. 执行请求
10. HTTP 4xx/5xx → 返回 `HTTPError`
11. HTTP 200 → 反序列化包络 `rpc.Response`：`Code != 0` 返回 `rpc.ResponseError`；`Code == 0` 将 `Data` 反序列化到 `resp`

### 传输错误类型

```go
// HTTPError：4xx/5xx 传输层错误
type HTTPError struct {
    Code    uint   // HTTP 状态码
    Message string // 响应体内容
    TraceID string
}
```

### Close

调用 `http.Client.CloseIdleConnections()` 释放空闲连接，返回 nil。

---

## 典型用法

```go
// 场景 A：直接指定地址
c, _ := httprpc.New(httprpc.Config{
    BaseURL: "http://localhost:8080",
    Timeout: 5 * time.Second,
}, nil)

// 场景 B：通过 registry 做服务发现
c, _ := httprpc.New(httprpc.Config{
    Service: "user-service",
    Timeout: 5 * time.Second,
}, myRegistryResolver)

// 调用
var resp UserResponse
err := c.Invoke(ctx, "POST", &CreateUserReq{Name: "Alice"}, &resp,
    httprpc.WithPath("/api/users"),
    httprpc.WithHeader("Authorization", "Bearer token"),
    httprpc.WithTraceID("abc-123"),
)

// 错误处理
var httpErr *httprpc.HTTPError
var respErr *rpc.ResponseError
switch {
case errors.As(err, &httpErr):
    // httpErr.Code 是 HTTP 状态码（404、500...）
case errors.As(err, &respErr):
    // respErr.Code 是业务错误码
}
```

---

## 不在本期范围内

- 中间件 / 拦截器机制
- gRPC 实现
- 重试、熔断策略
