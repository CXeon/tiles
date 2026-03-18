package gormlog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CXeon/tiles/logger"
	gormlib "gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const defaultSlowThreshold = 200 * time.Millisecond

// WrappedLogger 将 tiles logger.Logger 适配为 gorm logger.Interface
type WrappedLogger struct {
	inner         logger.Logger
	logLevel      gormlogger.LogLevel
	slowThreshold time.Duration
}

// Option 是 WrappedLogger 的可选配置函数
type Option func(*WrappedLogger)

// WithLogLevel 设置初始日志级别，默认 Warn
func WithLogLevel(l gormlogger.LogLevel) Option {
	return func(w *WrappedLogger) {
		w.logLevel = l
	}
}

// WithSlowThreshold 设置慢查询阈值，默认 200ms
func WithSlowThreshold(d time.Duration) Option {
	return func(w *WrappedLogger) {
		w.slowThreshold = d
	}
}

// New 用 tiles logger.Logger 创建一个满足 gorm logger.Interface 的 WrappedLogger
func New(l logger.Logger, opts ...Option) *WrappedLogger {
	w := &WrappedLogger{
		inner:         l,
		logLevel:      gormlogger.Info,
		slowThreshold: defaultSlowThreshold,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// LogMode 返回指定日志级别的新实例
func (w *WrappedLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	copied := *w
	copied.logLevel = level
	return &copied
}

// Info 对应 GORM 的 Info 级别日志
func (w *WrappedLogger) Info(_ context.Context, format string, args ...any) {
	if w.logLevel < gormlogger.Info {
		return
	}
	w.inner.Info(fmt.Sprintf(format, args...), nil)
}

// Warn 对应 GORM 的 Warn 级别日志
func (w *WrappedLogger) Warn(_ context.Context, format string, args ...any) {
	if w.logLevel < gormlogger.Warn {
		return
	}
	w.inner.Warn(fmt.Sprintf(format, args...), nil)
}

// Error 对应 GORM 的 Error 级别日志
func (w *WrappedLogger) Error(_ context.Context, format string, args ...any) {
	if w.logLevel < gormlogger.Error {
		return
	}
	w.inner.Error(fmt.Sprintf(format, args...), nil, nil)
}

// Trace 记录每条 SQL 的执行情况，根据耗时和错误自动选择日志级别
func (w *WrappedLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if w.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := logger.Fields{
		"elapsed":       elapsed,
		"rows_affected": rows,
		"sql":           sql,
	}

	switch {
	case err != nil && !errors.Is(err, gormlib.ErrRecordNotFound):
		if w.logLevel >= gormlogger.Error {
			w.inner.Error("gorm trace", err, fields)
		}
	case elapsed >= w.slowThreshold:
		if w.logLevel >= gormlogger.Warn {
			fields["slow_threshold"] = w.slowThreshold
			w.inner.Warn("gorm slow query", fields)
		}
	default:
		if w.logLevel >= gormlogger.Info {
			w.inner.Info("gorm trace", fields)
		}
	}
}
