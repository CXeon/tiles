package kv_store

import (
	"context"
	"errors"
)

// 默认 TTL 配置
const (
	DefaultTTLSeconds = 15 // 默认 TTL 时间（秒）
)

// Standard errors for KV operations
var (
	// ErrKeyNotFound is returned when a key does not exist
	ErrKeyNotFound = errors.New("key not found")
	// ErrConnectionFailed is returned when connection to backend fails
	ErrConnectionFailed = errors.New("connection failed")
)

// KvStore kv类型provider存储接口
type KvStore interface {
	// Put 保存数据
	// expired: 可选参数，过期时间（单位：秒）。0 或不传表示永不过期。
	Put(ctx context.Context, key string, value []byte, expired ...uint32) error

	// Get 获取数据
	Get(ctx context.Context, key string) ([]byte, error)

	// GetByPrefix 通过前缀获取数据
	//  返回map的key是匹配到的key
	GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error)

	// Delete 删除数据
	Delete(ctx context.Context, key string) error

	// DeleteByPrefix 通过前缀批量删除数据
	DeleteByPrefix(ctx context.Context, prefix string) error

	// KeepAlive 续期单个 key 的生命周期
	// ttl: 可选参数，续期的 TTL 值（单位：秒）。如果不传，使用默认 TTL。
	KeepAlive(ctx context.Context, key string, ttl ...uint32) error

	// BatchKeepAlive 批量续期多个 key，确保它们的 TTL 同步
	// 重要：解决 Redis 单个 key 续期导致的 TTL 时间漂移问题
	// 对于 Redis：使用 Pipeline 批量 EXPIRE，确保所有 key 的过期时间一致
	// 对于 Consul/Etcd/ZooKeeper：内部调用一次 Session/Lease 续约即可
	BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error

	// Close 关闭连接
	Close() error
}
