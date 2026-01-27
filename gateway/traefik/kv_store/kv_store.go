package kv_store

import (
	"context"
	"errors"
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
	Put(ctx context.Context, key string, value []byte) error

	// Get 获取数据
	Get(ctx context.Context, key string) ([]byte, error)

	// GetByPrefix 通过前缀获取数据
	//  返回map的key是匹配到的key
	GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error)

	// Delete 删除数据
	Delete(ctx context.Context, key string) error

	// DeleteByPrefix 通过前缀批量删除数据
	DeleteByPrefix(ctx context.Context, prefix string) error

	// Close 关闭连接
	Close() error
}
