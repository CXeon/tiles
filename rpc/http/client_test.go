package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CXeon/tiles/rpc"
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

// ── 测试辅助 ──────────────────────────────────────────────────────────────────

type userReq struct {
	Name string `json:"name"`
}

type userResp struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// newSuccessServer 返回模拟成功响应（Code=0）的测试服务器
func newSuccessServer(t *testing.T, data any, assertFn func(r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if assertFn != nil {
			assertFn(r)
		}
		dataBytes, _ := json.Marshal(data)
		resp := map[string]any{
			"code":    0,
			"message": "ok",
			"data":    json.RawMessage(dataBytes),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newBizErrServer 返回模拟业务错误（HTTP 200, Code != 0）的测试服务器
func newBizErrServer(t *testing.T, code uint, message, traceID string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"code":     code,
			"message":  message,
			"trace_id": traceID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ── 成功路径 ──────────────────────────────────────────────────────────────────

func TestInvoke_POST_Success(t *testing.T) {
	srv := newSuccessServer(t, userResp{ID: 1, Name: "Alice"}, func(r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/users", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var body userReq
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "Alice", body.Name)
	})

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	var resp userResp
	err = c.Invoke(context.Background(), http.MethodPost, &userReq{Name: "Alice"}, &resp,
		httprpc.WithPath("/api/users"),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.ID)
	assert.Equal(t, "Alice", resp.Name)
}

func TestInvoke_GET_NoBody(t *testing.T) {
	srv := newSuccessServer(t, userResp{ID: 2, Name: "Bob"}, func(r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/users/2", r.URL.Path)
	})

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	var resp userResp
	err = c.Invoke(context.Background(), http.MethodGet, nil, &resp,
		httprpc.WithPath("/api/users/2"),
	)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.ID)
}

func TestInvoke_NilResp_IgnoresData(t *testing.T) {
	srv := newSuccessServer(t, userResp{ID: 3, Name: "Carol"}, nil)

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/api/users/3"),
	)
	require.NoError(t, err)
}

// ── 错误路径 ──────────────────────────────────────────────────────────────────

func TestInvoke_ErrPathRequired(t *testing.T) {
	c, err := httprpc.New(httprpc.Config{BaseURL: "http://localhost"}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil)
	require.ErrorIs(t, err, rpc.ErrPathRequired)
}

func TestInvoke_HTTPError_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	t.Cleanup(srv.Close)

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/missing"),
	)
	var httpErr *httprpc.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, uint(404), httpErr.Code)
	assert.Equal(t, "not found", httpErr.Message)
}

func TestInvoke_HTTPError_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	t.Cleanup(srv.Close)

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodPost, nil, nil,
		httprpc.WithPath("/crash"),
	)
	var httpErr *httprpc.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, uint(500), httpErr.Code)
}

func TestInvoke_ResponseError(t *testing.T) {
	srv := newBizErrServer(t, 1001, "resource not found", "trace-xyz")

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	var resp userResp
	err = c.Invoke(context.Background(), http.MethodGet, nil, &resp,
		httprpc.WithPath("/api/users/99"),
	)
	var respErr *rpc.ResponseError
	require.ErrorAs(t, err, &respErr)
	assert.Equal(t, uint(1001), respErr.Code)
	assert.Equal(t, "resource not found", respErr.Message)
	assert.Equal(t, "trace-xyz", respErr.TraceID)
}

// ── TraceID ───────────────────────────────────────────────────────────────────

func TestInvoke_WithTraceID(t *testing.T) {
	var gotTraceID string
	srv := newSuccessServer(t, nil, func(r *http.Request) {
		gotTraceID = r.Header.Get("X-Trace-ID")
	})

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/api/ping"),
		httprpc.WithTraceID("explicit-trace"),
	)
	require.NoError(t, err)
	assert.Equal(t, "explicit-trace", gotTraceID)
}

func TestInvoke_TraceIDExtractor(t *testing.T) {
	var gotTraceID string
	srv := newSuccessServer(t, nil, func(r *http.Request) {
		gotTraceID = r.Header.Get("X-Trace-ID")
	})

	c, err := httprpc.New(httprpc.Config{
		BaseURL: srv.URL,
		TraceIDExtractor: func(_ context.Context) string {
			return "extracted-trace"
		},
	}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/api/ping"),
	)
	require.NoError(t, err)
	assert.Equal(t, "extracted-trace", gotTraceID)
}

func TestInvoke_WithTraceID_OverridesExtractor(t *testing.T) {
	var gotTraceID string
	srv := newSuccessServer(t, nil, func(r *http.Request) {
		gotTraceID = r.Header.Get("X-Trace-ID")
	})

	c, err := httprpc.New(httprpc.Config{
		BaseURL: srv.URL,
		TraceIDExtractor: func(_ context.Context) string {
			return "extractor-trace"
		},
	}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/api/ping"),
		httprpc.WithTraceID("override-trace"),
	)
	require.NoError(t, err)
	assert.Equal(t, "override-trace", gotTraceID)
}

// ── 其他 CallOption ───────────────────────────────────────────────────────────

func TestInvoke_WithHeader(t *testing.T) {
	var gotAuth string
	srv := newSuccessServer(t, nil, func(r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	})

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/api/secure"),
		httprpc.WithHeader("Authorization", "Bearer token123"),
	)
	require.NoError(t, err)
	assert.Equal(t, "Bearer token123", gotAuth)
}

func TestInvoke_WithQuery(t *testing.T) {
	var gotPage string
	srv := newSuccessServer(t, nil, func(r *http.Request) {
		gotPage = r.URL.Query().Get("page")
	})

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/api/users"),
		httprpc.WithQuery("page", "2"),
	)
	require.NoError(t, err)
	assert.Equal(t, "2", gotPage)
}

func TestInvoke_WithTimeout_PerCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := httprpc.New(httprpc.Config{BaseURL: srv.URL}, nil)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/slow"),
		httprpc.WithTimeout(10*time.Millisecond),
	)
	require.Error(t, err)
}

func TestInvoke_Resolver(t *testing.T) {
	srv := newSuccessServer(t, userResp{ID: 5, Name: "Eve"}, nil)

	resolver := &stubResolver{addr: srv.URL}
	c, err := httprpc.New(httprpc.Config{Service: "user-service"}, resolver)
	require.NoError(t, err)

	var resp userResp
	err = c.Invoke(context.Background(), http.MethodGet, nil, &resp,
		httprpc.WithPath("/api/users/5"),
	)
	require.NoError(t, err)
	assert.Equal(t, 5, resp.ID)
}

func TestInvoke_ResolverError(t *testing.T) {
	resolver := &stubResolver{err: errors.New("service unavailable")}
	c, err := httprpc.New(httprpc.Config{Service: "user-service"}, resolver)
	require.NoError(t, err)

	err = c.Invoke(context.Background(), http.MethodGet, nil, nil,
		httprpc.WithPath("/api/users"),
	)
	require.ErrorIs(t, err, rpc.ErrResolverFailed)
}

func TestClose_ReturnsNil(t *testing.T) {
	c, err := httprpc.New(httprpc.Config{BaseURL: "http://localhost"}, nil)
	require.NoError(t, err)
	assert.NoError(t, c.Close(context.Background()))
}
