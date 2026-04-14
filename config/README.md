# Config（配置模块）

提供统一的配置加载与热更新抽象接口，屏蔽本地文件与远程配置中心的差异。

## 接口定义

```go
type Config interface {
    // Load 从配置源加载配置（必须首先调用）
    Load() error

    // Get 系列方法通过 key 读取配置值（支持 "." 分隔的嵌套路径）
    Get(key string) any
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    GetFloat64(key string) float64

    // Unmarshal 将整个配置反序列化到结构体（需 mapstructure tag）
    Unmarshal(v any) error
    // UnmarshalKey 将指定 key 下的子配置反序列化到结构体
    UnmarshalKey(key string, v any) error

    // Watch 注册配置变更回调，Load 后可调用
    Watch(handler ChangeHandler) error

    // Close 停止监听并释放资源
    Close(ctx context.Context) error
}
```

## 变更事件

```go
type ChangeEvent struct {
    Changes []Change  // 精确变更列表，为空表示整体重载
}

type Change struct {
    Key        string
    OldValue   any
    NewValue   any
    ChangeType ChangeType  // "added" | "modified" | "deleted"
}
```

## 快速开始

以 Viper 实现为例：

```go
import (
    baseConfig "github.com/CXeon/tiles/config"
    viperConfig "github.com/CXeon/tiles/config/viper"
)

cfg := viperConfig.New(viperConfig.Config{
    ConfigPaths: []string{"./configs"},
    ConfigName:  "app",
    ConfigType:  "yaml",
})

if err := cfg.Load(); err != nil {
    log.Fatal(err)
}

// 读取配置
host := cfg.GetString("database.host")
port := cfg.GetInt("database.port")

// 反序列化到结构体
type DBConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
}
var dbCfg DBConfig
cfg.UnmarshalKey("database", &dbCfg)

// 监听变更
cfg.Watch(func(event *baseConfig.ChangeEvent) {
    for _, change := range event.Changes {
        log.Printf("[%s] %s: %v -> %v", change.ChangeType, change.Key, change.OldValue, change.NewValue)
    }
})

defer cfg.Close(context.Background())
```

## 可用实现

| 实现 | 包路径 | 特点 | 文档 |
|------|--------|------|------|
| **Viper** | `github.com/CXeon/tiles/config/viper` | 本地文件（YAML/JSON/TOML）+ 环境变量，基于 fsnotify 热更新 | [文档](viper/README.md) |
| **Apollo** | `github.com/CXeon/tiles/config/apollo` | 远程 Apollo 配置中心，服务端主动推送，支持本地备份容灾 | [文档](apollo/README.md) |

## 错误变量

| 变量 | 说明 |
|------|------|
| `ErrNotLoaded` | 未调用 `Load()` 就读取配置 |
| `ErrKeyNotFound` | 指定 key 不存在 |

## 相关链接

- [Viper 实现](viper/README.md)
- [Apollo 实现](apollo/README.md)
- [tiles 项目主页](../README.md)
