# Gateway（网关模块）

提供服务网关的抽象接口，屏蔽底层网关实现差异，统一服务实例的注册、注销、更新和心跳续约操作。

## 接口定义

```go
type Client interface {
    // Register 将服务实例注册到网关
    Register(ctx context.Context, endpoint *Endpoint) error

    // Deregister 从网关撤销服务实例
    Deregister(ctx context.Context, endpoint *Endpoint) error

    // Update 更新服务实例信息
    Update(ctx context.Context, endpoint *Endpoint) error

    // KeepAlive 手动续约服务实例配置（支持 TTL 机制的实现使用）
    KeepAlive(ctx context.Context, endpoint *Endpoint) error

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
    TTL        uint32            // 生存时间（秒），0 表示永不过期
    Weight     uint16            // 实例权重，0 表示不设置
}
```

## 快速开始

以 Traefik 实现为例：

```go
import (
    "github.com/CXeon/tiles/gateway"
    "github.com/CXeon/tiles/gateway/traefik"
)

// 创建网关客户端
client, err := traefik.NewClient(ctx, &traefik.Provider{
    Type:      "redis",
    Endpoints: []string{"localhost:6379"},
})
if err != nil {
    panic(err)
}
defer client.Close(ctx)

// 定义服务实例
endpoint := &gateway.Endpoint{
    Env:      "prod",
    Cluster:  "china",
    Company:  "mycompany",
    Project:  "myproject",
    Service:  "user-service",
    Protocol: gateway.ProtocolTypeHttp,
    Ip:       "192.168.1.100",
    Port:     8080,
    TTL:      30,   // 30 秒 TTL，自动续约
    Weight:   100,
}

// 注册
err = client.Register(ctx, endpoint)

// 停止时注销
defer client.Deregister(ctx, endpoint)
```

## 可用实现

| 实现 | 包路径 | 特点 | 文档 |
|------|--------|------|------|
| **Traefik** | `github.com/CXeon/tiles/gateway/traefik` | 基于 Traefik 3.x + KV Store，支持 Redis/Consul/Etcd/ZooKeeper | [文档](traefik/README.md) |

## 相关链接

- [Traefik 实现](traefik/README.md)
- [tiles 项目主页](../README.md)
