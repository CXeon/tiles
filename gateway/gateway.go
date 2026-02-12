package gateway

import "context"

// Client 网关客户端接口，定义了服务实例在网关中注册、注销、更新的通用操作
//
// 具体实现可能支持不同的特性（如 TTL、自动续约等），请参考具体实现的文档
type Client interface {
	// Register 将启动的服务实例注册到网关
	Register(ctx context.Context, endpoint *Endpoint) error

	// Deregister 从网关撤销服务实例
	Deregister(ctx context.Context, endpoint *Endpoint) error

	// Update 更新服务实例信息
	Update(ctx context.Context, endpoint *Endpoint) error

	// KeepAlive 手动续约服务实例配置（如果实现支持 TTL 机制）
	// 注：某些实现可能支持自动续约，具体请参考实现文档
	KeepAlive(ctx context.Context, endpoint *Endpoint) error

	// Close 关闭可能需要关闭的一些连接和资源
	Close(ctx context.Context) error
}
