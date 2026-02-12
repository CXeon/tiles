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
	endpoints       registry.CompanyRegistry
	lock            sync.RWMutex // 保护endpoints

	loadBalancer registry.LoadBalancer

	// 内部管理的 context，用于控制 生命周期
	cancelFuncKeepAlive context.CancelFunc
	cancelFuncWatch     context.CancelFunc
}

func (h *handler) register(ctx context.Context, endpoint registry.Endpoint) error {
	if h.currentEndpoint != nil {
		return registry.ErrNotRegistered
	}
	// 1.构造etcd的key
	key := h.getServiceEtcdKey(endpoint)
	// 2. 创建etcd租约
	lease := clientv3.NewLease(h.cli)
	resp, err := lease.Grant(ctx, 10)
	if err != nil {
		// h.lease.Close()
		return err
	}
	h.leaseID = resp.ID
	h.lease = lease

	// 3.保存到etcd
	err = h.putEndpoint(ctx, key, endpoint)
	if err != nil {
		// lease.Close()
		return err
	}

	// 4. 创建一个内部 context，用于控制 keepAlive 的生命周期
	innerCtx, cancel := context.WithCancel(context.Background())
	h.cancelFuncKeepAlive = cancel

	// 5. 启动 keepAlive goroutine（使用内部 context）
	go func() {
		h.keepAlive(innerCtx)
	}()

	h.currentEndpoint = &endpoint

	return nil
}

func (h *handler) keepAlive(ctx context.Context) error {
	respChan, err := h.lease.KeepAlive(ctx, h.leaseID)
	if err != nil {
		return err
	}
	for {
		select {
		case _, ok := <-respChan:
			if !ok {
				// channel 关闭，说明 keepAlive 失败
				return errors.New("keepAlive channel closed")
			}
		case <-ctx.Done():
			// context 被取消，正常退出
			return nil
		}
	}
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

	// 使用新切片代替原地修改
	newList := make([]registry.Endpoint, 0, len(list)-1)
	for _, e := range list {
		if e.ID() != endpoint.ID() {
			newList = append(newList, e)
		}
	}

	// 保存到etcd
	err = h.putEndpoints(ctx, key, newList)
	if err != nil {
		return err
	}

	return nil

}

func (h *handler) discover(ctx context.Context, services []string, option ...registry.ServiceOption) (registry.CompanyRegistry, error) {
	// 1. 检查 currentEndpoint 是否存在
	if h.currentEndpoint == nil {
		return nil, registry.ErrCurrentEndpointNil
	}

	// 2. 构建默认 opt（ComProj 默认为当前实例的 Company + Project）
	opt := registry.ServiceOpt{
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
		return nil, registry.ErrComProjEmpty
	}

	// 5. 锁定隔离维度（从 currentEndpoint 提取）
	env := h.currentEndpoint.Env
	cluster := h.currentEndpoint.Cluster
	protocol := h.currentEndpoint.Protocol
	if protocol == registry.ProtocolTypeHttps {
		protocol = registry.ProtocolTypeHttp
	}
	color := h.currentEndpoint.Color

	// 6. 按 services + ComProj 遍历查询，组织成 Company -> Project -> Service -> Instances 的结构
	result := make(registry.CompanyRegistry)
	for company, projects := range opt.ComProj {
		for _, project := range projects {
			for _, service := range services {
				key := h.buildDiscoverKey(env, cluster, company, project, protocol, service, color)
				list, err := h.getEndpoints(ctx, key)
				if err != nil {
					return nil, err
				}
				if len(list) == 0 {
					continue
				}

				// 确保 Company 层存在
				if result[company] == nil {
					result[company] = make(registry.ProjectRegistry)
				}

				// 确保 Project 层存在
				if result[company][project] == nil {
					result[company][project] = make(registry.ServiceRegistry)
				}

				result[company][project][service] = registry.EndpointsWithLoadBalancer{
					Endpoints:    list,
					LoadBalancer: h.createLoadBalancer(),
				}
			}
		}
	}

	// 7. 合并到handler的endpoints中(保持已存在的LoadBalancer)
	h.lock.Lock()
	defer h.lock.Unlock()

	// 合并result到h.endpoints,保持已存在的LoadBalancer
	for company, projects := range result {
		if h.endpoints[company] == nil {
			h.endpoints[company] = make(registry.ProjectRegistry)
		}
		for project, services := range projects {
			if h.endpoints[company][project] == nil {
				h.endpoints[company][project] = make(registry.ServiceRegistry)
			}
			for service, newData := range services {
				// 检查是否已存在LoadBalancer
				existing := h.endpoints[company][project][service]
				if existing.LoadBalancer != nil {
					// 保持原有LoadBalancer，只更新Endpoints
					newData.LoadBalancer = existing.LoadBalancer
				}
				h.endpoints[company][project][service] = newData
			}
		}
	}

	return result, nil
}

func (h *handler) watch(ctx context.Context, services []string, option ...registry.ServiceOption) error {
	// 1. 检查 currentEndpoint 是否存在
	if h.currentEndpoint == nil {
		return registry.ErrCurrentEndpointNil
	}

	// 2. 构建默认 opt（ComProj 默认为当前实例的 Company + Project）
	opt := registry.ServiceOpt{
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
		return registry.ErrComProjEmpty
	}

	// 5. 锁定隔离维度（从 currentEndpoint 提取）
	env := h.currentEndpoint.Env
	cluster := h.currentEndpoint.Cluster
	protocol := h.currentEndpoint.Protocol
	if protocol == registry.ProtocolTypeHttps {
		protocol = registry.ProtocolTypeHttp
	}
	color := h.currentEndpoint.Color

	// 6. 创建watch context
	watchCtx, cancelFuncWatch := context.WithCancel(context.Background())
	h.cancelFuncWatch = cancelFuncWatch

	// 7. 使用goroutine并发watch多个服务
	var wg sync.WaitGroup
	for company, projects := range opt.ComProj {
		for _, project := range projects {
			for _, service := range services {
				key := h.buildDiscoverKey(env, cluster, company, project, protocol, service, color)

				wg.Add(1)
				go func(key, company, project, service string) {
					defer wg.Done()
					h.watchSingleService(watchCtx, key, company, project, service)
				}(key, company, project, service)
			}
		}
	}

	// 8. 在后台等待所有goroutine结束
	go func() {
		wg.Wait()
	}()

	return nil
}

// watchSingleService 监听单个服务的变更
func (h *handler) watchSingleService(ctx context.Context, key, company, project, service string) {
	watchChan := h.cli.Watch(ctx, key)
	for {
		select {
		case watchResp, ok := <-watchChan:
			if !ok {
				// channel 已关闭，退出
				return
			}
			if watchResp.Err() != nil {
				// watch 失败，退出
				return
			}
			for _, ev := range watchResp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					// 更新或者新增handler里记录的服务实例信息
					list, err := h.unmarshalEndpoints(ev.Kv.Value)
					if err != nil {
						return
					}
					h.lock.Lock()
					if h.endpoints[company] == nil {
						h.endpoints[company] = make(registry.ProjectRegistry)
					}
					if h.endpoints[company][project] == nil {
						h.endpoints[company][project] = make(registry.ServiceRegistry)
					}

					// 检查是否已存在LoadBalancer，如果存在则保持，否则创建新的
					existing := h.endpoints[company][project][service]
					if existing.LoadBalancer == nil {
						// 首次创建，初始化LoadBalancer
						h.endpoints[company][project][service] = registry.EndpointsWithLoadBalancer{
							Endpoints:    list,
							LoadBalancer: h.createLoadBalancer(),
						}
					} else {
						// 保持原有LoadBalancer，只更新Endpoints
						existing.Endpoints = list
						h.endpoints[company][project][service] = existing
					}

					h.lock.Unlock()
				case clientv3.EventTypeDelete:
					// 从handler里记录的服务实例信息中删除对应的实例
					list, err := h.unmarshalEndpoints(ev.Kv.Value)
					if err != nil {
						return
					}
					// list 处理成map
					endpointMap := make(map[string]registry.Endpoint)
					for _, endpoint := range list {
						endpointMap[endpoint.ID()] = endpoint
					}

					h.lock.Lock()
					for c, projects := range h.endpoints {
						for p, instances := range projects {
							for s, endpoints := range instances {
								oldList := endpoints.Endpoints

								// 使用新切片代替原地修改
								newList := make([]registry.Endpoint, 0, len(oldList))
								for _, endpoint := range oldList {
									if _, ok := endpointMap[endpoint.ID()]; !ok {
										newList = append(newList, endpoint)
									}
								}

								// 只有实际删除了元素才更新
								if len(newList) != len(oldList) {
									// 保持原有LoadBalancer，只更新Endpoints
									existing := h.endpoints[c][p][s]
									existing.Endpoints = newList
									h.endpoints[c][p][s] = existing
								}
							}
						}
					}
					h.lock.Unlock()
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *handler) getService(ctx context.Context, service string, option ...registry.GetServiceOption) (registry.Endpoint, error) {
	// 1. 检查 currentEndpoint 是否存在
	if h.currentEndpoint == nil {
		return registry.Endpoint{}, registry.ErrCurrentEndpointNil
	}

	// 2. 构建默认 opt（Company + Project 默认为当前实例的）
	opt := registry.GetServiceOpt{
		Company: h.currentEndpoint.Company,
		Project: h.currentEndpoint.Project,
	}

	// 3. 应用用户传入的 option
	for _, o := range option {
		o(&opt)
	}

	// 4. 校验：Company 和 Project 不能为空
	if opt.Company == "" || opt.Project == "" {
		return registry.Endpoint{}, registry.ErrEmptyCompanyOrProject
	}

	// 5. 从缓存的 endpoints 中查询
	h.lock.RLock()
	defer h.lock.RUnlock()

	if h.endpoints[opt.Company] == nil {
		return registry.Endpoint{}, registry.ErrServiceNotFound
	}

	if h.endpoints[opt.Company][opt.Project] == nil {
		return registry.Endpoint{}, registry.ErrServiceNotFound
	}

	instances := h.endpoints[opt.Company][opt.Project][service]
	if len(instances.Endpoints) == 0 {
		return registry.Endpoint{}, registry.ErrServiceNotFound
	}

	// 6. 负载均衡
	result := instances.LoadBalancer.Select(instances.Endpoints)
	if result == nil {
		return registry.Endpoint{}, errors.New("no available endpoint")
	}
	return *result, nil
}

func (h *handler) close() error {
	// 1. 先取消 keepAlive和watch goroutine
	if h.cancelFuncKeepAlive != nil {
		h.cancelFuncKeepAlive()
		h.cancelFuncKeepAlive = nil
	}
	if h.cancelFuncWatch != nil {
		h.cancelFuncWatch()
		h.cancelFuncWatch = nil
	}

	// 2. 关闭 lease 和 client
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
