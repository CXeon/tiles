# Util - IP

提供获取本机有效 IPv4 地址的工具函数，跨平台支持 Linux、macOS 和 Windows。

## 功能

```go
func GetLocalIP() (string, error)
```

枚举网络接口，返回第一个满足以下条件的 IPv4 地址：
- 接口处于 Up 状态
- 非回环地址（排除 `127.x.x.x`）
- 非 Docker bridge 地址（排除 `172.17.x.x`）

## 快速开始

```go
import "github.com/CXeon/tiles/util/ip"

localIP, err := ip.GetLocalIP()
if err != nil {
    log.Fatal("failed to get local IP:", err)
}
fmt.Println("Local IP:", localIP)
// Output: Local IP: 192.168.1.100
```

## 使用场景

服务注册时自动填充实例 IP，无需手动配置：

```go
import (
    "github.com/CXeon/tiles/registry"
    "github.com/CXeon/tiles/util/ip"
)

localIP, err := ip.GetLocalIP()
if err != nil {
    panic(err)
}

endpoint := &registry.Endpoint{
    Ip:      localIP,
    Port:    8080,
    Service: "user-service",
    // ...
}
client.Register(ctx, endpoint)
```

## 运行测试

```bash
go test github.com/CXeon/tiles/util/ip/... -v
```

## 相关链接

- [tiles 项目主页](../../README.md)
