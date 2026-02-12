package traefik

import (
	"errors"
	"time"
)

// 默认中间件名称
const (
	DefaultAuthMiddleware = "ForwardAuth" // 默认的身份验证中间件
)

// 默认 Header Key 名称
const (
	HeaderKeyEnv     = "X-Env"     // 环境标识
	HeaderKeyCluster = "X-Cluster" // 集群标识
	HeaderKeyColor   = "X-Color"   // 染色标识
)

type ProviderType uint8

const (
	ProviderTypeRedis ProviderType = iota + 1
	ProviderTypeConsul
	ProviderTypeEtcd
	ProviderTypeZooKeeper
)

func (pt ProviderType) String() string {
	return [...]string{"Redis", "Consul", "Etcd", "ZooKeeper"}[pt-1]
}

func (pt ProviderType) Validate() error {
	if pt < 1 || pt > 4 {
		return errors.New("traefik provider type is not currently supported")
	}
	return nil
}

type Provider struct {
	KVType    ProviderType
	Endpoints []string
	Username  string
	Password  string
	DBIndex   int
	Namespace string

	// Connection pool settings
	PoolSize     int // Maximum number of connections
	MinIdleConns int // Minimum number of idle connections
	MaxIdleConns int // Maximum number of idle connections

	// Timeout settings
	ConnectTimeout time.Duration // Connection timeout
	ReadTimeout    time.Duration // Read operation timeout
	WriteTimeout   time.Duration // Write operation timeout
}
