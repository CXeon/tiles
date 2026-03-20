package context

import (
	"context"
	"maps"
	"net/http"
)

// contextKey 是专用类型，避免与其他包的 context key 碰撞
type contextKey int

const (
	keyTraceID contextKey = iota
	keyEnv
	keyCluster
	keyUserID
	keyColor
)

// HTTP Header 名称，供网关和客户端参考
const (
	HeaderTraceID = "X-Trace-Id"
	HeaderEnv     = "X-Env"
	HeaderCluster = "X-Cluster"
	HeaderUserID  = "X-User-Id"
	HeaderColor   = "X-Color"
)

type AppContext struct {
	context.Context
	ent *entity
}

type entity struct {
	TraceID string         `json:"trace_id"`
	Env     string         `json:"env"`
	Cluster string         `json:"cluster"`
	UserID  string         `json:"user_id"`
	Color   string         `json:"color"`
	Extra   map[string]any `json:"extra"`
}

// NewFromHTTPHeaders 从 HTTP Header 中提取 tiles 字段，构造 AppContext。
// 供中间件使用：appCtx := tilecontext.NewFromHTTPHeaders(c.Request.Context(), c.Request.Header)
func NewFromHTTPHeaders(ctx context.Context, headers http.Header) *AppContext {
	if ctx == nil {
		ctx = context.Background()
	}
	ent := &entity{
		TraceID: headers.Get(HeaderTraceID),
		Env:     headers.Get(HeaderEnv),
		Cluster: headers.Get(HeaderCluster),
		UserID:  headers.Get(HeaderUserID),
		Color:   headers.Get(HeaderColor),
		Extra:   make(map[string]any),
	}
	return &AppContext{Context: ctx, ent: ent}
}

// NewAppContext 从已有 context 构造 AppContext，用于在服务内部传递时保留字段。
// 若传入的已经是 *AppContext，则直接复制其字段。
func NewAppContext(ctx context.Context) *AppContext {
	if ctx == nil {
		ctx = context.Background()
	}

	ent := &entity{Extra: make(map[string]any)}

	if aCtx, ok := ctx.(*AppContext); ok {
		ent.TraceID = aCtx.ent.TraceID
		ent.Env = aCtx.ent.Env
		ent.Cluster = aCtx.ent.Cluster
		ent.UserID = aCtx.ent.UserID
		ent.Color = aCtx.ent.Color
		if aCtx.ent.Extra != nil {
			maps.Copy(ent.Extra, aCtx.ent.Extra)
		}
	}

	return &AppContext{Context: ctx, ent: ent}
}

// From 从 context 中提取 AppContext。若不是 *AppContext，返回空的 AppContext。
// 供 handler 使用：appCtx := tilecontext.From(c.Request.Context())
func From(ctx context.Context) *AppContext {
	if appCtx, ok := ctx.(*AppContext); ok {
		return appCtx
	}
	return NewAppContext(ctx)
}

// Value 优先从 ent 返回已知字段，其余委托给底层 context。
func (appCtx *AppContext) Value(key any) any {
	switch key {
	case keyTraceID:
		return appCtx.ent.TraceID
	case keyEnv:
		return appCtx.ent.Env
	case keyCluster:
		return appCtx.ent.Cluster
	case keyUserID:
		return appCtx.ent.UserID
	case keyColor:
		return appCtx.ent.Color
	default:
		return appCtx.Context.Value(key)
	}
}

func (appCtx *AppContext) TraceID() string {
	return appCtx.ent.TraceID
}

func (appCtx *AppContext) Env() string {
	return appCtx.ent.Env
}

func (appCtx *AppContext) Cluster() string {
	return appCtx.ent.Cluster
}

func (appCtx *AppContext) UserID() string {
	return appCtx.ent.UserID
}

func (appCtx *AppContext) Color() string {
	return appCtx.ent.Color
}

func (appCtx *AppContext) Extra(key string) any {
	return appCtx.ent.Extra[key]
}

func (appCtx *AppContext) SetExtra(key string, value any) {
	appCtx.ent.Extra[key] = value
}
