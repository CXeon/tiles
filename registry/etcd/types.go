package etcd

import (
	"time"
)

type Config struct {
	Endpoints   []string
	Username    string
	Password    string
	DialTimeout time.Duration

	LoadBalancerStrategy uint8 // 默认策略 0 round robin ,1 随机 ，2 加权随机
}
