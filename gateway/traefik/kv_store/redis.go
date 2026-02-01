package kv_store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	client *redis.Client
}

type RedisConfig struct {
	Endpoints      []string
	Password       string
	DB             int
	PoolSize       int
	MinIdleConns   int
	ConnectTimeout time.Duration // 连接超时时间，单位：time.Duration（如 5*time.Second）
	ReadTimeout    time.Duration // 读取超时时间，单位：time.Duration（如 3*time.Second）
	WriteTimeout   time.Duration // 写入超时时间，单位：time.Duration（如 3*time.Second）
}

func NewRedisStore(config RedisConfig) (KvStore, error) {
	ctx := context.Background()

	// Set default timeouts if not provided
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}

	// Set default pool settings
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 2
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Endpoints[0], // For simplicity, take the first endpoint. Can be extended to cluster.
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.ConnectTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	return &redisStore{
		client: client,
	}, nil
}

func (s *redisStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
	var expiration time.Duration
	if len(expired) > 0 && expired[0] > 0 {
		expiration = time.Duration(expired[0]) * time.Second
	}
	return s.client.Set(ctx, key, value, expiration).Err()
}

func (s *redisStore) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}
	return val, nil
}

func (s *redisStore) GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error) {
	var keys []string
	var cursor uint64
	pattern := prefix + "*"

	// Use SCAN instead of KEYS for better performance
	for {
		var batch []string
		var err error
		batch, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan error: %w", err)
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Use MGET for batch retrieval
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget error: %w", err)
	}

	result := make(map[string][]byte, len(keys))
	for i, key := range keys {
		if values[i] != nil {
			if str, ok := values[i].(string); ok {
				result[key] = []byte(str)
			}
		}
	}
	return result, nil
}

func (s *redisStore) Delete(ctx context.Context, key string) error {
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}
	return nil
}

func (s *redisStore) DeleteByPrefix(ctx context.Context, prefix string) error {
	var keys []string
	var cursor uint64
	pattern := prefix + "*"

	// Use SCAN to find all matching keys
	for {
		var batch []string
		var err error
		batch, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan error: %w", err)
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete in batches to avoid blocking
	const batchSize = 100
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]
		if err := s.client.Del(ctx, batch...).Err(); err != nil {
			return fmt.Errorf("redis delete batch error: %w", err)
		}
	}

	return nil
}

func (s *redisStore) Close() error {
	return s.client.Close()
}

func (s *redisStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
	var duration time.Duration
	if len(ttl) > 0 && ttl[0] > 0 {
		duration = time.Duration(ttl[0]) * time.Second
	} else {
		// 默认 TTL 15 秒
		duration = DefaultTTLSeconds * time.Second
	}
	return s.client.Expire(ctx, key, duration).Err()
}

func (s *redisStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
	if len(keys) == 0 {
		return nil
	}

	var duration time.Duration
	if len(ttl) > 0 && ttl[0] > 0 {
		duration = time.Duration(ttl[0]) * time.Second
	} else {
		// 默认 TTL 15 秒
		duration = DefaultTTLSeconds * time.Second
	}

	// 使用 Pipeline 批量操作，确保原子性和时间一致性
	// 所有 EXPIRE 命令在极短时间内执行，几乎同时设置过期时间
	pipe := s.client.Pipeline()
	for _, key := range keys {
		pipe.Expire(ctx, key, duration)
	}
	_, err := pipe.Exec(ctx)
	return err
}
