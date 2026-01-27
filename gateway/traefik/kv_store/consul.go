package kv_store

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
)

type consulStore struct {
	client *api.Client
}

type ConsulConfig struct {
	Endpoints      []string
	Username       string
	Password       string
	ConnectTimeout time.Duration // 连接超时时间，单位：time.Duration（如 5*time.Second）
	ReadTimeout    time.Duration // 读取超时时间，单位：time.Duration（如 10*time.Second）
}

func NewConsulStore(ctx context.Context, cfg ConsulConfig) (KvStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Set default timeouts
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 10 * time.Second
	}

	config := api.DefaultConfig()
	config.Address = cfg.Endpoints[0]

	// Only support Basic Auth (username + password)
	if cfg.Username != "" || cfg.Password != "" {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	// Configure timeouts
	config.HttpClient.Timeout = cfg.ReadTimeout

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	return &consulStore{
		client: client,
	}, nil
}

func (s *consulStore) Put(ctx context.Context, key string, value []byte) error {
	p := &api.KVPair{Key: key, Value: value}
	_, err := s.client.KV().Put(p, nil)
	if err != nil {
		return fmt.Errorf("consul put error: %w", err)
	}
	return nil
}

func (s *consulStore) Get(ctx context.Context, key string) ([]byte, error) {
	pair, _, err := s.client.KV().Get(key, nil)
	if err != nil {
		return nil, fmt.Errorf("consul get error: %w", err)
	}
	if pair == nil {
		return nil, ErrKeyNotFound
	}
	return pair.Value, nil
}

func (s *consulStore) GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	pairs, _, err := s.client.KV().List(prefix, nil)
	if err != nil {
		return nil, fmt.Errorf("consul list error: %w", err)
	}

	result := make(map[string][]byte)
	for _, pair := range pairs {
		result[pair.Key] = pair.Value
	}
	return result, nil
}

func (s *consulStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.KV().Delete(key, nil)
	if err != nil {
		return fmt.Errorf("consul delete error: %w", err)
	}
	return nil
}

func (s *consulStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	_, err := s.client.KV().DeleteTree(prefix, nil)
	if err != nil {
		return fmt.Errorf("consul delete tree error: %w", err)
	}
	return nil
}

func (s *consulStore) Close() error {
	// Consul client doesn't have a Close method in its API.
	return nil
}
