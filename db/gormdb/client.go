package gormdb

import (
	"database/sql"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	gormlib "gorm.io/gorm"
)

const (
	DriverMySQL      = "mysql"
	DriverPostgreSQL = "postgresql"
	DriverSQLite     = "sqlite"
)

// Config 是数据库连接的基础配置
type Config struct {
	// Driver 指定数据库类型："mysql" | "postgresql" | "sqlite"
	Driver string
	// Host 数据库主机地址（SQLite 不需要）
	Host string
	// Port 数据库端口（SQLite 不需要）
	Port int
	// Username 数据库用户名（SQLite 不需要）
	Username string
	// Password 数据库密码（SQLite 不需要）
	Password string
	// Database 数据库名称；SQLite 时为文件路径，如 "app.db" 或 ":memory:"
	Database string
}

// Client 是基于 GORM 的数据库客户端
type Client struct {
	db *gormlib.DB
}

// New 根据 Config 和可选 Option 创建数据库客户端
func New(cfg Config, opts ...Option) (*Client, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	gormCfg := &gormlib.Config{}
	if o.gormLogger != nil {
		gormCfg.Logger = o.gormLogger
	}

	var (
		db  *gormlib.DB
		err error
	)

	switch cfg.Driver {
	case DriverMySQL:
		db, err = openMySQL(cfg, o, gormCfg)
	case DriverPostgreSQL:
		db, err = openPostgreSQL(cfg, o, gormCfg)
	case DriverSQLite:
		db, err = openSQLite(cfg, gormCfg)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", cfg.Driver)
	}

	if err != nil {
		return nil, err
	}

	if err = applyPoolOptions(db, o); err != nil {
		return nil, err
	}

	return &Client{db: db}, nil
}

// GetDB 返回底层 *gorm.DB，用于执行 ORM 操作
func (c *Client) GetDB() *gormlib.DB {
	return c.db
}

// Pool 返回底层 *sql.DB，用于配置或直接使用连接池
func (c *Client) Pool() (*sql.DB, error) {
	return c.db.DB()
}

// Close 关闭数据库连接，释放连接池全部资源
func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func openMySQL(cfg Config, o *options, gormCfg *gormlib.Config) (*gormlib.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
		o.charset, o.parseTime, o.loc,
	)
	return gormlib.Open(mysql.New(mysql.Config{
		DSN:                       dsn,
		DefaultStringSize:         o.mysqlDefaultStringSize,
		DisableDatetimePrecision:  o.mysqlDisableDatetimePrecision,
		DontSupportRenameIndex:    o.mysqlDontSupportRenameIndex,
		DontSupportRenameColumn:   o.mysqlDontSupportRenameColumn,
		SkipInitializeWithVersion: o.mysqlSkipInitializeWithVersion,
	}), gormCfg)
}

func openPostgreSQL(cfg Config, o *options, gormCfg *gormlib.Config) (*gormlib.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		cfg.Host, cfg.Username, cfg.Password, cfg.Database, cfg.Port,
		o.postgresqlSSLMode, o.postgresqlTimeZone,
	)
	return gormlib.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: o.postgresqlPreferSimpleProtocol,
	}), gormCfg)
}

func openSQLite(cfg Config, gormCfg *gormlib.Config) (*gormlib.DB, error) {
	return gormlib.Open(sqlite.Open(cfg.Database), gormCfg)
}

func applyPoolOptions(db *gormlib.DB, o *options) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(o.maxIdleConns)
	sqlDB.SetMaxOpenConns(o.maxOpenConns)
	sqlDB.SetConnMaxLifetime(o.connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(o.connMaxIdleTime)
	return nil
}
