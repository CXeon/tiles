package traefik

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/CXeon/tiles/gateway"
	"github.com/CXeon/tiles/gateway/traefik/kv_store"
)

type HandlerOptions struct {
	HealthCheckPath  string
	ExcludeAuthPaths []string
}

type handler struct {
	store kv_store.KvStore
}

func NewHandler(ctx context.Context, provider *Provider) (*handler, error) {
	var store kv_store.KvStore
	var err error

	switch provider.KVType {
	case ProviderTypeRedis:
		store, err = kv_store.NewRedisStore(kv_store.RedisConfig{
			Endpoints:      provider.Endpoints,
			Password:       provider.Password,
			DB:             provider.DBIndex,
			PoolSize:       provider.PoolSize,
			MinIdleConns:   provider.MinIdleConns,
			ConnectTimeout: provider.ConnectTimeout,
			ReadTimeout:    provider.ReadTimeout,
			WriteTimeout:   provider.WriteTimeout,
		})
	case ProviderTypeConsul:
		store, err = kv_store.NewConsulStore(kv_store.ConsulConfig{
			Endpoints:      provider.Endpoints,
			Username:       provider.Username,
			Password:       provider.Password,
			ConnectTimeout: provider.ConnectTimeout,
			ReadTimeout:    provider.ReadTimeout,
		})
	case ProviderTypeEtcd:
		store, err = kv_store.NewEtcdStore(kv_store.EtcdConfig{
			Endpoints:      provider.Endpoints,
			Username:       provider.Username,
			Password:       provider.Password,
			ConnectTimeout: provider.ConnectTimeout,
			ReadTimeout:    provider.ReadTimeout,
		})
	case ProviderTypeZooKeeper:
		store, err = kv_store.NewZookeeperStore(kv_store.ZookeeperConfig{
			Endpoints:      provider.Endpoints,
			ConnectTimeout: provider.ConnectTimeout,
			SessionTimeout: provider.ReadTimeout,
		})
	default:
		return nil, fmt.Errorf("unsupported traefik provider type: %v", provider.KVType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kv store: %w", err)
	}

	return &handler{
		store: store,
	}, nil
}

func (h *handler) Register(ctx context.Context, endpoint gateway.Endpoint, opts ...HandlerOptions) error {
	var opt HandlerOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	constructor := NewConstructor()
	forwardAuth := constructor.MiddlewareName(endpoint.Env, endpoint.Cluster, endpoint.Company, endpoint.Project)

	var middlewares []string
	middlewares = append(middlewares, forwardAuth)

	// 1. 注册受保护路由 (Protected Router)
	// 规则：基础路径前缀 + 精确匹配环境、集群、染色
	basePath := fmt.Sprintf("/%s/%s/%s/", endpoint.Company, endpoint.Project, endpoint.Service)
	protectedRule := fmt.Sprintf("PathPrefix(`%s`) && Header(`%s`, `%s`) && Header(`%s`, `%s`) && Header(`%s`, `%s`)",
		basePath, HeaderKeyEnv, endpoint.Env, HeaderKeyCluster, endpoint.Cluster, HeaderKeyColor, endpoint.Color)
	err := h.upsertRouter(ctx, endpoint, "", protectedRule, middlewares, 0)
	if err != nil {
		return err
	}

	// 2. 注册公开路由 (Public Router) - 如果有排除路径
	if len(opt.ExcludeAuthPaths) > 0 {
		// 规则：所有排除路径的集合 + 精确匹配环境、集群、染色
		pathRules := make([]string, 0, len(opt.ExcludeAuthPaths))
		for _, p := range opt.ExcludeAuthPaths {
			pathRules = append(pathRules, fmt.Sprintf("PathPrefix(`%s`)", p))
		}
		publicRule := fmt.Sprintf("(%s) && Header(`%s`, `%s`) && Header(`%s`, `%s`) && Header(`%s`, `%s`)",
			strings.Join(pathRules, " || "), HeaderKeyEnv, endpoint.Env, HeaderKeyCluster, endpoint.Cluster, HeaderKeyColor, endpoint.Color)

		// 剔除身份验证中间件 (ForwardAuth)
		publicMiddlewares := make([]string, 0)
		for _, m := range middlewares {
			if m != forwardAuth {
				publicMiddlewares = append(publicMiddlewares, m)
			}
		}

		// 设置更高的优先级
		err = h.upsertRouter(ctx, endpoint, "public", publicRule, publicMiddlewares, 1000)
		if err != nil {
			return err
		}
	}

	// 3. 设置 HealthCheck Path（公共配置，直接 Put）
	if opt.HealthCheckPath != "" {
		serviceHealthcheckPathKey := constructor.GenServiceHealthCheckPathKey(endpoint)
		err := h.store.Put(ctx, serviceHealthcheckPathKey, []byte(opt.HealthCheckPath))
		if err != nil {
			return err
		}
	}

	// 4. 检查并设置service url
	loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
	loadbalancerServerMap, err := h.store.GetByPrefix(ctx, loadbalancerServiceKeyPrefix)
	if err != nil {
		return err
	}

	currentMaxServicesURLIndex := -1
	serverURLExists := false
	reg, err := regexp.Compile("^" + loadbalancerServiceKeyPrefix + "[0-9]+/url$")
	if err != nil {
		return err
	}

	for k, v := range loadbalancerServerMap {
		if reg.MatchString(k) {
			tmp := strings.Replace(k, loadbalancerServiceKeyPrefix, "", 1)
			tmpSli := strings.Split(tmp, "/")
			idx, err := strconv.Atoi(tmpSli[0])
			if err != nil {
				return err
			}
			if idx > currentMaxServicesURLIndex {
				currentMaxServicesURLIndex = idx
			}

			serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
			if serverURL == string(v) || serverURL+"/" == string(v) {
				serverURLExists = true
				break
			}
		}
	}

	if !serverURLExists {
		currentMaxServicesURLIndex = currentMaxServicesURLIndex + 1
		serviceURLKey := constructor.GenServiceUrlKey(currentMaxServicesURLIndex, endpoint)

		// 使用 TTL（如果 endpoint.TTL > 0）
		if endpoint.TTL > 0 {
			err = h.store.Put(ctx, serviceURLKey, []byte(fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)), endpoint.TTL)
		} else {
			err = h.store.Put(ctx, serviceURLKey, []byte(fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)))
		}
		if err != nil {
			return err
		}

		// 设置权重（也使用 TTL）
		if endpoint.Weight > 0 {
			weightKey := constructor.GenServiceWeightKey(currentMaxServicesURLIndex, endpoint)
			if endpoint.TTL > 0 {
				err = h.store.Put(ctx, weightKey, []byte(strconv.Itoa(int(endpoint.Weight))), endpoint.TTL)
			} else {
				err = h.store.Put(ctx, weightKey, []byte(strconv.Itoa(int(endpoint.Weight))))
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// upsertRouter 封装路由创建逻辑
func (h *handler) upsertRouter(ctx context.Context, endpoint gateway.Endpoint, suffix string, rule string, middlewares []string, priority int) error {
	constructor := NewConstructor()

	// 1. 设置 Rule
	ruleKey := constructor.GenRouterRuleKey(endpoint, suffix)
	err := h.store.Put(ctx, ruleKey, []byte(rule))
	if err != nil {
		return err
	}

	// 2. 设置 Middlewares
	if len(middlewares) > 0 {
		for i, m := range middlewares {
			mKey := constructor.GenRouterMiddlewareKey(i, endpoint, suffix)
			err = h.store.Put(ctx, mKey, []byte(strings.TrimSpace(m)))
			if err != nil {
				return err
			}
		}
	}

	// 3. 设置 Entrypoints (固定 web 和 websecure)
	entrypoints := []string{"web", "websecure"}
	for i, ep := range entrypoints {
		err = h.store.Put(ctx, constructor.GenRouterEntrypointKey(i, endpoint, suffix), []byte(ep))
		if err != nil {
			return err
		}
	}

	// 4. 设置 Service 关联（使用服务逻辑标识，而非实例 ID）
	routerServiceKey := constructor.GenRouterServiceKey(endpoint, suffix)
	serviceName := constructor.GenServiceName(endpoint)
	err = h.store.Put(ctx, routerServiceKey, []byte(serviceName))
	if err != nil {
		return err
	}

	// 5. 设置 Priority
	if priority > 0 {
		priorityKey := constructor.GenRouterPriorityKey(endpoint, suffix)
		err = h.store.Put(ctx, priorityKey, []byte(strconv.Itoa(priority)))
		if err != nil {
			return err
		}
	}

	return nil
}

// Deregister 注销服务（完全清除所有相关配置）
func (h *handler) Deregister(ctx context.Context, endpoint gateway.Endpoint, opts ...HandlerOptions) error {
	constructor := NewConstructor()

	// 1. 查找所有服务实例
	loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
	loadbalancerServerMap, err := h.store.GetByPrefix(ctx, loadbalancerServiceKeyPrefix)
	if err != nil {
		if errors.Is(err, kv_store.ErrConnectionFailed) {
			return fmt.Errorf("kv store connection failed: %w", err)
		}
		// ErrKeyNotFound 说明已经被删除，可以忽略
		if !errors.Is(err, kv_store.ErrKeyNotFound) {
			return fmt.Errorf("failed to get service instances: %w", err)
		}
		// 已经没有配置，直接返回
		return nil
	}

	// 2. 查找并删除当前实例
	serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
	instanceIndex := -1
	for k, v := range loadbalancerServerMap {
		if strings.HasSuffix(k, "/url") {
			if serverURL == string(v) || serverURL+"/" == string(v) {
				tmp := strings.TrimPrefix(k, loadbalancerServiceKeyPrefix)
				tmpSli := strings.Split(tmp, "/")
				if len(tmpSli) > 0 {
					instanceIndex, _ = strconv.Atoi(tmpSli[0])
					break
				}
			}
		}
	}

	// 如果没有找到当前实例，直接返回
	if instanceIndex < 0 {
		return nil
	}

	// 删除当前实例的配置
	instancePrefix := constructor.GenServiceInstancePrefix(instanceIndex, endpoint)
	if err := h.store.DeleteByPrefix(ctx, instancePrefix); err != nil {
		if errors.Is(err, kv_store.ErrConnectionFailed) {
			return fmt.Errorf("kv store connection failed: %w", err)
		}
		if !errors.Is(err, kv_store.ErrKeyNotFound) {
			return fmt.Errorf("failed to delete instance config: %w", err)
		}
	}

	// 3. 判断是否为最后一个实例
	remainingInstances := 0
	for k := range loadbalancerServerMap {
		if strings.HasSuffix(k, "/url") {
			// 排除当前实例
			if !strings.HasPrefix(k, instancePrefix) {
				remainingInstances++
			}
		}
	}

	// 4. 如果是最后一个实例，删除所有配置
	if remainingInstances == 0 {
		// 删除所有 router 配置
		routerPrefix := constructor.GenRouterPrefixAll(endpoint)
		if err := h.store.DeleteByPrefix(ctx, routerPrefix); err != nil {
			if errors.Is(err, kv_store.ErrConnectionFailed) {
				return fmt.Errorf("kv store connection failed: %w", err)
			}
			if !errors.Is(err, kv_store.ErrKeyNotFound) {
				return fmt.Errorf("failed to delete router config: %w", err)
			}
		}

		// 删除整个 service 配置
		servicePrefix := constructor.GenServicePrefix(endpoint)
		if err := h.store.DeleteByPrefix(ctx, servicePrefix); err != nil {
			if errors.Is(err, kv_store.ErrConnectionFailed) {
				return fmt.Errorf("kv store connection failed: %w", err)
			}
			if !errors.Is(err, kv_store.ErrKeyNotFound) {
				return fmt.Errorf("failed to delete service config: %w", err)
			}
		}
	}

	return nil
}

// Update 更新服务配置（先删除旧配置，再重新注册）
func (h *handler) Update(ctx context.Context, endpoint gateway.Endpoint, opts ...HandlerOptions) error {
	constructor := NewConstructor()

	// 1. 删除所有 router 配置（protected + public）
	routerPrefix := constructor.GenRouterPrefixAll(endpoint)
	if err := h.store.DeleteByPrefix(ctx, routerPrefix); err != nil {
		if errors.Is(err, kv_store.ErrConnectionFailed) {
			return fmt.Errorf("kv store connection failed: %w", err)
		}
		if !errors.Is(err, kv_store.ErrKeyNotFound) {
			return fmt.Errorf("failed to delete old router config: %w", err)
		}
	}

	// 2. 查找并删除当前实例的 service 配置
	loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
	loadbalancerServerMap, err := h.store.GetByPrefix(ctx, loadbalancerServiceKeyPrefix)
	if err != nil {
		if errors.Is(err, kv_store.ErrConnectionFailed) {
			return fmt.Errorf("kv store connection failed: %w", err)
		}
		// ErrKeyNotFound 是正常的，说明之前没有配置
		if !errors.Is(err, kv_store.ErrKeyNotFound) {
			return fmt.Errorf("failed to get service instances: %w", err)
		}
		loadbalancerServerMap = make(map[string][]byte)
	}

	// 查找当前实例的索引
	serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
	instanceIndex := -1
	for k, v := range loadbalancerServerMap {
		if strings.HasSuffix(k, "/url") {
			if serverURL == string(v) || serverURL+"/" == string(v) {
				tmp := strings.TrimPrefix(k, loadbalancerServiceKeyPrefix)
				tmpSli := strings.Split(tmp, "/")
				if len(tmpSli) > 0 {
					instanceIndex, _ = strconv.Atoi(tmpSli[0])
					break
				}
			}
		}
	}

	// 如果找到了当前实例，删除它的配置
	if instanceIndex >= 0 {
		instancePrefix := constructor.GenServiceInstancePrefix(instanceIndex, endpoint)
		if err := h.store.DeleteByPrefix(ctx, instancePrefix); err != nil {
			if errors.Is(err, kv_store.ErrConnectionFailed) {
				return fmt.Errorf("kv store connection failed: %w", err)
			}
			if !errors.Is(err, kv_store.ErrKeyNotFound) {
				return fmt.Errorf("failed to delete instance config: %w", err)
			}
		}
	}

	// 3. 调用 Register 重新注册
	return h.Register(ctx, endpoint, opts...)
}

func (h *handler) Close() error {
	if h.store != nil {
		return h.store.Close()
	}
	return nil
}

// Refresh 续期服务实例配置的 TTL
func (h *handler) Refresh(ctx context.Context, endpoint gateway.Endpoint) error {
	if endpoint.TTL == 0 {
		return nil // 无 TTL，无需刷新
	}

	constructor := NewConstructor()

	// 1. 查找当前实例索引
	loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
	loadbalancerServerMap, err := h.store.GetByPrefix(ctx, loadbalancerServiceKeyPrefix)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
	instanceIndex := -1
	reg, err := regexp.Compile("^" + loadbalancerServiceKeyPrefix + "[0-9]+/url$")
	if err != nil {
		return err
	}

	for k, v := range loadbalancerServerMap {
		if reg.MatchString(k) {
			if serverURL == string(v) || serverURL+"/" == string(v) {
				tmp := strings.Replace(k, loadbalancerServiceKeyPrefix, "", 1)
				tmpSli := strings.Split(tmp, "/")
				if len(tmpSli) > 0 {
					instanceIndex, _ = strconv.Atoi(tmpSli[0])
					break
				}
			}
		}
	}

	if instanceIndex < 0 {
		return fmt.Errorf("instance not found, need re-register")
	}

	// 2. 收集需要续期的所有 key
	keys := []string{
		constructor.GenServiceUrlKey(instanceIndex, endpoint),
	}

	if endpoint.Weight > 0 {
		keys = append(keys, constructor.GenServiceWeightKey(instanceIndex, endpoint))
	}

	// 3. 批量续期，确保所有 key 的 TTL 同步
	return h.store.BatchKeepAlive(ctx, keys, endpoint.TTL)
}
