# Util - GormLog

将 tiles `logger.Logger` 适配为 GORM `logger.Interface`，实现 GORM SQL 日志与项目日志系统的无缝集成。

## 特性

- **适配器模式**：包装任意 tiles `logger.Logger` 实现，无需修改日志配置
- **慢查询告警**：超过阈值（默认 200ms）的 SQL 以 Warn 级别输出
- **错误忽略**：`gorm.ErrRecordNotFound` 不触发错误日志，避免噪音
- **结构化字段**：每条 SQL 记录 `elapsed`、`rows_affected`、`sql` 字段
- **日志级别控制**：通过 `WithLogLevel` 或 GORM 的 `LogMode` 动态调整

## 快速开始

```go
import (
    "time"

    zapLogger "github.com/CXeon/tiles/logger/zap"
    "github.com/CXeon/tiles/util/gormlog"
    "github.com/CXeon/tiles/db/gormdb"
)

// 1. 创建 tiles logger
log := zapLogger.NewLogger(zapLogger.Config{
    EnableStdout: true,
    Level:        "info",
})

// 2. 创建 GORM 日志适配器
gormLog := gormlog.New(log,
    gormlog.WithSlowThreshold(200 * time.Millisecond), // 慢查询阈值
)

// 3. 注入到 gormdb
client, err := gormdb.New(cfg,
    gormdb.WithGormLogger(gormLog),
)
```

## 配置选项

| 选项函数 | 默认值 | 说明 |
|---------|--------|------|
| `WithLogLevel(l)` | `gormlogger.Info` | GORM 初始日志级别 |
| `WithSlowThreshold(d)` | `200ms` | 慢查询告警阈值 |

### GORM 日志级别

| 常量 | 值 | 说明 |
|------|----|------|
| `gormlogger.Silent` | 1 | 关闭所有日志 |
| `gormlogger.Error` | 2 | 只输出错误 |
| `gormlogger.Warn` | 3 | 错误 + 慢查询 |
| `gormlogger.Info` | 4 | 全部 SQL（含普通查询） |

## 日志输出示例

### 普通 SQL（Info 级别）

```json
{
  "level": "info",
  "msg": "gorm trace",
  "elapsed": "1.234ms",
  "rows_affected": 1,
  "sql": "INSERT INTO `users` ..."
}
```

### 慢查询（Warn 级别）

```json
{
  "level": "warn",
  "msg": "gorm slow query",
  "elapsed": "523.4ms",
  "rows_affected": 1000,
  "sql": "SELECT * FROM `orders` WHERE ...",
  "slow_threshold": "200ms"
}
```

### SQL 错误（Error 级别）

```json
{
  "level": "error",
  "msg": "gorm trace",
  "err": "Error 1062: Duplicate entry '...' for key 'PRIMARY'",
  "elapsed": "2.1ms",
  "rows_affected": 0,
  "sql": "INSERT INTO `users` ..."
}
```

## 运行测试

```bash
go test github.com/CXeon/tiles/util/gormlog/... -v
```

## 相关链接

- [db/gormdb](../../db/gormdb/README.md)
- [GORM Logger 接口文档](https://gorm.io/docs/logger.html)
- [tiles 项目主页](../../README.md)
