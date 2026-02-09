package registry

import (
	"fmt"
)

type ProtocolType string

const (
	ProtocolTypeHttp  = "http"
	ProtocolTypeHttps = "https"
)

func (p ProtocolType) Validate() error {
	switch p {
	case ProtocolTypeHttp, ProtocolTypeHttps:
		return nil
	}
	return fmt.Errorf("protocol type %s is not supported", p)
}

type Endpoint struct {
	InstanceID string            // 实例ID 全局唯一
	Env        string            // 环境 比如Test 测试环境，Dev 开发环境，Prod 生产环境
	Cluster    string            // 集群 比如China 中国集群，America 美国集群，Europe 欧洲集群
	Company    string            // 公司名称 比如 TalentLimited
	Project    string            // 项目名称
	Service    string            // 服务的名称
	Protocol   ProtocolType      // 通信协议 比如http
	Color      string            // 染色 比如Red
	Ip         string            // 地址
	Port       uint16            // 端口
	Extra      map[string]string // 额外元数据
	TTL        uint32            // 生存时间（单位：秒）。0 表示永不过期。
	Weight     uint16            // 实例权重，为0时不设置
}

func (e *Endpoint) GetExtra(key string) string {
	if e.Extra == nil {
		return ""
	}
	return e.Extra[key]
}
func (e *Endpoint) ID() string {
	if len(e.InstanceID) > 0 {
		return e.InstanceID
	}
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s.%s:%d", e.Env, e.Cluster, e.Company, e.Project, e.Service, e.Protocol, e.Color, e.Ip, e.Port)
}
