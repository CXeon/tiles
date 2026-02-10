package etcd

import (
	"context"
	"errors"
	"sync"

	"github.com/CXeon/tiles/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type handler struct {
	conf            Config
	cli             *clientv3.Client
	lease           clientv3.Lease
	leaseID         clientv3.LeaseID
	currentEndpoint *registry.Endpoint // 当前服务身份信息，用于默认隔离上下文
	currentInstance *registry.Endpoint // 当前已注册的实例
	endpoints       []*registry.Endpoint
	lock            sync.RWMutex // 保护endpoints
}

func (h *handler) register(ctx context.Context, endpoint registry.Endpoint) error {
	if h.currentInstance != nil {
		return errors.New("register already registered")
	}
	// 1.构造etcd的key
	key := h.getServiceEtcdKey(endpoint)
	// 2. 创建etcd租约
	lease := clientv3.NewLease(h.cli)
	resp, err := lease.Grant(ctx, 10)
	if err != nil {
		h.lease.Close()
		return err
	}
	h.leaseID = resp.ID
	h.lease = lease

	// 3.保存到etcd
	err = h.putEndpoint(ctx, key, endpoint)
	if err != nil {
		h.lease.Close()
		return err
	}

	h.currentInstance = &endpoint

	return nil
}

func (h *handler) deregister(ctx context.Context, endpoint registry.Endpoint) error {

	key := h.getServiceEtcdKey(endpoint)

	list, err := h.getEndpoints(ctx, key)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}

	// 从list找到对应实例删除
	for i, e := range list {
		if e.ID() == endpoint.ID() {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}

	// 保存到etcd
	err = h.putEndpoints(ctx, key, list)
	if err != nil {
		return err
	}

	return nil

}

func (h *handler) discover(ctx context.Context, services []string, option ...registry.GetServiceOption) ([]*registry.Endpoint, error) {
	// 1. 检查 currentEndpoint 是否存在
	if h.currentEndpoint == nil {
		return nil, errors.New("currentEndpoint is nil, cannot determine isolation context")
	}

	// 2. 构建默认 opt（ComProj 默认为当前实例的 Company + Project）
	opt := registry.GetServiceOpt{
		ComProj: map[string][]string{
			h.currentEndpoint.Company: {h.currentEndpoint.Project},
		},
	}

	// 3. 应用用户传入的 option（只会覆盖 ComProj）
	for _, o := range option {
		o(&opt)
	}

	// 4. 校验：ComProj 不能为空
	if len(opt.ComProj) == 0 {
		return nil, errors.New("ComProj cannot be empty")
	}

	// 5. 锁定隔离维度（从 currentEndpoint 提取）
	env := h.currentEndpoint.Env
	cluster := h.currentEndpoint.Cluster
	protocol := h.currentEndpoint.Protocol
	if protocol == registry.ProtocolTypeHttps {
		protocol = registry.ProtocolTypeHttp
	}
	color := h.currentEndpoint.Color

	// 6. 按 services + ComProj 遍历查询
	result := make([]*registry.Endpoint, 0)
	for _, service := range services {
		for company, projects := range opt.ComProj {
			for _, project := range projects {
				key := h.buildDiscoverKey(env, cluster, company, project, protocol, service, color)
				list, err := h.getEndpoints(ctx, key)
				if err != nil {
					return nil, err
				}
				// 转成指针切片
				for _, ep := range list {
					epCopy := ep
					result = append(result, &epCopy)
				}
			}
		}
	}

	return result, nil
}

func (h *handler) close() error {
	var err error
	if h.lease != nil {
		err = h.lease.Close()
	}
	if h.cli != nil {
		cliErr := h.cli.Close()
		if cliErr != nil {
			err = cliErr
		}
	}
	return err
}
