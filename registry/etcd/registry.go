package etcd

import (
	"context"
	"errors"

	"github.com/CXeon/tiles/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Registry struct {
	handler *handler
}

func NewRegistry(conf Config) (*Registry, error) {

	h := &handler{
		conf:      conf,
		endpoints: make([]*registry.Endpoint, 0),
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
		return errors.New("endpoint is nil")
	}

	err := r.handler.register(ctx, *endpoint)
	if err != nil {
		return err
	}

	return nil

}

func (r *Registry) Deregister(ctx context.Context, endpoint *registry.Endpoint) error {
	if endpoint == nil {
		return errors.New("endpoint is nil")
	}
	return r.handler.deregister(ctx, *endpoint)
}

func (r *Registry) Discover(ctx context.Context, services []string, option ...registry.GetServiceOption) ([]*registry.Endpoint, error) {
	return r.handler.discover(ctx, services, option...)
}

func (r *Registry) Watch(ctx context.Context, service string, handler registry.WatchHandler, option ...registry.WatchOption) (registry.Watcher, error) {
	// TODO implement me
	panic("implement me")
}

func (r *Registry) Update(ctx context.Context, endpoint *registry.Endpoint) error {
	// TODO implement me
	panic("implement me")
}

func (r *Registry) Close(ctx context.Context) error {
	return r.handler.close()
}
