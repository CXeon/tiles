# Config - Viper Implementation

基于 [spf13/viper](https://github.com/spf13/viper) 的本地配置文件实现。

## 特性

- **多格式支持**：支持 YAML、JSON、TOML、ENV 等格式的配置文件
- **环境变量绑定**：支持前缀过滤的环境变量自动绑定
- **文件热更新**：基于 fsnotify 监听文件变更，自动触发回调
- **变更追踪**：通过新旧快照对比，精确报告新增、修改、删除的配置项
- **并发安全**：使用 `sync.RWMutex` 保护 handler 列表和快照

## 安装

```bash
go get github.com/CXeon/tiles/config/viper
```

## 快速开始

```go
import (
    baseConfig "github.com/CXeon/tiles/config"
    viperConfig "github.com/CXeon/tiles/config/viper"
)

// 创建配置实例
cfg := viperConfig.New(viperConfig.Config{
    ConfigPaths: []string{".", "./configs"},
    ConfigName:  "config",
    ConfigType:  "yaml",
    EnvPrefix:   "APP",
    AutoEnv:     true,
})

// 加载配置
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

// 监听配置变更
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
    // ConfigPaths 是配置文件搜索路径，默认为 ["."]
    ConfigPaths []string

    // ConfigName 是配置文件名（不含扩展名），如 "config"
    // 对应文件：config.yaml、config.json 等
    ConfigName string

    // ConfigType 是文件类型："yaml"、"json"、"toml"、"env" 等，默认 "yaml"
    ConfigType string

    // EnvPrefix 是环境变量前缀
    // 如设置 "APP"，则读取 APP_DATABASE_HOST 对应 key "database.host"
    EnvPrefix string

    // AutoEnv 是否自动绑定所有环境变量，默认 true
    AutoEnv bool
}
```

## 使用场景

### 纯文件模式

```go
cfg := viperConfig.New(viperConfig.Config{
    ConfigPaths: []string{"./configs"},
    ConfigName:  "app",
    ConfigType:  "yaml",
})
cfg.Load()
```

### 文件 + 环境变量叠加

环境变量会覆盖文件中的同名配置（环境变量优先级更高）：

```go
cfg := viperConfig.New(viperConfig.Config{
    ConfigPaths: []string{"./configs"},
    ConfigName:  "app",
    ConfigType:  "yaml",
    EnvPrefix:   "APP",  // APP_DATABASE_HOST 覆盖 database.host
    AutoEnv:     true,
})
cfg.Load()
```

### Watch 热更新

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
cd config/viper
go test -v
```

## 相关链接

- [spf13/viper](https://github.com/spf13/viper)
- [Config 接口定义](../config.go)
