package context

import (
	"context"
	"maps"
)

const (
	traceID = "X-Trace-Id"
	env     = "X-Env"
	cluster = "X-Cluster"
	userID  = "X-User-Id"
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
	Extra   map[string]any `json:"extra"`
}

func NewAppContext(ctx context.Context) *AppContext {
	if ctx == nil {
		ctx = context.Background()
	}

	appCtx := &AppContext{Context: ctx}
	ent := &entity{}
	ent.Extra = make(map[string]any)

	if aCtx, ok := ctx.(*AppContext); ok {
		ent.TraceID = aCtx.ent.TraceID
		ent.Env = aCtx.ent.Env
		ent.Cluster = aCtx.ent.Cluster
		ent.UserID = aCtx.ent.UserID
		if aCtx.ent.Extra != nil {
			maps.Copy(ent.Extra, aCtx.ent.Extra)
		}

		appCtx.ent = ent
		return appCtx
	}

	// 默认ctx的默认获取方法

	traceIDInter := ctx.Value(traceID)
	if traceIDInter != nil {

		traceIDStr, ok := traceIDInter.(string)
		if ok {
			ent.TraceID = traceIDStr
		}
	}
	envInter := ctx.Value(env)
	if envInter != nil {
		envStr, ok := envInter.(string)
		if ok {
			ent.Env = envStr
		}
	}
	clusterInter := ctx.Value(cluster)
	if clusterInter != nil {
		clusterStr, ok := clusterInter.(string)
		if ok {
			ent.Cluster = clusterStr
		}
	}
	userIDInter := ctx.Value(userID)
	if userIDInter != nil {
		userIDStr, ok := userIDInter.(string)
		if ok {
			ent.UserID = userIDStr
		}
	}

	appCtx.ent = ent
	return appCtx

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

func (appCtx *AppContext) Extra(key string) any {
	return appCtx.ent.Extra[key]
}
