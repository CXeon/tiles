package http_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httprpc "github.com/CXeon/tiles/rpc/http"
)

func TestNew_RequiresBaseURLOrResolver(t *testing.T) {
	_, err := httprpc.New(httprpc.Config{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BaseURL and resolver cannot both be empty")
}

func TestNew_RequiresServiceWhenResolverSet(t *testing.T) {
	resolver := &stubResolver{addr: "http://localhost:8080"}
	_, err := httprpc.New(httprpc.Config{}, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service name is required")
}

func TestNew_SucceedsWithBaseURL(t *testing.T) {
	c, err := httprpc.New(httprpc.Config{BaseURL: "http://localhost"}, nil)
	require.NoError(t, err)
	require.NotNil(t, c)
}

// stubResolver 实现 rpc.Resolver 接口，用于测试
type stubResolver struct {
	addr string
	err  error
}

func (s *stubResolver) Resolve(_ context.Context, _ string) (string, error) {
	return s.addr, s.err
}
