module github.com/CXeon/tiles/logger/zap

go 1.24.2

require (
	github.com/CXeon/tiles/logger v0.0.0
	go.uber.org/zap v1.27.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require go.uber.org/multierr v1.10.0 // indirect

replace github.com/CXeon/tiles/logger => ..
