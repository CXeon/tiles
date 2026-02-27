package config

import (
	"context"
	"errors"
)

var (
	ErrNotLoaded   = errors.New("config not loaded, call Load() first")
	ErrKeyNotFound = errors.New("config key not found")
)

type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeDeleted  ChangeType = "deleted"
)

// Change 表示单个配置项的变更
type Change struct {
	Key        string
	OldValue   any
	NewValue   any
	ChangeType ChangeType
}

// ChangeEvent 是配置变更事件，Changes 为空表示整体重载
type ChangeEvent struct {
	Changes []Change
}

type ChangeHandler func(event *ChangeEvent)

type Config interface {
	// Load 从配置源加载配置（必须首先调用）
	Load() error

	// Get 系列方法通过 key 读取配置值（支持 "." 分隔的嵌套路径）
	Get(key string) any
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat64(key string) float64

	// Unmarshal 将整个配置反序列化到结构体（需 mapstructure tag）
	Unmarshal(v any) error
	// UnmarshalKey 将指定 key 下的子配置反序列化到结构体
	UnmarshalKey(key string, v any) error

	// Watch 注册配置变更回调，Load 后可调用
	Watch(handler ChangeHandler) error

	// Close 停止监听并释放资源
	Close(ctx context.Context) error
}
