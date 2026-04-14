# DB - GormDB

基于 [GORM](https://gorm.io) 封装的数据库客户端，统一连接管理与连接池配置，支持 MySQL、PostgreSQL、SQLite。

## 特性

- **多驱动支持**：MySQL、PostgreSQL、SQLite（基于 glebarez/sqlite，纯 Go 实现，无 CGO）
- **开箱即用的连接池**：合理默认值（最大 100 连接，空闲 10，连接复用 30 分钟）
- **函数式选项**：通过 `Option` 灵活覆盖连接池、字符集、SSL 等参数
- **GORM Logger 集成**：注入自定义 Logger，与项目日志系统对接
- **驱动专属选项**：MySQL 和 PostgreSQL 各有独立配置项

## 快速开始

### MySQL

```go
import "github.com/CXeon/tiles/db/gormdb"

client, err := gormdb.New(gormdb.Config{
    Driver:   gormdb.DriverMySQL,
    Host:     "localhost",
    Port:     3306,
    Username: "root",
    Password: "password",
    Database: "mydb",
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

db := client.GetDB()
db.Create(&User{Name: "alice"})
```

### PostgreSQL

```go
client, err := gormdb.New(gormdb.Config{
    Driver:   gormdb.DriverPostgreSQL,
    Host:     "localhost",
    Port:     5432,
    Username: "postgres",
    Password: "password",
    Database: "mydb",
})
```

### SQLite（开发/测试）

```go
// 持久化文件
client, err := gormdb.New(gormdb.Config{
    Driver:   gormdb.DriverSQLite,
    Database: "app.db",
})

// 内存模式（测试用）
client, err := gormdb.New(gormdb.Config{
    Driver:   gormdb.DriverSQLite,
    Database: ":memory:",
})
```

## 配置说明

### Config 结构体

```go
type Config struct {
    Driver   string  // "mysql" | "postgresql" | "sqlite"
    Host     string  // 数据库主机（SQLite 不需要）
    Port     int     // 数据库端口（SQLite 不需要）
    Username string  // 用户名（SQLite 不需要）
    Password string  // 密码（SQLite 不需要）
    Database string  // 数据库名；SQLite 时为文件路径或 ":memory:"
}
```

### 连接池选项（默认值）

| 选项函数 | 默认值 | 说明 |
|---------|--------|------|
| `WithMaxOpenConns(n)` | 100 | 最大打开连接数 |
| `WithMaxIdleConns(n)` | 10 | 最大空闲连接数 |
| `WithConnMaxLifetime(d)` | 30 分钟 | 连接最大复用时长 |
| `WithConnMaxIdleTime(d)` | 10 分钟 | 连接最大空闲时长 |

### 通用选项

| 选项函数 | 默认值 | 说明 |
|---------|--------|------|
| `WithCharset(s)` | `utf8mb4` | 字符集（MySQL） |
| `WithParseTime(b)` | `true` | 自动解析时间类型（MySQL） |
| `WithLoc(s)` | `Local` | 时区 |
| `WithGormLogger(l)` | nil | 注入自定义 GORM Logger |

### MySQL 专属选项

| 选项函数 | 说明 |
|---------|------|
| `WithMysqlDefaultStringSize(n)` | string 字段默认长度，默认 256 |
| `WithMysqlDisableDatetimePrecision(b)` | 禁用 datetime 精度（MySQL 5.6 之前） |
| `WithMysqlDontSupportRenameIndex(b)` | 使用删除+重建方式重命名索引 |
| `WithMysqlDontSupportRenameColumn(b)` | 使用 change 语法重命名列 |
| `WithMysqlSkipInitializeWithVersion(b)` | 跳过自动版本检测 |

### PostgreSQL 专属选项

| 选项函数 | 默认值 | 说明 |
|---------|--------|------|
| `WithPostgresqlSSLMode(s)` | `disable` | SSL 模式 |
| `WithPostgresqlTimeZone(s)` | `Asia/Shanghai` | 时区 |
| `WithPostgresqlPreferSimpleProtocol(b)` | false | 禁用 Prepared Statement 缓存 |

## 使用场景

### 与项目日志系统集成

搭配 `util/gormlog` 将 tiles logger 适配为 GORM Logger：

```go
import (
    "github.com/CXeon/tiles/db/gormdb"
    "github.com/CXeon/tiles/util/gormlog"
    zapLogger "github.com/CXeon/tiles/logger/zap"
)

log := zapLogger.NewLogger(zapLogger.Config{EnableStdout: true, Level: "info"})

gormLog := gormlog.New(log,
    gormlog.WithSlowThreshold(200*time.Millisecond),
)

client, err := gormdb.New(cfg,
    gormdb.WithGormLogger(gormLog),
)
```

### 自定义连接池

```go
client, err := gormdb.New(cfg,
    gormdb.WithMaxOpenConns(50),
    gormdb.WithMaxIdleConns(5),
    gormdb.WithConnMaxLifetime(15*time.Minute),
)
```

## Client 方法

| 方法 | 说明 |
|------|------|
| `GetDB() *gorm.DB` | 返回底层 `*gorm.DB`，执行 ORM 操作 |
| `Pool() (*sql.DB, error)` | 返回底层 `*sql.DB`，直接操作连接池 |
| `Close() error` | 关闭连接，释放连接池全部资源 |

## 运行测试

```bash
go test github.com/CXeon/tiles/db/gormdb/... -v
```

## 相关链接

- [GORM 官方文档](https://gorm.io/docs)
- [gormlog 工具](../util/gormlog/README.md)
- [tiles 项目主页](../../README.md)
