package registry

import "context"

type Client interface {
	Register(ctx context.Context, endpoint *Endpoint) error

	Deregister(ctx context.Context, endpoint *Endpoint) error

	// Discover 一次性拉取指定服务的实例列表
	//
	// 隔离语义：
	//   - Env、Cluster、Protocol、Color 由 Client 创建时的 CurrentEndpoint 决定，无法通过 option 修改
	//   - 如果创建 Client 时未提供 CurrentEndpoint，则必须通过 option 指定 ComProj，否则返回错误
	//
	// 默认行为（不传 option）：
	//   - 查询"与 CurrentEndpoint 同公司、同项目"下的服务
	//
	// 跨公司/项目查询：
	//   - 通过 WithGetOptComProj 指定目标 Company + Project 列表
	//   - 环境隔离维度（Env/Cluster/Protocol/Color）保持不变
	Discover(ctx context.Context, service []string, option ...GetServiceOption) ([]*Endpoint, error)

	// Watch 订阅指定服务的实例变更
	Watch(ctx context.Context, service string, handler WatchHandler, option ...WatchOption) (Watcher, error)

	Update(ctx context.Context, endpoint *Endpoint) error

	Close(ctx context.Context) error
}

type WatchHandler func(service string, endpoints []*Endpoint)

type Watcher interface {
	Stop(ctx context.Context) error
}
