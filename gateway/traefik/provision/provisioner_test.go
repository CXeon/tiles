package provision

import (
	"context"
	"errors"
	"testing"

	"github.com/CXeon/tiles/gateway/traefik"
	"github.com/CXeon/tiles/gateway/traefik/kv_store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockKvStore is a testify-based mock for kv_store.KvStore.
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

// helper: build expected KV prefix for the given ForwardAuthConfig.
func middlewarePrefix(cfg ForwardAuthConfig) string {
	con := traefik.NewConstructor()
	name := con.MiddlewareName(cfg.Env, cfg.Cluster, cfg.Company, cfg.Project)
	return con.MiddlewareKeyPrefix(name)
}

// --- Task 5.1: SetForwardAuth tests ---

// TestSetForwardAuth_WritesCorrectKeys verifies that all required KV keys are written
// with the correct values on a normal SetForwardAuth call.
func TestSetForwardAuth_WritesCorrectKeys(t *testing.T) {
	store := new(mockKvStore)
	p := &Provisioner{store: store}

	cfg := ForwardAuthConfig{
		Env:                 "prod",
		Cluster:             "china",
		Company:             "acme",
		Project:             "billing",
		Address:             "http://10.0.0.1:8080",
		TrustForwardHeader:  true,
		AuthResponseHeaders: []string{"X-User-Id", "X-Role"},
	}

	prefix := middlewarePrefix(cfg)
	store.On("Put", prefix+"forwardAuth/address", []byte("http://10.0.0.1:8080")).Return(nil)
	store.On("Put", prefix+"forwardAuth/trustForwardHeader", []byte("true")).Return(nil)
	store.On("Put", prefix+"forwardAuth/authResponseHeaders/0", []byte("X-User-Id")).Return(nil)
	store.On("Put", prefix+"forwardAuth/authResponseHeaders/1", []byte("X-Role")).Return(nil)

	err := p.SetForwardAuth(context.Background(), cfg)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

// TestSetForwardAuth_Idempotent verifies that calling SetForwardAuth twice returns nil
// both times (idempotent overwrites).
func TestSetForwardAuth_Idempotent(t *testing.T) {
	store := new(mockKvStore)
	p := &Provisioner{store: store}

	cfg := ForwardAuthConfig{
		Env:     "dev",
		Cluster: "china",
		Company: "acme",
		Project: "api",
		Address: "http://10.0.0.2:9090",
	}

	prefix := middlewarePrefix(cfg)
	store.On("Put", prefix+"forwardAuth/address", []byte("http://10.0.0.2:9090")).Return(nil)
	store.On("Put", prefix+"forwardAuth/trustForwardHeader", []byte("false")).Return(nil)

	assert.NoError(t, p.SetForwardAuth(context.Background(), cfg))
	assert.NoError(t, p.SetForwardAuth(context.Background(), cfg))
	store.AssertNumberOfCalls(t, "Put", 4) // 2 keys × 2 calls
}

// TestSetForwardAuth_KVStoreError verifies that a KV error is propagated and no
// further writes are attempted.
func TestSetForwardAuth_KVStoreError(t *testing.T) {
	store := new(mockKvStore)
	p := &Provisioner{store: store}

	cfg := ForwardAuthConfig{
		Env: "prod", Cluster: "china", Company: "acme", Project: "svc",
		Address: "http://10.0.0.3:8080",
	}

	storeErr := kv_store.ErrConnectionFailed
	prefix := middlewarePrefix(cfg)
	store.On("Put", prefix+"forwardAuth/address", mock.Anything).Return(storeErr)

	err := p.SetForwardAuth(context.Background(), cfg)
	assert.Error(t, err)
	assert.ErrorIs(t, err, storeErr)
	// Only the first Put was attempted; no further writes.
	store.AssertNumberOfCalls(t, "Put", 1)
}

// --- Task 5.2: RemoveForwardAuth tests ---

// TestRemoveForwardAuth_DeletesPrefix verifies that DeleteByPrefix is called with
// the correct middleware key prefix.
func TestRemoveForwardAuth_DeletesPrefix(t *testing.T) {
	store := new(mockKvStore)
	p := &Provisioner{store: store}

	cfg := ForwardAuthConfig{
		Env: "prod", Cluster: "china", Company: "acme", Project: "billing",
	}
	prefix := middlewarePrefix(cfg)
	store.On("DeleteByPrefix", prefix).Return(nil)

	err := p.RemoveForwardAuth(context.Background(), cfg)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

// TestRemoveForwardAuth_NotFound_Idempotent verifies that removing a non-existent
// middleware returns nil (idempotent).
func TestRemoveForwardAuth_NotFound_Idempotent(t *testing.T) {
	store := new(mockKvStore)
	p := &Provisioner{store: store}

	cfg := ForwardAuthConfig{
		Env: "dev", Cluster: "us", Company: "acme", Project: "ghost",
	}
	prefix := middlewarePrefix(cfg)
	store.On("DeleteByPrefix", prefix).Return(kv_store.ErrKeyNotFound)

	err := p.RemoveForwardAuth(context.Background(), cfg)
	assert.NoError(t, err)
	store.AssertExpectations(t)
}

// TestRemoveForwardAuth_StoreError propagates non-NotFound errors.
func TestRemoveForwardAuth_StoreError(t *testing.T) {
	store := new(mockKvStore)
	p := &Provisioner{store: store}

	cfg := ForwardAuthConfig{
		Env: "prod", Cluster: "china", Company: "acme", Project: "svc",
	}
	prefix := middlewarePrefix(cfg)
	storeErr := errors.New("connection refused")
	store.On("DeleteByPrefix", prefix).Return(storeErr)

	err := p.RemoveForwardAuth(context.Background(), cfg)
	assert.Error(t, err)
	assert.ErrorIs(t, err, storeErr)
}

// --- Task 5.3: Middleware name format tests ---

// TestMiddlewareName_Format verifies the ForwardAuth middleware name follows the
// {Env}.{Cluster}.{Company}.{Project}.ForwardAuth format.
func TestMiddlewareName_Format(t *testing.T) {
	con := traefik.NewConstructor()

	tests := []struct {
		env, cluster, company, project string
		expected                       string
	}{
		{"prod", "china", "acme", "billing", "prod.china.acme.billing.ForwardAuth"},
		{"dev", "us", "testco", "api", "dev.us.testco.api.ForwardAuth"},
		{"staging", "eu", "corp", "frontend", "staging.eu.corp.frontend.ForwardAuth"},
	}

	for _, tt := range tests {
		name := con.MiddlewareName(tt.env, tt.cluster, tt.company, tt.project)
		assert.Equal(t, tt.expected, name)
	}
}

// TestMiddlewareKeyPrefix_Format verifies the KV key prefix format.
func TestMiddlewareKeyPrefix_Format(t *testing.T) {
	con := traefik.NewConstructor()

	name := con.MiddlewareName("prod", "china", "acme", "billing")
	prefix := con.MiddlewareKeyPrefix(name)
	assert.Equal(t, "traefik/http/middlewares/prod.china.acme.billing.ForwardAuth/", prefix)
}
