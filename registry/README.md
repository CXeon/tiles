# Registry（服务注册中心模块）

提供服务注册与发现的抽象接口，支持多维度流量隔离、实时监听和负载均衡。

## 接口定义

```go
type Client interface {
    // Register 注册服务实例（同时设置隔离上下文）
    Register(ctx context.Context, endpoint *Endpoint) error

    // Deregister 注销服务实例
    Deregister(ctx context.Context, endpoint *Endpoint) error

    // Discover 一次性拉取指定服务的实例列表
    Discover(ctx context.Context, service []string, option ...ServiceOption) (CompanyRegistry, error)

    // Watch 订阅指定服务的实例变更，后台并发监听
    Watch(ctx context.Context, service []string, option ...ServiceOption) error

    // GetService 从本地缓存获取服务实例（自动应用负载均衡策略）
    GetService(ctx context.Context, service string, option ...GetServiceOption) (Endpoint, error)

    // Close 关闭连接，释放资源
    Close(ctx context.Context) error
}
```

## Endpoint 结构

```go
type Endpoint struct {
    InstanceID string            // 实例 ID，全局唯一（留空时自动生成）
    Env        string            // 环境：dev / test / prod
    Cluster    string            // 集群：china / america / europe
    Company    string            // 公司名称
    Project    string            // 项目名称
    Service    string            // 服务名称
    Protocol   ProtocolType      // 通信协议：http
    Color      string            // 染色标识：blue / green / gray
    Ip         string            // 实例 IP
    Port       uint16            // 实例端口
    Extra      map[string]string // 额外元数据
    Weight     uint16            // 实例权重，0 表示不设置
}
```

## 服务发现结果

```go
// 三层嵌套结构：Company -> Project -> Service -> 实例列表+负载均衡器
type CompanyRegistry  map[string]ProjectRegistry
type ProjectRegistry  map[string]ServiceRegistry
type ServiceRegistry  map[string]EndpointsWithLoadBalancer

type EndpointsWithLoadBalancer struct {
    Endpoints    []Endpoint
    LoadBalancer LoadBalancer
}
```

## 快速开始

```go
import (
    "github.com/CXeon/tiles/registry"
    registryEtcd "github.com/CXeon/tiles/registry/etcd"
)

client, err := registryEtcd.NewRegistry(registryEtcd.Config{
    Endpoints:            []string{"localhost:2379"},
    DialTimeout:          5 * time.Second,
    LoadBalancerStrategy: 0, // 0=RoundRobin, 1=Random, 2=WeightedRandom
})
if err != nil {
    panic(err)
}
defer client.Close(ctx)

// 注册服务（同时设置隔离上下文）
endpoint := &registry.Endpoint{
    Env: "prod", Cluster: "china",
    Company: "mycompany", Project: "myproject",
    Service: "user-service",
    Protocol: registry.ProtocolTypeHttp,
    Ip: "192.168.1.100", Port: 8080,
}
client.Register(ctx, endpoint)

// 订阅服务变更（后台自动更新本地缓存）
client.Watch(ctx, []string{"user-service", "order-service"})

// 获取服务实例（自动负载均衡）
ep, err := client.GetService(ctx, "user-service")
fmt.Printf("selected: %s:%d\n", ep.Ip, ep.Port)
```

## 标准错误

| 变量 | 说明 |
|------|------|
| `ErrServiceNotFound` | 服务未在缓存中找到 |
| `ErrNotRegistered` | 服务未注册 |
| `ErrInvalidEndpoint` | 端点无效或为 nil |
| `ErrEmptyService` | 服务名为空 |
| `ErrEmptyServices` | 服务列表为空 |
| `ErrCurrentEndpointNil` | 未注册，无法确定隔离上下文 |

## 可用实现

| 实现 | 包路径 | 特点 | 文档 |
|------|--------|------|------|
| **Etcd** | `github.com/CXeon/tiles/registry/etcd` | 基于 etcd lease 自动续约，支持三种负载均衡策略 | [文档](etcd/README.md) |

## 相关链接

- [Etcd 实现](etcd/README.md)
- [tiles 项目主页](../README.md)
