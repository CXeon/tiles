package gateway

import "context"

// Client 网关客户端
type Client interface {
	// Register 将启动的服务实例注册到网关
	Register(ctx context.Context, endpoint *Endpoint) error

	// Deregister 从网关撤销服务实例
	Deregister(ctx context.Context, endpoint *Endpoint) error

	// Update 更新服务实例信息
	Update(ctx context.Context, endpoint *Endpoint) error

	// Close 关闭可能需要关闭的一些连接和资源
	Close(ctx context.Context) error
}
