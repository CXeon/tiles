module github.com/CXeon/tiles/logger/logrus

go 1.24.2

require (
	github.com/CXeon/tiles/logger v0.0.0
	github.com/sirupsen/logrus v1.9.3
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect

replace github.com/CXeon/tiles/logger => ..
