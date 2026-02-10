package etcd

import (
	"time"
)

type Config struct {
	Endpoints   []string
	Username    string
	Password    string
	DialTimeout time.Duration
}
