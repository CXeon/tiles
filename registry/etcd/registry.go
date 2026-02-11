package etcd

import (
	"context"

	"github.com/CXeon/tiles/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Registry struct {
	handler *handler
}

func NewRegistry(conf Config) (*Registry, error) {

	h := &handler{
		conf:      conf,
		endpoints: make(registry.CompanyRegistry),
	}

	// 和etcd建立连接
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Endpoints,
		DialTimeout: conf.DialTimeout,
		Username:    conf.Username,
		Password:    conf.Password,
	})

	if err != nil {
		return nil, err
	}

	h.cli = cli

	return &Registry{
		handler: h,
	}, nil
}

func (r *Registry) Register(ctx context.Context, endpoint *registry.Endpoint) error {
	if endpoint == nil {
		return registry.ErrInvalidEndpoint
	}

	err := r.handler.register(ctx, *endpoint)
	if err != nil {
		return err
	}

	return nil

}

func (r *Registry) Deregister(ctx context.Context, endpoint *registry.Endpoint) error {
	if endpoint == nil {
		return registry.ErrInvalidEndpoint
	}
	return r.handler.deregister(ctx, *endpoint)
}

func (r *Registry) Discover(ctx context.Context, services []string, option ...registry.ServiceOption) (registry.CompanyRegistry, error) {
	if len(services) == 0 {
		return nil, registry.ErrEmptyServices
	}
	return r.handler.discover(ctx, services, option...)
}

func (r *Registry) Watch(ctx context.Context, services []string, option ...registry.ServiceOption) error {
	if len(services) == 0 {
		return registry.ErrEmptyServices
	}
	return r.handler.watch(ctx, services, option...)
}

func (r *Registry) GetService(ctx context.Context, service string, option ...registry.GetServiceOption) (registry.Endpoint, error) {
	if len(service) == 0 {
		return registry.Endpoint{}, registry.ErrEmptyService
	}
	return r.handler.getService(ctx, service, option...)
}

func (r *Registry) Close(ctx context.Context) error {
	return r.handler.close()
}
