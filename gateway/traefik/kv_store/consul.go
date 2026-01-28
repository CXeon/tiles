package kv_store

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
)

type consulStore struct {
	client    *api.Client
	sessionID string // 全局 Session ID，所有带 TTL 的 key 共享
	ctx       context.Context
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
		ctx:    ctx,
	}, nil
}

func (s *consulStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
	p := &api.KVPair{Key: key, Value: value}

	// 如果有 TTL，需要创建或使用 Session
	if len(expired) > 0 && expired[0] > 0 {
		// 确保 Session 存在
		if err := s.ensureSession(expired[0]); err != nil {
			return fmt.Errorf("failed to ensure session: %w", err)
		}
		p.Session = s.sessionID
	}

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
	// Destroy session if exists
	if s.sessionID != "" {
		s.client.Session().Destroy(s.sessionID, nil)
	}
	// Consul client doesn't have a Close method in its API.
	return nil
}

// ensureSession 确保 Session 存在，如果不存在则创建
func (s *consulStore) ensureSession(ttl uint32) error {
	if s.sessionID != "" {
		// Session 已存在，无需重新创建
		return nil
	}

	// 创建 Session
	sessionEntry := &api.SessionEntry{
		Behavior: "delete", // Session 过期时自动删除关联的 key
		TTL:      fmt.Sprintf("%ds", ttl),
	}

	sessionID, _, err := s.client.Session().Create(sessionEntry, nil)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	s.sessionID = sessionID
	return nil
}

func (s *consulStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
	// Consul 使用 Session 续期，不需要 key 参数
	if s.sessionID == "" {
		return fmt.Errorf("no session to renew")
	}

	// Renew Session
	_, _, err := s.client.Session().Renew(s.sessionID, nil)
	if err != nil {
		return fmt.Errorf("failed to renew session: %w", err)
	}
	return nil
}

func (s *consulStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
	// Consul 全局共享 Session，所有 key 关联同一个 Session
	// 只需要 Renew 一次 Session，所有 key 同时续期
	return s.KeepAlive(ctx, "", ttl...)
}
