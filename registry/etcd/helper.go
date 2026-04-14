package etcd

import (
	"context"
	"fmt"

	"github.com/CXeon/tiles/registry"
	"github.com/bytedance/sonic"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func (h *handler) getServiceEtcdKey(e registry.Endpoint) string {

	return fmt.Sprintf("%s/%s/%s/%s/%s/%s@%s", e.Env, e.Cluster, e.Company, e.Project, e.Protocol, e.Service, e.Color)
}

func (h *handler) buildDiscoverKey(env, cluster, company, project string, protocol registry.ProtocolType, service, color string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s@%s", env, cluster, company, project, protocol, service, color)
}

func (h *handler) marshalEndpointsStr(list []registry.Endpoint) (string, error) {
	return sonic.MarshalString(list)
}

func (h *handler) unmarshalEndpointsStr(s string) ([]registry.Endpoint, error) {
	list := make([]registry.Endpoint, 0)
	err := sonic.UnmarshalString(s, &list)
	return list, err
}

func (h *handler) unmarshalEndpoints(b []byte) ([]registry.Endpoint, error) {
	list := make([]registry.Endpoint, 0)
	err := sonic.Unmarshal(b, &list)
	return list, err
}

func (h *handler) getFromEtcd(ctx context.Context, key string, isPrefix bool) (map[string][]registry.Endpoint, error) {

	opts := []clientv3.OpOption{}
	if isPrefix {
		opts = append(opts, clientv3.WithPrefix())
	}
	resp, err := h.cli.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]registry.Endpoint)
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		var list []registry.Endpoint
		list, err := h.unmarshalEndpoints(kv.Value)
		if err != nil {
			return nil, err
		}
		result[key] = list
	}

	return result, nil
}

func (h *handler) getEndpoints(ctx context.Context, key string) ([]registry.Endpoint, error) {
	result, err := h.getFromEtcd(ctx, key, false)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[key], nil
}

func (h *handler) putEndpoint(ctx context.Context, key string, endpoint registry.Endpoint) error {
	// 从etcd查询数据检查重复
	list, err := h.getEndpoints(ctx, key)
	if err != nil {
		return err
	}

	exists := false
	for _, e := range list {
		if e.ID() == endpoint.ID() {
			exists = true
			break
		}
	}

	if !exists {
		list = append(list, endpoint)

		err = h.putEndpoints(ctx, key, list)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) putEndpoints(ctx context.Context, key string, list []registry.Endpoint) error {
	if list == nil {
		list = make([]registry.Endpoint, 0)
	}
	str, err := h.marshalEndpointsStr(list)
	if err != nil {
		return err
	}
	_, err = h.cli.Put(ctx, key, str, clientv3.WithLease(h.leaseID))
	if err != nil {
		return err
	}
	return nil
}

func (h *handler) createLoadBalancer() registry.LoadBalancer {
	switch h.conf.LoadBalancerStrategy {
	case 1:
		return registry.NewRandomBalancer()
	case 2:
		return registry.NewWeightedRandomBalancer()
	default:
		return registry.NewRoundRobinBalancer()
	}
}
