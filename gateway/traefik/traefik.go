package traefik

import (
	"context"
	"fmt"

	"github.com/CXeon/tiles/gateway"
)

type traefikClient struct {
	handler *handler
}

func NewTraefikClient(ctx context.Context, provider *Provider) (gateway.Client, error) {
	h, err := NewHandler(ctx, provider)
	if err != nil {
		return nil, err
	}
	return &traefikClient{
		handler: h,
	}, nil
}

func (c *traefikClient) Register(ctx context.Context, endpoint *gateway.Endpoint) error {
	if endpoint == nil {
		return fmt.Errorf("endpoint is nil")
	}

	if err := endpoint.Protocol.Validate(); err != nil {
		return err
	}

	// 协议转换: https -> http
	normalizedEndpoint := *endpoint
	if normalizedEndpoint.Protocol == gateway.ProtocolTypeHttps {
		normalizedEndpoint.Protocol = gateway.ProtocolTypeHttp
	}

	return c.handler.Register(normalizedEndpoint)
}

func (c *traefikClient) Deregister(ctx context.Context, endpoint *gateway.Endpoint) error {
	if endpoint == nil {
		return fmt.Errorf("endpoint is nil")
	}

	normalizedEndpoint := *endpoint
	if normalizedEndpoint.Protocol == gateway.ProtocolTypeHttps {
		normalizedEndpoint.Protocol = gateway.ProtocolTypeHttp
	}

	return c.handler.Deregister(normalizedEndpoint)
}

func (c *traefikClient) Update(ctx context.Context, endpoint *gateway.Endpoint) error {
	if endpoint == nil {
		return fmt.Errorf("endpoint is nil")
	}

	if err := endpoint.Protocol.Validate(); err != nil {
		return err
	}

	normalizedEndpoint := *endpoint
	if normalizedEndpoint.Protocol == gateway.ProtocolTypeHttps {
		normalizedEndpoint.Protocol = gateway.ProtocolTypeHttp
	}

	return c.handler.Update(normalizedEndpoint)
}

func (c *traefikClient) Close(ctx context.Context) error {
	return c.handler.Close()
}
