package gateway

import "context"

// Client 网关客户端
type Client interface {
	// Register 将启动的API服务注册到网关
	Register(ctx context.Context, api API) error

	// Deregister 从网关撤销API服务
	Deregister(ctx context.Context, api API) error

	// Close 关闭可能需要关闭的一些连接和资源
	Close(ctx context.Context) error
}
