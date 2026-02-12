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
	// TTL        uint32            // 生存时间（单位：秒）。0 表示永不过期。
	Weight uint16 // 实例权重，为0时不设置
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
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s:%d", e.Env, e.Cluster, e.Company, e.Project, e.Service, e.Protocol, e.Color, e.Ip, e.Port)
}

// 获取实例权重
func (e *Endpoint) GetWeight() uint16 {
	return e.Weight
}

type ServiceOpt struct {
	// 跨公司/项目查询
	// - 如果为空，则使用 CurrentEndpoint 的 Company + Project（默认同隔离）
	// - 如果指定，则在"当前环境隔离级别"下，跨指定的 Company + Project 查询
	ComProj map[string][]string // key=Company, value=[]Project
}

type ServiceOption func(*ServiceOpt)

func WithGetOptComProj(comProj map[string][]string) ServiceOption {
	return func(o *ServiceOpt) {
		o.ComProj = comProj
	}
}

type DiscoveredEndpoints struct {
	m CompanyRegistry
}

type EndpointsWithLoadBalancer struct {
	Endpoints    []Endpoint
	LoadBalancer LoadBalancer
}

// ServiceRegistry 表示某个项目下所有服务的注册信息
// key=服务名, value=该服务的实例列表+负载均衡器
type ServiceRegistry map[string]EndpointsWithLoadBalancer

// ProjectRegistry 表示某个公司下所有项目的服务注册信息
// key=项目名, value=该项目的服务注册表
type ProjectRegistry map[string]ServiceRegistry

// CompanyRegistry 表示所有公司的服务注册信息（服务发现的返回结果）
// key=公司名, value=该公司的项目注册表
type CompanyRegistry map[string]ProjectRegistry

type GetServiceOpt struct {
	Company string
	Project string
}

type GetServiceOption func(*GetServiceOpt)

func WithGetCompany(company string) GetServiceOption {
	return func(o *GetServiceOpt) {
		o.Company = company
	}
}

func WithGetProject(project string) GetServiceOption {
	return func(o *GetServiceOpt) {
		o.Project = project
	}
}
