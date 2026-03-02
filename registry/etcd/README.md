# Registry - Etcd Implementation

基于 [etcd](https://etcd.io/) 的服务注册中心实现。

## 特性

- **服务注册与发现**：支持服务实例的注册、注销和发现
- **实时监听(Watch)**：并发监听多个服务的实例变更，自动更新本地缓存
- **多种负载均衡策略**：支持 RoundRobin、Random、WeightedRandom 三种策略
- **多维度流量隔离**：支持环境、集群、公司、项目、染色等维度的隔离
- **状态保持**：LoadBalancer 实例在服务更新时保持状态，确保 RoundRobin 等策略的连续性
- **自动续约**：通过 etcd lease 机制实现服务实例的自动续约

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "time"
    "github.com/CXeon/tiles/registry"
    registryEtcd "github.com/CXeon/tiles/registry/etcd"
)

func main() {
    ctx := context.Background()

    // 创建 Registry Client
    cfg := registryEtcd.Config{
        Endpoints:            []string{"localhost:2379"},
        Username:             "",
        Password:             "",
        DialTimeout:          5 * time.Second,
        LoadBalancerStrategy: 0, // 0=RoundRobin, 1=Random, 2=WeightedRandom
    }
    
    client, err := registryEtcd.NewRegistry(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close(ctx)

    // 定义服务实例信息
    endpoint := &registry.Endpoint{
        Env:      "prod",
        Cluster:  "china",
        Company:  "mycompany",
        Project:  "myproject",
        Service:  "user-service",
        Protocol: registry.ProtocolTypeHttp,
        Ip:       "192.168.1.100",
        Port:     8080,
        Color:    "",      // 可选：染色标识
        Weight:   100,     // 可选：权重，默认为0(等同于1)
    }

    // 注册服务实例
    err = client.Register(ctx, endpoint)
    if err != nil {
        panic(err)
    }

    // 运行服务...
    // etcd 会自动续约 lease

    // 服务停止时注销
    err = client.Deregister(ctx, endpoint)
    if err != nil {
        panic(err)
    }
}
```

## 配置说明

### Config 结构体

```go
type Config struct {
    // Endpoints 是 etcd 集群的端点列表
    Endpoints []string
    
    // Username 是 etcd 认证用户名（可选）
    Username string
    
    // Password 是 etcd 认证密码（可选）
    Password string
    
    // DialTimeout 是连接超时时间
    DialTimeout time.Duration
    
    // LoadBalancerStrategy 是负载均衡策略
    // 0 = RoundRobin (默认)
    // 1 = Random
    // 2 = WeightedRandom
    LoadBalancerStrategy uint8
}
```

## 使用场景

### 场景1: 服务注册

```go
endpoint := &registry.Endpoint{
    Env:      "prod",
    Cluster:  "china",
    Company:  "mycompany",
    Project:  "myproject",
    Service:  "order-service",
    Protocol: registry.ProtocolTypeHttp,
    Ip:       "192.168.1.101",
    Port:     8081,
    Weight:   100,
}

err := client.Register(ctx, endpoint)
if err != nil {
    log.Fatal(err)
}
```

### 场景2: 服务发现

```go
// 发现当前公司/项目下的服务实例
services := []string{"user-service", "order-service"}
result, err := client.Discover(ctx, services)
if err != nil {
    log.Fatal(err)
}

// result 的结构: map[company]map[project]map[service]EndpointsWithLoadBalancer
for company, projects := range result {
    for project, srvcs := range projects {
        for service, data := range srvcs {
            fmt.Printf("Company: %s, Project: %s, Service: %s\n", company, project, service)
            fmt.Printf("Instances: %d\n", len(data.Endpoints))
        }
    }
}
```

### 场景3: 跨公司/项目服务发现

```go
// 发现指定公司/项目的服务实例
services := []string{"payment-service"}
result, err := client.Discover(ctx, services,
    registry.WithGetOptComProj(map[string][]string{
        "companyA": {"projectX", "projectY"},
        "companyB": {"projectZ"},
    }),
)
if err != nil {
    log.Fatal(err)
}
```

### 场景4: 实时监听服务变更 (Watch)

```go
// 监听多个服务的实例变更
services := []string{"user-service", "order-service", "payment-service"}
err := client.Watch(ctx, services)
if err != nil {
    log.Fatal(err)
}

// Watch 会在后台并发监听所有服务
// 每个服务的变更会自动更新到本地缓存
// 无需额外处理

// 获取服务实例（自动负载均衡）
endpoint, err := client.GetService(ctx, "user-service")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Selected instance: %s:%d\n", endpoint.Ip, endpoint.Port)
```

### 场景5: 负载均衡选择

```go
// 通过 GetService 获取服务实例时，会自动应用配置的负载均衡策略

// 使用 RoundRobin 策略 (LoadBalancerStrategy = 0)
endpoint1, _ := client.GetService(ctx, "user-service")  // 返回实例A
endpoint2, _ := client.GetService(ctx, "user-service")  // 返回实例B
endpoint3, _ := client.GetService(ctx, "user-service")  // 返回实例C
endpoint4, _ := client.GetService(ctx, "user-service")  // 返回实例A (循环)

// 使用 WeightedRandom 策略 (LoadBalancerStrategy = 2)
// 权重高的实例被选中的概率更大
```

## 负载均衡策略

### RoundRobin (策略 0)

轮询选择实例，按顺序依次返回。

**特点**:
- 请求均匀分布
- 有状态：记录当前选择的索引
- Watch 更新时保持状态连续性

### Random (策略 1)

随机选择实例。

**特点**:
- 无状态
- 简单快速
- 适合无差异的实例

### WeightedRandom (策略 2)

根据权重进行加权随机选择。

**特点**:
- 权重越高，被选中概率越大
- 权重为 0 的实例被视为权重 1
- 适合实例性能有差异的场景

**示例**:
```go
// 实例A权重100，实例B权重200
// 实例B被选中的概率是实例A的2倍
endpoint1 := &registry.Endpoint{
    Weight: 100,
    // ...
}
endpoint2 := &registry.Endpoint{
    Weight: 200,
    // ...
}
```

## 流量隔离

Registry 支持多维度的流量隔离：

### 环境隔离 (Env)

```go
endpoint := &registry.Endpoint{
    Env: "prod",  // 或 "dev", "test", "staging"
    // ...
}
```

不同环境的服务实例相互隔离，不会互相发现。

### 集群隔离 (Cluster)

```go
endpoint := &registry.Endpoint{
    Cluster: "china",  // 或 "america", "europe"
    // ...
}
```

不同集群的服务实例相互隔离。

### 公司/项目隔离 (Company/Project)

```go
endpoint := &registry.Endpoint{
    Company: "mycompany",
    Project: "myproject",
    // ...
}
```

默认情况下，服务发现只查询当前公司/项目的实例。可以通过 `WithGetOptComProj` 选项跨公司/项目查询。

### 协议隔离 (Protocol)

```go
endpoint := &registry.Endpoint{
    Protocol: registry.ProtocolTypeHttp,  // 或 ProtocolTypeHttps
    // ...
}
```

**注意**: Https 协议在存储时会被规范化为 Http。

### 染色隔离 (Color)

```go
endpoint := &registry.Endpoint{
    Color: "blue",  // 或 "green", "gray"
    // ...
}
```

支持蓝绿部署、灰度发布等场景。

## 标准化错误处理

Registry 定义了标准错误变量，便于错误类型判断：

```go
import (
    "errors"
    "github.com/CXeon/tiles/registry"
)

endpoint, err := client.GetService(ctx, "user-service")
if err != nil {
    if errors.Is(err, registry.ErrServiceNotFound) {
        // 服务未找到，可能还未注册
        log.Println("Service not found")
    } else if errors.Is(err, registry.ErrEmptyService) {
        // 服务名为空
        log.Println("Empty service name")
    } else {
        // 其他错误
        log.Fatal(err)
    }
}
```

**可用的错误变量**:
- `ErrServiceNotFound` - 服务未在缓存中找到
- `ErrNotRegistered` - 服务未注册
- `ErrInvalidEndpoint` - 端点无效或为 nil
- `ErrEmptyService` - 服务名为空
- `ErrEmptyServices` - 服务列表为空
- `ErrEmptyCompanyOrProject` - 公司或项目名为空
- `ErrCurrentEndpointNil` - 当前端点为 nil，无法确定隔离上下文
- `ErrComProjEmpty` - ComProj 参数为空

## 常见问题

### Q1: 为什么需要先 Register 才能 Discover/Watch?

**A**: Register 会设置 `currentEndpoint`，作为隔离上下文的基准。Discover 和 Watch 默认使用当前端点的 Env/Cluster/Company/Project/Color 作为隔离维度。

### Q2: Watch 如何确保多个服务都被监听?

**A**: Watch 方法已重构为并发模式，为每个服务启动独立的 goroutine 进行监听，不会因为某个服务阻塞而影响其他服务。

### Q3: LoadBalancer 的状态会在服务更新时丢失吗?

**A**: 不会。优化后的实现会保持 LoadBalancer 实例，只更新 Endpoints 列表，确保 RoundRobin 等有状态策略的索引连续性。

### Q4: 如何选择负载均衡策略?

**A**: 
- **RoundRobin**: 默认选择，请求分布均匀，适合大多数场景
- **Random**: 简单快速，适合无状态、无差异的实例
- **WeightedRandom**: 适合实例性能有差异的场景，可根据权重分配流量

### Q5: etcd 连接断开后会发生什么?

**A**: 
- Register 的 lease 会失效，服务实例会自动从 etcd 删除
- Watch 会自动退出
- 需要重新 Register 和 Watch

## 运行测试

```bash
go test github.com/CXeon/tiles/registry/etcd/... -v
```

**注意**: 测试需要本地运行 etcd 实例。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 [MIT License](../../LICENSE) 开源协议。
