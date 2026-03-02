# Traefik Gateway Client

Traefik Gateway Client 是 Tiles 网关模块的 Traefik 3.x 实现，提供了服务实例动态注册、配置管理和 TTL 自动续约等功能。

## 特性

- **动态配置管理**：支持 Traefik 3.x Router 和 Service 的动态配置
- **多存储后端**：支持 Redis、Consul、Etcd、ZooKeeper 作为配置存储
- **TTL 自动续约**：可选的 TTL 机制，默认启用自动续约（心跳间隔 TTL/3）
- **灵活的路由配置**：支持自定义中间件、健康检查、权重分配
- **多实例协调**：自动管理多实例注册，避免配置冲突
- **流量隔离**：内置环境、集群、染色等维度的流量隔离

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "github.com/CXeon/tiles/gateway"
    "github.com/CXeon/tiles/gateway/traefik"
)

func main() {
    ctx := context.Background()

    // 创建 KV Store Provider（以 Redis 为例）
    provider := &traefik.Provider{
        Type: "redis",
        Endpoints: []string{"localhost:6379"},
    }

    // 创建 Traefik Gateway Client
    client, err := traefik.NewClient(ctx, provider)
    if err != nil {
        panic(err)
    }
    defer client.Close(ctx)

    // 定义服务实例信息
    endpoint := &gateway.Endpoint{
        Env:      "prod",
        Cluster:  "china",
        Company:  "mycompany",
        Project:  "myproject",
        Service:  "myservice",
        Protocol: gateway.ProtocolTypeHttp,
        Ip:       "192.168.1.100",
        Port:     8080,
        Color:    "blue",
        TTL:      30, // 30 秒 TTL，自动续约
        Weight:   100,
    }

    // 注册服务实例
    err = client.Register(ctx, endpoint)
    if err != nil {
        panic(err)
    }

    // 运行服务...
    // 后台自动续约，无需手动维护心跳

    // 服务停止时注销
    err = client.Deregister(ctx, endpoint)
    if err != nil {
        panic(err)
    }
}
```

## 高级配置

### 自定义中间件和健康检查

```go
client, err := traefik.NewClient(ctx, provider,
    traefik.WithMiddlewares([]string{"ForwardAuth", "RateLimit"}),
    traefik.WithExcludeAuthPaths([]string{"/public", "/health"}),
    traefik.WithHealthCheckPath("/ping"),
)
```

### 禁用自动续约（手动管理心跳）

```go
import "time"

client, err := traefik.NewClient(ctx, provider,
    traefik.WithAutoRenew(false), // 禁用自动续约
)

// 手动续约
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        client.KeepAlive(ctx, endpoint)
    }
}()
```

### 使用不同的 KV Store

#### Redis
```go
provider := &traefik.Provider{
    Type: "redis",
    Endpoints: []string{"localhost:6379"},
    Password: "your-password",  // 可选
}
```

#### Consul
```go
provider := &traefik.Provider{
    Type: "consul",
    Endpoints: []string{"localhost:8500"},
}
```

#### Etcd
```go
provider := &traefik.Provider{
    Type: "etcd",
    Endpoints: []string{"localhost:2379"},
}
```

#### ZooKeeper
```go
provider := &traefik.Provider{
    Type: "zookeeper",
    Endpoints: []string{"localhost:2181"},
}
```

## TTL 机制

### TTL 工作原理

Traefik Client 支持可选的 TTL（Time To Live）机制，用于自动清理崩溃服务的配置：

- **服务实例配置**（URL、Weight）带 TTL，会在 TTL 秒后自动过期
- **路由配置**（Router、Middleware、HealthCheck）不带 TTL，保持稳定
- **默认启用自动续约**：心跳间隔为 TTL/3，自动处理续约失败重试
- **批量同步续约**：使用 BatchKeepAlive 确保多个 key 的 TTL 同步，避免时间漂移

### 使用 TTL

```go
endpoint := &gateway.Endpoint{
    // ... 其他字段 ...
    TTL: 30, // 30 秒 TTL，默认自动续约
}

// 注册后，后台自动每 10 秒（TTL/3）续约一次
client.Register(ctx, endpoint)
```

### 手动续约模式

```go
// 禁用自动续约
client, _ := traefik.NewClient(ctx, provider,
    traefik.WithAutoRenew(false),
)

// 手动调用 KeepAlive
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        err := client.KeepAlive(ctx, endpoint)
        if err != nil {
            // 续约失败，尝试重新注册
            client.Register(ctx, endpoint)
        }
    }
}()
```

## 流量隔离

Traefik Client 内置多维度流量隔离支持：

### 环境隔离（Env）
```go
endpoint := &gateway.Endpoint{
    Env: "prod", // 或 "dev", "test"
    // ...
}
```

### 集群隔离（Cluster）
```go
endpoint := &gateway.Endpoint{
    Cluster: "china", // 或 "america", "europe"
    // ...
}
```

### 染色隔离（Color）
```go
endpoint := &gateway.Endpoint{
    Color: "blue", // 或 "green", "gray"（蓝绿部署、灰度发布）
    // ...
}
```

生成的路由规则会自动添加 Header 匹配条件：
```
PathPrefix(`/mycompany/myproject/myservice/`) 
  && Header(`X-Env`, `prod`) 
  && Header(`X-Cluster`, `china`) 
  && Header(`X-Color`, `blue`)
```

## 中间件和公开路由

### 配置认证中间件

```go
client, err := traefik.NewClient(ctx, provider,
    traefik.WithMiddlewares([]string{"ForwardAuth"}),
)
```

### 配置公开路由（排除认证）

```go
client, err := traefik.NewClient(ctx, provider,
    traefik.WithMiddlewares([]string{"ForwardAuth"}),
    traefik.WithExcludeAuthPaths([]string{
        "/mycompany/myproject/myservice/public",
        "/mycompany/myproject/myservice/health",
    }),
)
```

这会生成两个 Router：
1. **受保护路由**（默认）：带 ForwardAuth 中间件
2. **公开路由**（排除认证）：优先级更高（Priority: 1000），不带中间件

## KV Store 抽象层

位于 `kv_store/` 目录，统一多种 KV 存储后端：

### 支持的后端

| 后端 | TTL 实现 | 特点 |
|------|----------|------|
| Redis | EXPIRE | Pipeline 批量续约，确保时间同步 |
| Consul | Session | 全局共享 Session，自动同步 |
| Etcd | Lease | 自动后台续约 |
| ZooKeeper | Ephemeral Node | 连接断开自动清理 |

### KV Store 接口

```go
type KvStore interface {
    // Put 保存数据，可选 TTL 参数（单位：秒）
    Put(ctx context.Context, key string, value []byte, expired ...uint32) error
    
    // Get 获取单个 key
    Get(ctx context.Context, key string) ([]byte, error)
    
    // GetByPrefix 批量获取指定前缀的所有 key
    GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error)
    
    // Delete 删除单个 key
    Delete(ctx context.Context, key string) error
    
    // DeleteByPrefix 批量删除指定前缀的所有 key
    DeleteByPrefix(ctx context.Context, prefix string) error
    
    // KeepAlive 续约单个 key 的 TTL
    KeepAlive(ctx context.Context, key string, ttl ...uint32) error
    
    // BatchKeepAlive 批量续约多个 key，确保 TTL 同步
    BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error
    
    // Close 关闭连接
    Close() error
}
```

## 测试

```bash
# 运行所有测试
go test github.com/CXeon/tiles/gateway/traefik/... -v

# 运行特定测试
go test github.com/CXeon/tiles/gateway/traefik/... -v -run TestTraefikClient_Register

# 查看测试覆盖率
go test github.com/CXeon/tiles/gateway/traefik/... -cover
```

## 架构说明

### 组件关系

```
Client (traefikClient)
  ├─ Handler (handler)
  │   ├─ Constructor (constructor)
  │   └─ KV Store (kv_store.KvStore)
  │       ├─ Redis
  │       ├─ Consul
  │       ├─ Etcd
  │       └─ ZooKeeper
  └─ Auto Renewal Goroutines
```

### 配置生成规则

#### Router 配置
- **Router Rule**：`traefik/http/routers/<name>/rule`
  - 值：`PathPrefix(...) && Header(...)`
- **Router Service**：`traefik/http/routers/<name>/service`
  - 值：逻辑服务标识（如 `prod.china.mycompany.myproject.myservice.http.blue`）
- **Router Middlewares**：`traefik/http/routers/<name>/middlewares/<index>`
  - 值：中间件名称（如 `ForwardAuth`）
- **Router Entrypoints**：`traefik/http/routers/<name>/entrypoints/<index>`
  - 值：`web`, `websecure`

#### Service 配置
- **Service URL**：`traefik/http/services/<name>/loadbalancer/servers/<index>/url`
  - 值：`http://IP:Port`
  - **带 TTL**（会自动过期）
- **Service Weight**：`traefik/http/services/<name>/loadbalancer/servers/<index>/weight`
  - 值：权重数值（如 `100`）
  - **带 TTL**（会自动过期）
- **HealthCheck Path**：`traefik/http/services/<name>/loadbalancer/healthcheck/path`
  - 值：健康检查路径（如 `/health`）
  - **不带 TTL**（保持稳定）

## 常见问题

### Q1: 服务崩溃后配置会残留吗？

**A**: 不会。如果配置了 TTL（`endpoint.TTL > 0`），服务实例配置（URL、Weight）会在 TTL 秒后自动过期。Router 配置保持稳定，不会过期（符合预期，新实例可以直接复用）。

### Q2: 自动续约失败会怎样？

**A**: 自动续约失败后，会尝试重新执行 `Register` 操作，重新写入所有配置。如果持续失败，配置最终会在 TTL 到期后被清理。

### Q3: 为什么 Router 配置不带 TTL？

**A**: Router 配置（路由规则、中间件）是全局共享的，多个实例复用同一套 Router。如果带 TTL，最后一个实例下线后 Router 会被删除，新实例注册时需要重新创建，影响性能。保持 Router 稳定，Traefik 会在没有可用实例时返回 503。

### Q4: 如何避免 Redis TTL 时间漂移？

**A**: 使用 `BatchKeepAlive` 方法批量续约。Redis 实现使用 Pipeline 批量执行 EXPIRE 命令，确保所有 key 在毫秒级别内同步过期时间。

### Q5: 可以不使用 TTL 吗？

**A**: 可以。设置 `endpoint.TTL = 0` 即可退化为无 TTL 模式，配置永不过期，需要手动调用 `Deregister` 清理。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 [MIT License](../../LICENSE) 开源协议。
