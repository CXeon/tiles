package context

import "context"
import "github.com/gin-gonic/gin"

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
	if ginCtx, ok := ctx.(*gin.Context); ok {
		ent.TraceID = ginCtx.Request.Header.Get(traceID)
		ent.Env = ginCtx.Request.Header.Get(env)
		ent.Cluster = ginCtx.Request.Header.Get(cluster)
		ent.UserID = ginCtx.Request.Header.Get(userID)

		appCtx.ent = ent
		return appCtx
	}

	if aCtx, ok := ctx.(*AppContext); ok {
		// TODO get方法
	}

	// 默认ctx的默认获取方法

}
