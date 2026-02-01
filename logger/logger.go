package logger

type Fields map[string]any

type Logger interface {
	Debug(msg string, fields Fields)
	Info(msg string, fields Fields)
	Warn(msg string, fields Fields)
	Error(msg string, err error, fields Fields)
}
