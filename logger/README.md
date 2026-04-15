# Logger（日志模块）

提供统一的日志抽象接口，屏蔽底层日志库差异，支持多种主流日志库实现。

## 接口定义

```go
type Fields map[string]any

type Logger interface {
    Debug(msg string, fields Fields)
    Info(msg string, fields Fields)
    Warn(msg string, fields Fields)
    Error(msg string, err error, fields Fields)
}
```

## 快速开始

以 Zap 实现为例（其他实现接口相同）：

```go
import (
    baseLogger "github.com/CXeon/tiles/logger"
    zapLogger "github.com/CXeon/tiles/logger/zap"
)

log := zapLogger.NewLogger(zapLogger.Config{
    Level:        "info",
    EnableStdout: true,
})

log.Info("server started", baseLogger.Fields{
    "addr": ":8080",
    "env":  "production",
})

log.Error("database connection failed", err, baseLogger.Fields{
    "host": "localhost",
    "port": 5432,
})
```

## 可用实现

| 实现 | 包路径 | 特点 | 文档 |
|------|--------|------|------|
| **Zap** | `github.com/CXeon/tiles/logger/zap` | 高性能，适合高并发场景 | [文档](zap/README.md) |
| **Logrus** | `github.com/CXeon/tiles/logger/logrus` | 社区流行，生态成熟 | [文档](logrus/README.md) |
| **Slog** | `github.com/CXeon/tiles/logger/slog` | Go 1.21+ 官方标准库，零第三方依赖 | [文档](slog/README.md) |

## 选择建议

- **新项目推荐 Slog**：Go 1.21+ 官方方案，零依赖，未来有更多官方支持
- **高性能场景选 Zap**：对日志吞吐量有严格要求时
- **已有 Logrus 项目**：保持一致性，无需迁移

## 设计原则

- 所有实现使用 `"err"` 作为错误字段名，便于日志检索和告警规则统一
- `Fields` 类型为 `map[string]any`，灵活支持各种字段类型
- 统一的 `Config` 结构（各实现）涵盖文件路径、日志级别、轮转策略和标准输出开关

## 相关链接

- [Zap 实现](zap/README.md)
- [Logrus 实现](logrus/README.md)
- [Slog 实现](slog/README.md)
- [tiles 项目主页](../README.md)
