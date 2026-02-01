package logrus

import (
	"io"
	"os"
	"strings"

	"github.com/CXeon/tiles/logger"
	"github.com/sirupsen/logrus"
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
	logger *logrus.Logger
}

func NewLogger(cfg Config) logger.Logger {
	l := logrus.New()

	// 根据 Filename 和 EnableStdout 决定输出目标
	var writer io.Writer
	if cfg.Filename == "" && cfg.EnableStdout {
		// 只输出到 stdout
		writer = os.Stdout
	} else if cfg.Filename != "" && cfg.EnableStdout {
		// 同时输出到文件和 stdout
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		writer = io.MultiWriter(fileWriter, os.Stdout)
	} else if cfg.Filename != "" {
		// 只输出到文件
		writer = &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
	} else {
		// Filename 为空且 EnableStdout 为 false，默认输出到 stdout
		writer = os.Stdout
	}

	l.SetOutput(writer)
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetLevel(parseLevel(cfg.Level))

	return &loggerImpl{logger: l}
}

func (l *loggerImpl) Debug(msg string, fields logger.Fields) {
	if len(fields) == 0 {
		l.logger.Debug(msg)
		return
	}
	l.logger.WithFields(toFields(fields)).Debug(msg)
}

func (l *loggerImpl) Info(msg string, fields logger.Fields) {
	if len(fields) == 0 {
		l.logger.Info(msg)
		return
	}
	l.logger.WithFields(toFields(fields)).Info(msg)
}

func (l *loggerImpl) Warn(msg string, fields logger.Fields) {
	if len(fields) == 0 {
		l.logger.Warn(msg)
		return
	}
	l.logger.WithFields(toFields(fields)).Warn(msg)
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
	l.logger.WithFields(toFields(fields)).Error(msg)
}

func toFields(fields logger.Fields) logrus.Fields {
	if len(fields) == 0 {
		return nil
	}
	lf := logrus.Fields{}
	for k, v := range fields {
		lf[k] = v
	}
	return lf
}

func parseLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logrus.DebugLevel
	case "info", "":
		return logrus.InfoLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}
