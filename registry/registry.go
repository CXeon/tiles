package registry

import (
	"context"
	"errors"
)

// 标准错误变量
var (
	ErrServiceNotFound       = errors.New("service not found in cache")
	ErrNotRegistered         = errors.New("service not registered")
	ErrInvalidEndpoint       = errors.New("endpoint is nil or invalid")
	ErrEmptyService          = errors.New("service name cannot be empty")
	ErrEmptyServices         = errors.New("services is empty")
	ErrEmptyCompanyOrProject = errors.New("company or project cannot be empty")
	ErrCurrentEndpointNil    = errors.New("currentEndpoint is nil, cannot determine isolation context")
	ErrComProjEmpty          = errors.New("ComProj cannot be empty")
)

type Client interface {
	Register(ctx context.Context, endpoint *Endpoint) error

	Deregister(ctx context.Context, endpoint *Endpoint) error

	// Discover 一次性拉取指定服务的实例列表
	Discover(ctx context.Context, service []string, option ...ServiceOption) (CompanyRegistry, error)

	// Watch 订阅指定服务的实例变更
	Watch(ctx context.Context, service []string, option ...ServiceOption) error

	// Update(ctx context.Context, endpoint *Endpoint) error

	GetService(ctx context.Context, service string, option ...GetServiceOption) (Endpoint, error)

	Close(ctx context.Context) error
}
