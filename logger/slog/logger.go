package slog

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/CXeon/tiles/logger"
	"gopkg.in/natefinch/lumberjack.v2"
)

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
	// Level 是默认的日志级别，例如："debug"、"info"、"warn"、"error"
	Level string
	// EnableStdout 是否同时输出到标准输出
	EnableStdout bool
}

type loggerImpl struct {
	logger *slog.Logger
}

func NewLogger(cfg Config) logger.Logger {
	// 根据 Filename 和 EnableStdout 决定输出目标
	var w io.Writer
	if cfg.Filename == "" && cfg.EnableStdout {
		// 只输出到 stdout
		w = os.Stdout
	} else if cfg.Filename != "" && cfg.EnableStdout {
		// 同时输出到文件和 stdout
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		w = io.MultiWriter(fileWriter, os.Stdout)
	} else if cfg.Filename != "" {
		// 只输出到文件
		w = &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
	} else {
		// Filename 为空且 EnableStdout 为 false，默认输出到 stdout
		w = os.Stdout
	}

	level := parseLevel(cfg.Level)

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})

	return &loggerImpl{logger: slog.New(h)}
}

func (l *loggerImpl) Debug(msg string, fields logger.Fields) {
	if len(fields) == 0 {
		l.logger.Debug(msg)
		return
	}
	l.logger.Debug(msg, toKeyValues(fields)...)
}

func (l *loggerImpl) Info(msg string, fields logger.Fields) {
	if len(fields) == 0 {
		l.logger.Info(msg)
		return
	}
	l.logger.Info(msg, toKeyValues(fields)...)
}

func (l *loggerImpl) Warn(msg string, fields logger.Fields) {
	if len(fields) == 0 {
		l.logger.Warn(msg)
		return
	}
	l.logger.Warn(msg, toKeyValues(fields)...)
}

func (l *loggerImpl) Error(msg string, err error, fields logger.Fields) {
	if fields == nil {
		fields = logger.Fields{}
	}
	if err != nil {
		fields["err"] = err
	}
	if len(fields) == 0 {
		l.logger.Error(msg)
		return
	}
	l.logger.Error(msg, toKeyValues(fields)...)
}

func toKeyValues(fields logger.Fields) []any {
	if len(fields) == 0 {
		return nil
	}

	kv := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		kv = append(kv, k, v)
	}

	return kv
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
