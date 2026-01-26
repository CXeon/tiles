package traefik

import (
	"context"
	"testing"

	"github.com/CXeon/tiles/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockKvStore struct {
	mock.Mock
}

func (m *mockKvStore) Put(key string, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *mockKvStore) Add(key string, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *mockKvStore) Get(key string) ([]byte, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockKvStore) GetByPrefix(prefix string) (map[string][]byte, error) {
	args := m.Called(prefix)
	return args.Get(0).(map[string][]byte), args.Error(1)
}

func (m *mockKvStore) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *mockKvStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestTraefikClient_Register(t *testing.T) {
	mockStore := new(mockKvStore)
	h := &handler{
		ctx:   context.Background(),
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
	}

	// HTTPS should be normalized to HTTP
	// Check constructor key generation
	constructor := NewConstructor()
	ruleKey := constructor.GenRouterRuleKey(gateway.Endpoint{
		Env:      "dev",
		Cluster:  "china",
		Company:  "testco",
		Project:  "testprj",
		Service:  "testsvc",
		Protocol: gateway.ProtocolTypeHttp, // Expecting normalized protocol
	})

	mockStore.On("Get", ruleKey).Return([]byte{}, nil)
	mockStore.On("Get", mock.Anything).Return([]byte{}, nil)
	mockStore.On("Put", mock.Anything, mock.Anything).Return(nil)
	mockStore.On("GetByPrefix", mock.Anything).Return(map[string][]byte{}, nil)

	err := client.Register(context.Background(), endpoint)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTraefikClient_Deregister(t *testing.T) {
	mockStore := new(mockKvStore)
	h := &handler{
		ctx:   context.Background(),
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
	}

	constructor := NewConstructor()
	prefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(*endpoint)

	mockStore.On("GetByPrefix", prefix).Return(map[string][]byte{
		prefix + "0/url": []byte("http://127.0.0.1:8080"),
	}, nil)
	mockStore.On("Delete", prefix+"0/url").Return(nil)
	mockStore.On("Delete", mock.Anything).Return(nil)

	err := client.Deregister(context.Background(), endpoint)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
