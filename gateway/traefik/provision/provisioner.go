package provision

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/CXeon/tiles/gateway/traefik"
	"github.com/CXeon/tiles/gateway/traefik/kv_store"
)

// ForwardAuthConfig holds the configuration for a ForwardAuth middleware instance.
type ForwardAuthConfig struct {
	Company             string
	Project             string
	Env                 string
	Cluster             string
	Address             string
	TrustForwardHeader  bool
	AuthResponseHeaders []string
}

// Provisioner writes ForwardAuth middleware definitions to the KV Store.
// It is intended for use by privileged services (e.g. traefik-support) that hold
// credentials allowing writes to traefik/http/middlewares/.
type Provisioner struct {
	store kv_store.KvStore
}

// NewProvisioner creates a new Provisioner connected to the given provider's KV Store.
func NewProvisioner(ctx context.Context, provider *traefik.Provider) (*Provisioner, error) {
	store, err := newKvStore(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to KV store: %w", err)
	}
	return &Provisioner{store: store}, nil
}

// SetForwardAuth idempotently writes a ForwardAuth middleware definition to the KV Store.
// The middleware name follows the format {Env}.{Cluster}.{Company}.{Project}.ForwardAuth.
func (p *Provisioner) SetForwardAuth(ctx context.Context, cfg ForwardAuthConfig) error {
	con := traefik.NewConstructor()
	name := con.MiddlewareName(cfg.Env, cfg.Cluster, cfg.Company, cfg.Project)
	prefix := con.MiddlewareKeyPrefix(name)

	if err := p.store.Put(ctx, prefix+"forwardAuth/address", []byte(cfg.Address)); err != nil {
		return fmt.Errorf("failed to write address: %w", err)
	}
	if err := p.store.Put(ctx, prefix+"forwardAuth/trustForwardHeader", []byte(strconv.FormatBool(cfg.TrustForwardHeader))); err != nil {
		return fmt.Errorf("failed to write trustForwardHeader: %w", err)
	}
	for i, h := range cfg.AuthResponseHeaders {
		key := fmt.Sprintf("%sforwardAuth/authResponseHeaders/%d", prefix, i)
		if err := p.store.Put(ctx, key, []byte(h)); err != nil {
			return fmt.Errorf("failed to write authResponseHeaders[%d]: %w", i, err)
		}
	}
	return nil
}

// RemoveForwardAuth deletes the ForwardAuth middleware definition for the given config.
// It is idempotent: deleting a non-existent middleware returns nil.
func (p *Provisioner) RemoveForwardAuth(ctx context.Context, cfg ForwardAuthConfig) error {
	con := traefik.NewConstructor()
	name := con.MiddlewareName(cfg.Env, cfg.Cluster, cfg.Company, cfg.Project)
	prefix := con.MiddlewareKeyPrefix(name)

	if err := p.store.DeleteByPrefix(ctx, prefix); err != nil {
		if errors.Is(err, kv_store.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("failed to remove ForwardAuth middleware: %w", err)
	}
	return nil
}

// Close releases the KV Store connection.
func (p *Provisioner) Close() error {
	if p.store != nil {
		return p.store.Close()
	}
	return nil
}

// newKvStore creates a KvStore from the given Provider configuration.
func newKvStore(provider *traefik.Provider) (kv_store.KvStore, error) {
	switch provider.KVType {
	case traefik.ProviderTypeRedis:
		return kv_store.NewRedisStore(kv_store.RedisConfig{
			Endpoints:      provider.Endpoints,
			Password:       provider.Password,
			DB:             provider.DBIndex,
			PoolSize:       provider.PoolSize,
			MinIdleConns:   provider.MinIdleConns,
			ConnectTimeout: provider.ConnectTimeout,
			ReadTimeout:    provider.ReadTimeout,
			WriteTimeout:   provider.WriteTimeout,
		})
	case traefik.ProviderTypeConsul:
		return kv_store.NewConsulStore(kv_store.ConsulConfig{
			Endpoints:      provider.Endpoints,
			Username:       provider.Username,
			Password:       provider.Password,
			ConnectTimeout: provider.ConnectTimeout,
			ReadTimeout:    provider.ReadTimeout,
		})
	case traefik.ProviderTypeEtcd:
		return kv_store.NewEtcdStore(kv_store.EtcdConfig{
			Endpoints:      provider.Endpoints,
			Username:       provider.Username,
			Password:       provider.Password,
			ConnectTimeout: provider.ConnectTimeout,
			ReadTimeout:    provider.ReadTimeout,
		})
	case traefik.ProviderTypeZooKeeper:
		return kv_store.NewZookeeperStore(kv_store.ZookeeperConfig{
			Endpoints:      provider.Endpoints,
			ConnectTimeout: provider.ConnectTimeout,
			SessionTimeout: provider.ReadTimeout,
		})
	default:
		return nil, fmt.Errorf("unsupported traefik provider type: %v", provider.KVType)
	}
}
