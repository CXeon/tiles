# Context（上下文模块）

提供微服务链路中的标准化请求上下文，用于在服务内部和跨服务调用之间透传通用字段。

## 特性

- **链路字段封装**：携带 `TraceID`、`Env`、`Cluster`、`UserID`、`Color` 等标准链路字段
- **HTTP Header 提取**：支持从 HTTP Header 自动提取字段，供网关/中间件使用
- **扩展字段支持**：通过 `Extra` 存储任意业务自定义字段
- **标准接口兼容**：嵌套 `context.Context`，可直接传入所有标准库及第三方函数
- **安全复制**：`NewAppContext` 从现有 `*AppContext` 中深拷贝字段，避免意外共享

## 快速开始

### 在 HTTP 中间件中提取上下文

```go
import tilecontext "github.com/CXeon/tiles/context"

// Gin 示例
func TileContextMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        appCtx := tilecontext.NewFromHTTPHeaders(c.Request.Context(), c.Request.Header)
        c.Request = c.Request.WithContext(appCtx)
        c.Next()
    }
}
```

### 在 Handler 中读取字段

```go
func GetUserHandler(c *gin.Context) {
    appCtx := tilecontext.From(c.Request.Context())

    log.Info("request received", logger.Fields{
        "trace_id": appCtx.TraceID(),
        "user_id":  appCtx.UserID(),
        "env":      appCtx.Env(),
    })
}
```

### 在服务内部传递

```go
func (svc *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    // 从传入 ctx 构造 AppContext，保留链路字段
    appCtx := tilecontext.NewAppContext(ctx)

    // 继续传递给下游
    return svc.repo.Find(appCtx, id)
}
```

### 设置和读取扩展字段

```go
appCtx := tilecontext.NewAppContext(ctx)
appCtx.SetExtra("request_id", "req-abc123")
appCtx.SetExtra("platform", "ios")

// 读取
requestID := appCtx.Extra("request_id") // any 类型
```

## AppContext 结构

```go
type AppContext struct {
    context.Context          // 嵌套标准 context
    // 内部封装以下字段：
    // TraceID, Env, Cluster, UserID, Color string
    // Extra map[string]any
}
```

## 构造函数

| 函数 | 说明 | 典型使用场景 |
|------|------|-------------|
| `NewFromHTTPHeaders(ctx, headers)` | 从 HTTP Header 提取字段 | HTTP 中间件 |
| `NewAppContext(ctx)` | 从 context 构造，若已是 `*AppContext` 则复制字段 | 服务内部传递 |
| `From(ctx)` | 从 context 提取 `*AppContext`，失败时返回空实例 | Handler 读取字段 |

## HTTP Header 常量

| 常量 | Header 名称 | 说明 |
|------|------------|------|
| `HeaderTraceID` | `X-Trace-Id` | 链路追踪 ID |
| `HeaderEnv` | `X-Env` | 环境标识（dev / test / prod） |
| `HeaderCluster` | `X-Cluster` | 集群标识 |
| `HeaderUserID` | `X-User-Id` | 当前用户 ID |
| `HeaderColor` | `X-Color` | 染色标识（蓝绿/灰度） |

## 字段读取方法

| 方法 | 返回类型 | 说明 |
|------|---------|------|
| `TraceID()` | `string` | 链路追踪 ID |
| `Env()` | `string` | 环境标识 |
| `Cluster()` | `string` | 集群标识 |
| `UserID()` | `string` | 当前用户 ID |
| `Color()` | `string` | 染色标识 |
| `Extra(key)` | `any` | 读取扩展字段 |
| `SetExtra(key, value)` | - | 写入扩展字段 |

## 运行测试

```bash
go test github.com/CXeon/tiles/context/... -v
```

## 相关链接

- [tiles 项目主页](../README.md)
