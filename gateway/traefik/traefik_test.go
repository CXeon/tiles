package traefik

import (
	"context"
	"fmt"
	"testing"

	"github.com/CXeon/tiles/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockKvStore struct {
	mock.Mock
}

func (m *mockKvStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *mockKvStore) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockKvStore) GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	args := m.Called(prefix)
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (m *mockKvStore) Delete(ctx context.Context, key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *mockKvStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	args := m.Called(prefix)
	return args.Error(0)
}

func (m *mockKvStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *mockKvStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
	args := m.Called(keys)
	return args.Error(0)
}

func (m *mockKvStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestTraefikClient_Register(t *testing.T) {
	mockStore := new(mockKvStore)
	h := &handler{
		store: mockStore,
	}
	client := &traefikClient{
		handler: h,
	}

	endpoint := &gateway.Endpoint{
		Env:      "dev",
		Cluster:  "china",
		Company:  "testco",
		Project:  "testprj",
		Service:  "testsvc",
		Protocol: gateway.ProtocolTypeHttps,
		Ip:       "127.0.0.1",
		Port:     8080,
		Color:    "blue",
	}

	// HTTPS should be normalized to HTTP
	// Check constructor key generation
	constructor := NewConstructor()
	_ = constructor.GenRouterRuleKey(gateway.Endpoint{
		Env:      "dev",
		Cluster:  "china",
		Company:  "testco",
		Project:  "testprj",
		Service:  "testsvc",
		Protocol: gateway.ProtocolTypeHttp, // Expecting normalized protocol
		Color:    "blue",
	})

	// 预期行为：
	// 1. 注册受保护路由
	normalizedEndpoint := *endpoint
	normalizedEndpoint.Protocol = gateway.ProtocolTypeHttp

	expectedMiddleware := constructor.MiddlewareName("dev", "china", "testco", "testprj")

	protectedRule := fmt.Sprintf("PathPrefix(`/testco/testprj/testsvc/`) && Header(`%s`, `dev`) && Header(`%s`, `china`) && Header(`%s`, `blue`)", HeaderKeyEnv, HeaderKeyCluster, HeaderKeyColor)
	mockStore.On("Put", constructor.GenRouterRuleKey(normalizedEndpoint, ""), []byte(protectedRule)).Return(nil)
	mockStore.On("Put", constructor.GenRouterMiddlewareKey(0, normalizedEndpoint, ""), []byte(expectedMiddleware)).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(0, normalizedEndpoint, ""), []byte("web")).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(1, normalizedEndpoint, ""), []byte("websecure")).Return(nil)
	// Service Name 应该是逻辑服务标识，而非 Instance ID
	expectedServiceName := "dev.china.testco.testprj.testsvc.http.blue"
	mockStore.On("Put", constructor.GenRouterServiceKey(normalizedEndpoint, ""), []byte(expectedServiceName)).Return(nil)

	// 其他必要 Mock
	mockStore.On("GetByPrefix", constructor.GenServiceLoadbalancerServiceKeyPrefix(normalizedEndpoint)).Return(map[string][]byte{}, nil)
	mockStore.On("Put", mock.Anything, mock.Anything).Return(nil)

	err := client.Register(context.Background(), endpoint)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTraefikClient_ExcludeAuthPaths(t *testing.T) {
	mockStore := new(mockKvStore)
	h := &handler{
		store: mockStore,
	}
	client := &traefikClient{
		handler:          h,
		excludeAuthPaths: []string{"/testco/testprj/svc/public", "/testco/testprj/svc/health"},
	}

	endpoint := &gateway.Endpoint{
		Env:      "dev",
		Cluster:  "china",
		Company:  "testco",
		Project:  "testprj",
		Service:  "svc",
		Protocol: gateway.ProtocolTypeHttp,
		Ip:       "127.0.0.1",
		Port:     8080,
		Color:    "blue",
	}

	constructor := NewConstructor()
	expectedMiddleware := constructor.MiddlewareName("dev", "china", "testco", "testprj")

	// 首先设置通配符，作为fallback
	mockStore.On("GetByPrefix", mock.Anything).Return(map[string][]byte{}, nil)
	mockStore.On("Put", mock.Anything, mock.Anything).Return(nil)

	// 1. 注册受保护路由 (默认)
	protectedRule := fmt.Sprintf("PathPrefix(`/testco/testprj/svc/`) && Header(`%s`, `dev`) && Header(`%s`, `china`) && Header(`%s`, `blue`)", HeaderKeyEnv, HeaderKeyCluster, HeaderKeyColor)
	mockStore.On("Put", constructor.GenRouterRuleKey(*endpoint, ""), []byte(protectedRule)).Return(nil)
	mockStore.On("Put", constructor.GenRouterMiddlewareKey(0, *endpoint, ""), []byte(expectedMiddleware)).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(0, *endpoint, ""), []byte("web")).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(1, *endpoint, ""), []byte("websecure")).Return(nil)
	// Service Name 应该是逻辑服务标识
	expectedServiceName := "dev.china.testco.testprj.svc.http.blue"
	mockStore.On("Put", constructor.GenRouterServiceKey(*endpoint, ""), []byte(expectedServiceName)).Return(nil)

	// 2. 注册公开路由
	// 注意拼接后的路径：/testco/testprj/svc/public 和 /testco/testprj/svc/health
	publicRule := fmt.Sprintf("(PathPrefix(`/testco/testprj/svc/public`) || PathPrefix(`/testco/testprj/svc/health`)) && Header(`%s`, `dev`) && Header(`%s`, `china`) && Header(`%s`, `blue`)", HeaderKeyEnv, HeaderKeyCluster, HeaderKeyColor)
	mockStore.On("Put", constructor.GenRouterRuleKey(*endpoint, "public"), []byte(publicRule)).Return(nil)
	mockStore.On("Put", constructor.GenRouterPriorityKey(*endpoint, "public"), []byte("1000")).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(0, *endpoint, "public"), []byte("web")).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(1, *endpoint, "public"), []byte("websecure")).Return(nil)
	// Service Name 应该是逻辑服务标识
	mockStore.On("Put", constructor.GenRouterServiceKey(*endpoint, "public"), []byte(expectedServiceName)).Return(nil)

	err := client.Register(context.Background(), endpoint)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTraefikClient_Deregister(t *testing.T) {
	mockStore := new(mockKvStore)
	h := &handler{
		store: mockStore,
	}
	client := &traefikClient{
		handler: h,
	}

	endpoint := &gateway.Endpoint{
		Env:      "dev",
		Cluster:  "china",
		Company:  "testco",
		Project:  "testprj",
		Service:  "testsvc",
		Protocol: gateway.ProtocolTypeHttp,
		Ip:       "127.0.0.1",
		Port:     8080,
		Color:    "blue",
	}

	constructor := NewConstructor()
	prefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(*endpoint)

	// Mock: 返回单个实例（最后一个实例）
	mockStore.On("GetByPrefix", prefix).Return(map[string][]byte{
		prefix + "0/url": []byte("http://127.0.0.1:8080"),
	}, nil)

	// Mock: 删除当前实例的配置
	instancePrefix := constructor.GenServiceInstancePrefix(0, *endpoint)
	mockStore.On("DeleteByPrefix", instancePrefix).Return(nil)

	// Mock: 删除所有 router 配置（因为是最后一个实例）
	routerPrefix := constructor.GenRouterPrefixAll(*endpoint)
	mockStore.On("DeleteByPrefix", routerPrefix).Return(nil)

	// Mock: 删除整个 service 配置
	servicePrefix := constructor.GenServicePrefix(*endpoint)
	mockStore.On("DeleteByPrefix", servicePrefix).Return(nil)

	err := client.Deregister(context.Background(), endpoint)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTraefikClient_RegisterWithOptions(t *testing.T) {
	mockStore := new(mockKvStore)
	h := &handler{
		store: mockStore,
	}
	client := &traefikClient{
		handler:         h,
		healthCheckPath: "/ping",
	}

	endpoint := &gateway.Endpoint{
		Env:      "dev",
		Cluster:  "china",
		Company:  "testco",
		Project:  "testprj",
		Service:  "testsvc",
		Protocol: gateway.ProtocolTypeHttp,
		Ip:       "127.0.0.1",
		Port:     8080,
		Weight:   50, // 设置Weight字段
		Color:    "blue",
	}

	constructor := NewConstructor()
	expectedMiddleware := constructor.MiddlewareName("dev", "china", "testco", "testprj")

	// 预期行为：
	// 1. 设置中间件 ForwardAuth（从 endpoint 动态计算）
	mockStore.On("Put", constructor.GenRouterRuleKey(*endpoint, ""), mock.Anything).Return(nil)
	mockStore.On("Put", constructor.GenRouterMiddlewareKey(0, *endpoint, ""), []byte(expectedMiddleware)).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(0, *endpoint, ""), []byte("web")).Return(nil)
	mockStore.On("Put", constructor.GenRouterEntrypointKey(1, *endpoint, ""), []byte("websecure")).Return(nil)
	// Service Name 应该是逻辑服务标识
	expectedServiceName := "dev.china.testco.testprj.testsvc.http.blue"
	mockStore.On("Put", constructor.GenRouterServiceKey(*endpoint, ""), []byte(expectedServiceName)).Return(nil)

	// 2. 设置权重 50
	// 权重是在设置 URL 时设置的
	mockStore.On("GetByPrefix", constructor.GenServiceLoadbalancerServiceKeyPrefix(*endpoint)).Return(map[string][]byte{}, nil)
	mockStore.On("Put", constructor.GenServiceUrlKey(0, *endpoint), []byte("http://127.0.0.1:8080")).Return(nil)
	mockStore.On("Put", constructor.GenServiceWeightKey(0, *endpoint), []byte("50")).Return(nil)

	// 3. 设置健康检查路径 /ping
	mockStore.On("Put", constructor.GenServiceHealthCheckPathKey(*endpoint), []byte("/ping")).Return(nil)

	err := client.Register(context.Background(), endpoint)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

// TestNormalizeEndpoint 测试 endpoint 归一化函数
func TestNormalizeEndpoint(t *testing.T) {
	t.Run("nil endpoint", func(t *testing.T) {
		_, err := normalizeEndpoint(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint is nil")
	})

	t.Run("invalid protocol", func(t *testing.T) {
		endpoint := &gateway.Endpoint{
			Protocol: "invalid",
		}
		_, err := normalizeEndpoint(endpoint)
		assert.Error(t, err)
	})

	t.Run("HTTPS to HTTP conversion", func(t *testing.T) {
		endpoint := &gateway.Endpoint{
			Company:  "test",
			Project:  "proj",
			Service:  "svc",
			Protocol: gateway.ProtocolTypeHttps,
			Ip:       "127.0.0.1",
			Port:     8080,
		}
		normalized, err := normalizeEndpoint(endpoint)
		assert.NoError(t, err)
		assert.Equal(t, string(gateway.ProtocolTypeHttp), string(normalized.Protocol))
		// 确保原始 endpoint 没有被修改
		assert.Equal(t, string(gateway.ProtocolTypeHttps), string(endpoint.Protocol))
	})

	t.Run("normal HTTP endpoint", func(t *testing.T) {
		endpoint := &gateway.Endpoint{
			Company:  "test",
			Project:  "proj",
			Service:  "svc",
			Protocol: gateway.ProtocolTypeHttp,
			Ip:       "127.0.0.1",
			Port:     8080,
		}
		normalized, err := normalizeEndpoint(endpoint)
		assert.NoError(t, err)
		assert.Equal(t, string(gateway.ProtocolTypeHttp), string(normalized.Protocol))
	})
}

// TestBuildHandlerOptions 测试 HandlerOptions 构造
func TestBuildHandlerOptions(t *testing.T) {
	t.Run("with complete paths", func(t *testing.T) {
		client := &traefikClient{
			excludeAuthPaths: []string{"/company/project/service/public", "/company/project/service/health"},
			healthCheckPath:  "/ping",
		}
		opts := client.buildHandlerOptions()
		assert.Equal(t, []string{"/company/project/service/public", "/company/project/service/health"}, opts.ExcludeAuthPaths)
		assert.Equal(t, "/ping", opts.HealthCheckPath)
	})

	t.Run("with empty paths", func(t *testing.T) {
		client := &traefikClient{
			healthCheckPath: "/health",
		}
		opts := client.buildHandlerOptions()
		assert.Empty(t, opts.ExcludeAuthPaths)
		assert.Equal(t, "/health", opts.HealthCheckPath)
	})
}
