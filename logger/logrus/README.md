# Logger - Logrus Implementation

基于 [Logrus](https://github.com/sirupsen/logrus) 的社区流行日志实现。

## 特性

- **成熟稳定**：Logrus 是 Go 社区最流行的日志库之一
- **JSON 格式**：生产环境友好的 JSON 格式输出
- **日志轮转**：集成 lumberjack 支持自动日志轮转
- **灵活输出**：支持文件、stdout 或同时输出
- **丰富生态**：大量第三方 Hook 和格式化器

## 安装

```bash
go get github.com/CXeon/tiles/logger/logrus
```

## 快速开始

### 基本使用

```go
import (
    baseLogger "github.com/CXeon/tiles/logger"
    logrusLogger "github.com/CXeon/tiles/logger/logrus"
)

// 创建日志实例
cfg := logrusLogger.Config{
    Filename:     "/var/log/app.log",
    MaxSize:      100,        // MB
    MaxBackups:   7,          // 保留7个备份
    MaxAge:       30,         // 保留30天
    Compress:     true,       // 压缩旧日志
    Level:        "info",     // 日志级别
    EnableStdout: false,      // 不输出到控制台
}

log := logrusLogger.NewLogger(cfg)

// 使用日志
log.Info("server started", baseLogger.Fields{
    "addr": ":8080",
    "env":  "production",
})

log.Error("database connection failed", err, baseLogger.Fields{
    "host": "localhost",
    "port": 5432,
})
```

## 配置说明

### Config 结构体

```go
type Config struct {
    // Filename 是日志输出文件路径
    Filename string
    
    // MaxSize 是单个日志文件的最大大小（单位：MB）
    MaxSize int
    
    // MaxBackups 是最多保留的旧日志文件个数
    MaxBackups int
    
    // MaxAge 是最多保留的天数
    MaxAge int
    
    // Compress 表示是否压缩旧日志文件
    Compress bool
    
    // Level 是默认的日志级别
    // 支持: "debug", "info", "warn", "error", "fatal", "panic"
    Level string
    
    // EnableStdout 是否同时输出到标准输出
    EnableStdout bool
}
```

### 输出策略

| Filename | EnableStdout | 行为 |
|----------|--------------|------|
| ""       | true         | 只输出到 stdout |
| "app.log" | true        | 同时输出到文件和 stdout |
| "app.log" | false       | 只输出到文件 |
| ""       | false        | 默认输出到 stdout |

## 使用场景

### 开发环境（本地调试）

```go
cfg := logrusLogger.Config{
    Filename:     "dev.log",
    MaxSize:      10,
    EnableStdout: true,   // 同时输出到控制台和文件
    Level:        "debug",
}
```

### 生产环境（容器）

```go
cfg := logrusLogger.Config{
    Filename:     "",      // 不写文件
    EnableStdout: true,    // 只输出到 stdout，由容器平台收集
    Level:        "info",
}
```

### 生产环境（物理机）

```go
cfg := logrusLogger.Config{
    Filename:     "/var/log/app.log",
    MaxSize:      100,
    MaxBackups:   7,
    MaxAge:       30,
    Compress:     true,
    EnableStdout: false,   // 只写文件
    Level:        "info",
}
```

## 日志格式

### 输出示例

```json
{
  "level": "info",
  "msg": "user login",
  "time": "2026-01-29T12:34:56+08:00",
  "user_id": 123,
  "ip": "192.168.0.1"
}
```

### 错误日志

```json
{
  "level": "error",
  "msg": "create user failed",
  "time": "2026-01-29T12:34:56+08:00",
  "err": "insert user: duplicate key",
  "user_id": 123
}
```

## 运行测试

```bash
cd logger/logrus
go test -v
```

## 性能特点

- Logrus 在性能上略逊于 Zap，但在绝大多数业务场景下完全够用
- 社区生态成熟，有大量现成的 Hook 和插件
- 接口友好，易于上手

## 最佳实践

1. **生产环境建议关闭 debug 级别**，减少日志量
2. **容器环境推荐只输出到 stdout**，由日志平台统一收集
3. **非容器环境配置合理的轮转策略**，避免磁盘爆满
4. **错误日志统一使用 `err` 字段名**，便于日志检索和告警

## 相关链接

- [Logrus 官方文档](https://github.com/sirupsen/logrus)
- [Lumberjack 日志轮转](https://github.com/natefinch/lumberjack)
- [Logger 接口定义](../logger.go)
