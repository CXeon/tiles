package registry

import "context"

type Client interface {
	Register(ctx context.Context, endpoint *Endpoint) error

	Deregister(ctx context.Context, endpoint *Endpoint) error

	// Discover 一次性拉取指定服务的实例列表
	Discover(ctx context.Context, service []string, option ...ServiceOption) (CompanyRegistry, error)

	// Watch 订阅指定服务的实例变更
	Watch(ctx context.Context, service []string, option ...ServiceOption) error

	// Update(ctx context.Context, endpoint *Endpoint) error

	GetService(ctx context.Context, service string, option ...GetServiceOption) (Endpoint, error)

	Close(ctx context.Context) error
}
