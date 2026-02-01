package kv_store

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdStore struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID // 全局 Lease ID，所有带 TTL 的 key 共享
}

type EtcdConfig struct {
	Endpoints      []string
	Username       string
	Password       string
	ConnectTimeout time.Duration // 连接超时时间，单位：time.Duration（如 5*time.Second）
	ReadTimeout    time.Duration // 读取超时时间，单位：time.Duration（如 10*time.Second）
}

func NewEtcdStore(cfg EtcdConfig) (KvStore, error) {
	ctx := context.Background()

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

func (s *etcdStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
	// 如果有 TTL，需要创建或使用 Lease
	if len(expired) > 0 && expired[0] > 0 {
		// 确保 Lease 存在
		if err := s.ensureLease(ctx, int64(expired[0])); err != nil {
			return fmt.Errorf("failed to ensure lease: %w", err)
		}
		_, err := s.client.Put(ctx, key, string(value), clientv3.WithLease(s.leaseID))
		if err != nil {
			return fmt.Errorf("etcd put error: %w", err)
		}
	} else {
		_, err := s.client.Put(ctx, key, string(value))
		if err != nil {
			return fmt.Errorf("etcd put error: %w", err)
		}
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
	// Revoke lease if exists
	if s.leaseID != 0 {
		s.client.Revoke(context.Background(), s.leaseID)
	}
	return s.client.Close()
}

// ensureLease 确保 Lease 存在，如果不存在则创建
func (s *etcdStore) ensureLease(ctx context.Context, ttl int64) error {
	if s.leaseID != 0 {
		// Lease 已存在，无需重新创建
		return nil
	}

	// 创建 Lease 并启动自动续期
	leaseResp, err := s.client.Grant(ctx, ttl)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	s.leaseID = leaseResp.ID

	// 启动自动 KeepAlive
	ch, err := s.client.KeepAlive(ctx, s.leaseID)
	if err != nil {
		return fmt.Errorf("failed to start keep alive: %w", err)
	}

	// 后台消费 KeepAlive 响应，防止管道堵塞
	go func() {
		for range ch {
			// 消费响应，防止堆积
		}
	}()

	return nil
}

func (s *etcdStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
	// Etcd 使用 Lease 续期，不需要 key 参数
	if s.leaseID == 0 {
		return fmt.Errorf("no lease to keep alive")
	}

	// 手动续期一次
	_, err := s.client.KeepAliveOnce(ctx, s.leaseID)
	if err != nil {
		return fmt.Errorf("failed to keep alive: %w", err)
	}
	return nil
}

func (s *etcdStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
	// Etcd 所有 key 关联同一个 Lease
	// 只需要刷新一次 Lease，所有 key 同时续期
	return s.KeepAlive(ctx, "", ttl...)
}
