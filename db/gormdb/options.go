package gormdb

import (
	"time"

	gormlogger "gorm.io/gorm/logger"
)

// Option 是数据库客户端的可选配置函数
type Option func(*options)

type options struct {
	// 通用选项
	charset   string
	parseTime bool
	loc       string

	// 连接池选项
	maxIdleConns    int
	maxOpenConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration

	// GORM 日志
	gormLogger gormlogger.Interface

	// MySQL 专属选项
	mysqlDefaultStringSize         uint
	mysqlDisableDatetimePrecision  bool
	mysqlDontSupportRenameIndex    bool
	mysqlDontSupportRenameColumn   bool
	mysqlSkipInitializeWithVersion bool

	// PostgreSQL 专属选项
	postgresqlSSLMode              string
	postgresqlTimeZone             string
	postgresqlPreferSimpleProtocol bool
}

func defaultOptions() *options {
	return &options{
		charset:            "utf8mb4",
		parseTime:          true,
		loc:                "Local",
		postgresqlSSLMode:  "disable",
		postgresqlTimeZone: "Asia/Shanghai",
		maxIdleConns:       10,
		maxOpenConns:       100,
		connMaxLifetime:    30 * time.Minute,
		connMaxIdleTime:    10 * time.Minute,
	}
}

// --- 通用选项 ---

// WithCharset 设置字符集，默认 utf8mb4
func WithCharset(charset string) Option {
	return func(o *options) {
		o.charset = charset
	}
}

// WithParseTime 设置是否解析时间类型，MySQL 默认 true
func WithParseTime(parseTime bool) Option {
	return func(o *options) {
		o.parseTime = parseTime
	}
}

// WithLoc 设置时区，默认 Local
func WithLoc(loc string) Option {
	return func(o *options) {
		o.loc = loc
	}
}

// --- 连接池选项 ---

// WithMaxIdleConns 设置最大空闲连接数
func WithMaxIdleConns(n int) Option {
	return func(o *options) {
		o.maxIdleConns = n
	}
}

// WithMaxOpenConns 设置最大打开连接数
func WithMaxOpenConns(n int) Option {
	return func(o *options) {
		o.maxOpenConns = n
	}
}

// WithConnMaxLifetime 设置连接最大复用时长
func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *options) {
		o.connMaxLifetime = d
	}
}

// WithConnMaxIdleTime 设置连接最大空闲时长
func WithConnMaxIdleTime(d time.Duration) Option {
	return func(o *options) {
		o.connMaxIdleTime = d
	}
}

// --- GORM 日志 ---

// WithGormLogger 注入自定义 GORM 日志实现
func WithGormLogger(l gormlogger.Interface) Option {
	return func(o *options) {
		o.gormLogger = l
	}
}

// --- MySQL 专属 ---

// WithMysqlDefaultStringSize 设置 string 类型字段的默认长度，默认 256
func WithMysqlDefaultStringSize(size uint) Option {
	return func(o *options) {
		o.mysqlDefaultStringSize = size
	}
}

// WithMysqlDisableDatetimePrecision 禁用 datetime 精度（MySQL 5.6 之前不支持）
func WithMysqlDisableDatetimePrecision(disable bool) Option {
	return func(o *options) {
		o.mysqlDisableDatetimePrecision = disable
	}
}

// WithMysqlDontSupportRenameIndex 使用删除+重建方式重命名索引（MySQL 5.7 之前、MariaDB 不支持重命名索引）
func WithMysqlDontSupportRenameIndex(dont bool) Option {
	return func(o *options) {
		o.mysqlDontSupportRenameIndex = dont
	}
}

// WithMysqlDontSupportRenameColumn 使用 change 语法重命名列（MySQL 8 之前、MariaDB 不支持 rename column）
func WithMysqlDontSupportRenameColumn(dont bool) Option {
	return func(o *options) {
		o.mysqlDontSupportRenameColumn = dont
	}
}

// WithMysqlSkipInitializeWithVersion 跳过根据 MySQL 版本自动配置
func WithMysqlSkipInitializeWithVersion(skip bool) Option {
	return func(o *options) {
		o.mysqlSkipInitializeWithVersion = skip
	}
}

// --- PostgreSQL 专属 ---

// WithPostgresqlSSLMode 设置 SSL 模式，默认 disable
func WithPostgresqlSSLMode(sslMode string) Option {
	return func(o *options) {
		o.postgresqlSSLMode = sslMode
	}
}

// WithPostgresqlTimeZone 设置时区，默认 Local
func WithPostgresqlTimeZone(tz string) Option {
	return func(o *options) {
		o.postgresqlTimeZone = tz
	}
}

// WithPostgresqlPreferSimpleProtocol 禁用 Prepared Statement 缓存
func WithPostgresqlPreferSimpleProtocol(prefer bool) Option {
	return func(o *options) {
		o.postgresqlPreferSimpleProtocol = prefer
	}
}
