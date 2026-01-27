package traefik

import (
	"errors"
	"time"
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
