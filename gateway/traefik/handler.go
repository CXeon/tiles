package traefik

import (
	"context"
	"fmt"
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
	// TODO 根据类型创建对应store实现
	if ctx == nil {
		ctx = context.Background()
	}
	return &handler{
		ctx:   ctx,
		store: nil,
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

	// 2. TODO 根据option设置middleware

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

		if string(v) == "web" {
			defaultEntrypointWebExists = true
			continue
		}

		if string(v) == "websecure" {
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

	// 5. TODO 根据option设置权重

	// 6. 检查并设置service url
	loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
	loadbalancerServerMap, err := h.store.GetByPrefix(loadbalancerServiceKeyPrefix)
	if err != nil {
		return err
	}

	currentMaxServicesIndex := -1

}
