# Config - Apollo Implementation

基于 [apolloconfig/agollo](https://github.com/apolloconfig/agollo) 的远程 Apollo 配置中心实现。

## 特性

- **远程集中配置管理**：从 Apollo 服务端拉取配置，统一管理多服务配置
- **主动推送**：Apollo 服务端变更后实时推送，无需轮询
- **本地备份容灾**：启用 `IsBackupConfig` 后，配置自动缓存到本地，服务端不可用时仍可正常启动
- **多命名空间支持**：通过 `NamespaceName` 指定不同命名空间
- **变更精确感知**：区分 added / modified / deleted 三种变更类型

## 安装

```bash
go get github.com/CXeon/tiles/config/apollo
```

## 快速开始

```go
import (
    baseConfig "github.com/CXeon/tiles/config"
    apolloConfig "github.com/CXeon/tiles/config/apollo"
)

// 创建配置实例
cfg := apolloConfig.New(apolloConfig.Config{
    AppID:          "my-service",
    Cluster:        "default",
    IP:             "http://apollo.example.com:8080",
    NamespaceName:  "application",
    Secret:         "your-secret-key",
    IsBackupConfig: true,
})

// 加载配置（连接 Apollo 并拉取配置）
if err := cfg.Load(); err != nil {
    log.Fatal(err)
}

// 读取配置
host := cfg.GetString("database.host")
port := cfg.GetInt("database.port")

// 反序列化到结构体
type DatabaseConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
}
var dbCfg DatabaseConfig
cfg.UnmarshalKey("database", &dbCfg)

// 监听配置变更（Apollo 服务端推送时触发）
cfg.Watch(func(event *baseConfig.ChangeEvent) {
    for _, change := range event.Changes {
        log.Printf("config changed: key=%s type=%s", change.Key, change.ChangeType)
    }
})

// 释放资源
defer cfg.Close(context.Background())
```

## 配置说明

### Config 结构体

```go
type Config struct {
    // AppID 是 Apollo App ID（必填）
    AppID string

    // Cluster 是集群名称，如 "default"、"dev"、"prod"（必填）
    Cluster string

    // IP 是 Apollo Config Service 地址（必填）
    // 格式：http://apollo-host:8080
    IP string

    // NamespaceName 是命名空间，默认 "application"
    NamespaceName string

    // Secret 是 Apollo 认证密钥（可选，开启访问控制时必填）
    Secret string

    // IsBackupConfig 是否启用本地备份容灾，默认 true
    // 开启后配置会缓存在本地，Apollo 不可用时服务仍可正常启动
    IsBackupConfig bool
}
```

## 使用场景

### 开发环境

```go
cfg := apolloConfig.New(apolloConfig.Config{
    AppID:   "my-service",
    Cluster: "dev",
    IP:      "http://apollo-dev.company.com:8080",
})
```

### 生产环境多集群

```go
// 通过环境变量注入集群信息
cluster := os.Getenv("APOLLO_CLUSTER")  // e.g. "cn-north-1"
cfg := apolloConfig.New(apolloConfig.Config{
    AppID:          "my-service",
    Cluster:        cluster,
    IP:             "http://apollo.company.com:8080",
    Secret:         os.Getenv("APOLLO_SECRET"),
    IsBackupConfig: true,
})
```

### 配置热更新

```go
cfg.Watch(func(event *baseConfig.ChangeEvent) {
    for _, change := range event.Changes {
        switch change.ChangeType {
        case baseConfig.ChangeTypeAdded:
            log.Printf("new config: %s = %v", change.Key, change.NewValue)
        case baseConfig.ChangeTypeModified:
            log.Printf("config updated: %s: %v -> %v", change.Key, change.OldValue, change.NewValue)
        case baseConfig.ChangeTypeDeleted:
            log.Printf("config deleted: %s", change.Key)
        }
    }
})
```

## 运行测试

```bash
cd config/apollo
go test -v
```

> 注意：测试需要访问可用的 Apollo 服务端，建议在集成测试环境中运行。

## 相关链接

- [apolloconfig/agollo](https://github.com/apolloconfig/agollo)
- [Apollo 官方文档](https://www.apolloconfig.com)
- [Config 接口定义](../config.go)
