package logrus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CXeon/tiles/logger"
)

func TestLogrusLogger_StdoutOnly(t *testing.T) {
	cfg := Config{
		Filename:     "",
		EnableStdout: true,
		Level:        "debug",
	}

	log := NewLogger(cfg)
	if log == nil {
		t.Fatal("NewLogger returned nil")
	}

	log.Debug("debug message", logger.Fields{"key": "value"})
	log.Info("info message", logger.Fields{"user_id": 123})
	log.Warn("warn message", nil)
	log.Error("error message", nil, nil)
}

func TestLogrusLogger_FileOnly(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		Filename:     logFile,
		MaxSize:      1,
		MaxBackups:   3,
		MaxAge:       7,
		Compress:     false,
		EnableStdout: false,
		Level:        "info",
	}

	log := NewLogger(cfg)
	if log == nil {
		t.Fatal("NewLogger returned nil")
	}

	log.Info("file only log", logger.Fields{"test": "data"})
	log.Error("file error log", os.ErrNotExist, logger.Fields{"path": "/tmp/test"})

	// 验证文件是否创建
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("log file was not created: %s", logFile)
	}
}

func TestLogrusLogger_FileAndStdout(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test_multi.log")

	cfg := Config{
		Filename:     logFile,
		MaxSize:      1,
		MaxBackups:   3,
		MaxAge:       7,
		Compress:     false,
		EnableStdout: true,
		Level:        "debug",
	}

	log := NewLogger(cfg)
	if log == nil {
		t.Fatal("NewLogger returned nil")
	}

	log.Debug("multi output debug", logger.Fields{"multi": true})
	log.Info("multi output info", logger.Fields{"test": 456})
	log.Warn("multi output warn", logger.Fields{"status": "warning"})
	log.Error("multi output error", os.ErrClosed, logger.Fields{"resource": "file"})

	// 验证文件是否创建
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("log file was not created: %s", logFile)
	}
}

func TestLogrusLogger_ErrorWithNilFields(t *testing.T) {
	cfg := Config{
		Filename:     "",
		EnableStdout: true,
		Level:        "error",
	}

	log := NewLogger(cfg)
	if log == nil {
		t.Fatal("NewLogger returned nil")
	}

	log.Error("error with nil fields", os.ErrPermission, nil)
	log.Error("error with nil error", nil, logger.Fields{"field": "value"})
	log.Error("error both nil", nil, nil)
}

func TestLogrusLogger_LevelParsing(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
		{"empty defaults to info", ""},
		{"unknown defaults to info", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Filename:     "",
				EnableStdout: true,
				Level:        tt.level,
			}

			log := NewLogger(cfg)
			if log == nil {
				t.Fatalf("NewLogger returned nil for level: %s", tt.level)
			}

			log.Info("test message", nil)
		})
	}
}
