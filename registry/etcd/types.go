package etcd

import (
	"time"

	"github.com/CXeon/tiles/registry"
)

type Config struct {
	Endpoints   []string
	Username    string
	Password    string
	DialTimeout time.Duration

	// 当前服务的身份信息（用于默认隔离上下文）
	// 如果为 nil，则 Discover 时必须显式传 ComProj
	CurrentEndpoint *registry.Endpoint
}
