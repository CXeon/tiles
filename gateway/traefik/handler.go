package traefik

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/CXeon/tiles/gateway"
	"github.com/CXeon/tiles/gateway/traefik/kv_store"
)

type handler struct {
	ctx   context.Context
	store kv_store.KvStore
}

func NewHandler(ctx context.Context, provider *Provider) (*handler, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var store kv_store.KvStore
	var err error

	switch provider.KVType {
	case ProviderTypeRedis:
		store, err = kv_store.NewRedisStore(ctx, provider.Endpoints, provider.Password, provider.DBIndex)
	case ProviderTypeConsul:
		store, err = kv_store.NewConsulStore(ctx, provider.Endpoints, provider.Token)
	default:
		return nil, fmt.Errorf("unsupported traefik provider type: %v", provider.KVType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kv store: %w", err)
	}

	return &handler{
		ctx:   ctx,
		store: store,
	}, nil
}

func (h *handler) Register(endpoint gateway.Endpoint) error {

	constructor := NewConstructor()
	// 1.检查ruleKey是否存在，如果不存在新增，存在继续后面流程
	ruleKey := constructor.GenRouterRuleKey(endpoint)
	rule, err := h.store.Get(ruleKey)
	if err != nil {
		return err
	}
	if len(rule) == 0 {
		pathPrefix := fmt.Sprintf("/%s/%s/%s/", endpoint.Company, endpoint.Project, endpoint.Service)
		rulePathPrefix := fmt.Sprintf("PathPrefix(`%s`)", pathPrefix)

		ruleHeaderRegexp := fmt.Sprintf("HeaderRegexp(`X-Env`, `[A-Za-z0-9]`)&&HeaderRegexp(`X-Cluster`, `[A-Za-z0-9]`)")

		str := fmt.Sprintf("%s&&%s", rulePathPrefix, ruleHeaderRegexp)
		rule = []byte(str)
		err = h.store.Put(ruleKey, rule)
		if err != nil {
			return err
		}
	}

	// 2. 根据 option 设置 router middleware
	middlewares := endpoint.GetExtra("traefik.router.middlewares")
	if len(middlewares) > 0 {
		middlewareList := strings.Split(middlewares, ",")
		for i, m := range middlewareList {
			mKey := constructor.GenRouterMiddlewareKey(i, endpoint)
			err = h.store.Put(mKey, []byte(strings.TrimSpace(m)))
			if err != nil {
				return err
			}
		}
	}

	// 3.检查并设置router的entrypoint
	routerEntrypointPrefix := constructor.GenRouterEntrypointKeyPrefix(endpoint)
	routerEntrypointMap, err := h.store.GetByPrefix(routerEntrypointPrefix)
	if err != nil {
		return err
	}
	currentMaxRouterEntrypointIndex := -1
	defaultEntrypointWebExists, defaultEntrypointWebsecureExists := false, false

	for k, v := range routerEntrypointMap {
		slashIndex := strings.LastIndex(k, "/")
		numStr := k[slashIndex+1:]
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return err
		}
		if num > currentMaxRouterEntrypointIndex {
			currentMaxRouterEntrypointIndex = num
		}

		if strings.ToLower(string(v)) == "web" {
			defaultEntrypointWebExists = true
			continue
		}

		if strings.ToLower(string(v)) == "websecure" {
			defaultEntrypointWebsecureExists = true
			continue
		}

	}

	if !defaultEntrypointWebExists {
		currentMaxRouterEntrypointIndex = currentMaxRouterEntrypointIndex + 1
		err = h.store.Put(constructor.GenRouterEntrypointKey(currentMaxRouterEntrypointIndex, endpoint), []byte("web"))
		if err != nil {
			return err
		}
	}
	if !defaultEntrypointWebsecureExists {
		currentMaxRouterEntrypointIndex = currentMaxRouterEntrypointIndex + 1
		err = h.store.Put(constructor.GenRouterEntrypointKey(currentMaxRouterEntrypointIndex, endpoint), []byte("websecure"))
		if err != nil {
			return err
		}
	}

	// 4. 检查并设置router对应的service
	routerServiceKey := constructor.GenRouterServiceKey(endpoint)
	routerService, err := h.store.Get(routerServiceKey)
	if err != nil {
		return err
	}
	if len(routerService) == 0 {
		err = h.store.Put(routerServiceKey, []byte(endpoint.ID()))
		if err != nil {
			return err
		}
	}

	// 5. 根据 option 设置 service 权重
	// 移至步骤 6 处理

	// 6. 检查并设置service url
	loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
	loadbalancerServerMap, err := h.store.GetByPrefix(loadbalancerServiceKeyPrefix)
	if err != nil {
		return err
	}

	currentMaxServicesURLIndex := -1
	serverURLExists := false
	reg, err := regexp.Compile("^" + loadbalancerServiceKeyPrefix + "[0-9]+/url&")
	if err != nil {
		return err
	}

	for k, v := range loadbalancerServerMap {
		if reg.MatchString(k) {
			tmp := strings.Replace(k, loadbalancerServiceKeyPrefix, "", 1)
			tmpSli := strings.Split(tmp, "/")
			currentMaxServicesURLIndex, err = strconv.Atoi(tmpSli[0])
			if err != nil {
				return err
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
		err = h.store.Put(serviceURLKey, []byte(fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)))
		if err != nil {
			return err
		}

		// 设置权重
		weight := endpoint.GetExtra("traefik.service.weight")
		if len(weight) > 0 {
			weightKey := constructor.GenServiceWeightKey(currentMaxServicesURLIndex, endpoint)
			err = h.store.Put(weightKey, []byte(weight))
			if err != nil {
				return err
			}
		}

		serviceHealthcheckPathKey := constructor.GenServiceHealthCheckPathKey(endpoint)

		healthcheckPath := endpoint.GetExtra("traefik.service.healthcheck.path")
		if len(healthcheckPath) == 0 {
			healthcheckPath = "/health"
		}
		err = h.store.Put(serviceHealthcheckPathKey, []byte(healthcheckPath))
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) Deregister(endpoint gateway.Endpoint) error {
	constructor := NewConstructor()

	// 1. 删除 service url
	loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
	loadbalancerServerMap, err := h.store.GetByPrefix(loadbalancerServiceKeyPrefix)
	if err != nil {
		return err
	}

	serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
	for k, v := range loadbalancerServerMap {
		if strings.HasSuffix(k, "/url") {
			if serverURL == string(v) || serverURL+"/" == string(v) {
				// 获取索引，例如 .../servers/0/url -> 0
				tmp := strings.TrimPrefix(k, loadbalancerServiceKeyPrefix)
				tmpSli := strings.Split(tmp, "/")
				if len(tmpSli) > 0 {
					indexStr := tmpSli[0]
					index, _ := strconv.Atoi(indexStr)

					// 删除该索引下的所有相关 key
					err = h.store.Delete(k)
					if err != nil {
						return err
					}

					// 尝试删除 weight 和 preservePath
					_ = h.store.Delete(constructor.GenServiceWeightKey(index, endpoint))
					_ = h.store.Delete(constructor.GenServicePreservePathKey(index, endpoint))
				}
			}
		}
	}

	return nil
}

func (h *handler) Update(endpoint gateway.Endpoint) error {
	// Update 逻辑目前和 Register 类似，因为都是幂等操作
	return h.Register(endpoint)
}

func (h *handler) Close() error {
	if h.store != nil {
		return h.store.Close()
	}
	return nil
}
