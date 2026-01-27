package kv_store

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdStore struct {
	client *clientv3.Client
}

type EtcdConfig struct {
	Endpoints      []string
	Username       string
	Password       string
	ConnectTimeout time.Duration // 连接超时时间，单位：time.Duration（如 5*time.Second）
	ReadTimeout    time.Duration // 读取超时时间，单位：time.Duration（如 10*time.Second）
}

func NewEtcdStore(ctx context.Context, cfg EtcdConfig) (KvStore, error) {
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

	config := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.ConnectTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	}

	client, err := clientv3.New(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Test connection
	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	_, err = client.Get(timeoutCtx, "/", clientv3.WithLimit(1))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	return &etcdStore{
		client: client,
	}, nil
}

func (s *etcdStore) Put(ctx context.Context, key string, value []byte) error {
	_, err := s.client.Put(ctx, key, string(value))
	if err != nil {
		return fmt.Errorf("etcd put error: %w", err)
	}
	return nil
}

func (s *etcdStore) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("etcd get error: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return nil, ErrKeyNotFound
	}
	return resp.Kvs[0].Value, nil
}

func (s *etcdStore) GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	resp, err := s.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd get prefix error: %w", err)
	}

	result := make(map[string][]byte, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		result[string(kv.Key)] = kv.Value
	}
	return result, nil
}

func (s *etcdStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("etcd delete error: %w", err)
	}
	return nil
}

func (s *etcdStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	_, err := s.client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("etcd delete prefix error: %w", err)
	}
	return nil
}

func (s *etcdStore) Close() error {
	return s.client.Close()
}
