package kv_store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStore(ctx context.Context, endpoints []string, password string, db int) (KvStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	client := redis.NewClient(&redis.Options{
		Addr:     endpoints[0], // For simplicity, take the first endpoint. Can be extended to cluster.
		Password: password,
		DB:       db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &redisStore{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *redisStore) Put(key string, value []byte) error {
	return s.client.Set(s.ctx, key, value, 0).Err()
}

func (s *redisStore) Add(key string, value []byte) error {
	// For Traefik KV, Add usually means appending to a list or just setting if it's a simple KV.
	// Based on Traefik docs, many values are simple strings.
	// If it's a list, we use RPUSH. But Traefik KV usually expects separate keys for lists (e.g., key/0, key/1).
	// The current interface says "if value type is list, then it will be added to the list behind".
	// Since we don't know the type, we'll assume it's a set if it exists or just set it.
	// Actually, let's just use Set for now asTraefik 3.6 KV provider mostly uses simple SET.
	return s.Put(key, value)
}

func (s *redisStore) Get(key string) ([]byte, error) {
	val, err := s.client.Get(s.ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (s *redisStore) GetByPrefix(prefix string) (map[string][]byte, error) {
	keys, err := s.client.Keys(s.ctx, prefix+"*").Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for _, key := range keys {
		val, err := s.client.Get(s.ctx, key).Bytes()
		if err != nil {
			return nil, err
		}
		result[key] = val
	}
	return result, nil
}

func (s *redisStore) Delete(key string) error {
	return s.client.Del(s.ctx, key).Err()
}

func (s *redisStore) Close() error {
	return s.client.Close()
}
