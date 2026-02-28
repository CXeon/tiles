package apollo

import (
	"context"
	"fmt"
	"sync"

	"github.com/CXeon/tiles/config"
	"github.com/apolloconfig/agollo/v4"
	agolloConfig "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/mitchellh/mapstructure"
)

// Config 是 Apollo 实现的配置参数
type Config struct {
	// AppID 是 Apollo App ID（必填）
	AppID string
	// Cluster 是集群名称，如 "default"、"dev"（必填）
	Cluster string
	// IP 是 Apollo 服务器地址（必填）
	IP string
	// NamespaceName 是命名空间，默认 "application"
	NamespaceName string
	// Secret 是认证密钥（可选）
	Secret string
	// IsBackupConfig 是否启用本地备份，默认 true
	IsBackupConfig bool
}

type configImpl struct {
	cfg       Config
	client    agollo.Client
	mu        sync.RWMutex
	handlers  []config.ChangeHandler
	loaded    bool
	namespace string
}

// New 创建一个基于 Apollo 的 Config 实现
func New(cfg Config) config.Config {
	ns := cfg.NamespaceName
	if ns == "" {
		ns = "application"
	}
	return &configImpl{
		cfg:       cfg,
		namespace: ns,
	}
}

func (c *configImpl) Load() error {
	appConfig := &agolloConfig.AppConfig{
		AppID:          c.cfg.AppID,
		Cluster:        c.cfg.Cluster,
		IP:             c.cfg.IP,
		NamespaceName:  c.namespace,
		Secret:         c.cfg.Secret,
		IsBackupConfig: c.cfg.IsBackupConfig,
	}

	client, err := agollo.StartWithConfig(func() (*agolloConfig.AppConfig, error) {
		return appConfig, nil
	})
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.client = client
	c.loaded = true
	c.mu.Unlock()

	return nil
}

func (c *configImpl) Get(key string) any {
	val, err := c.client.GetConfigCache(c.namespace).Get(key)
	if err != nil {
		return nil
	}
	return val
}

func (c *configImpl) GetString(key string) string {
	val := c.Get(key)
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

func (c *configImpl) GetInt(key string) int {
	val := c.Get(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func (c *configImpl) GetBool(key string) bool {
	val := c.Get(key)
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	if s, ok := val.(string); ok {
		return s == "true" || s == "1" || s == "yes"
	}
	return false
}

func (c *configImpl) GetFloat64(key string) float64 {
	val := c.Get(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func (c *configImpl) Unmarshal(v any) error {
	cache := c.client.GetConfigCache(c.namespace)
	data := make(map[string]any)
	cache.Range(func(key, value any) bool {
		if k, ok := key.(string); ok {
			data[k] = value
		}
		return true
	})
	return mapstructure.Decode(data, v)
}

func (c *configImpl) UnmarshalKey(key string, v any) error {
	val := c.Get(key)
	if val == nil {
		return config.ErrKeyNotFound
	}
	return mapstructure.Decode(val, v)
}

func (c *configImpl) Watch(handler config.ChangeHandler) error {
	c.mu.Lock()
	if !c.loaded {
		c.mu.Unlock()
		return config.ErrNotLoaded
	}
	c.handlers = append(c.handlers, handler)
	c.mu.Unlock()

	c.client.AddChangeListener(&changeListener{impl: c})
	return nil
}

func (c *configImpl) Close(_ context.Context) error {
	c.mu.Lock()
	c.handlers = nil
	c.mu.Unlock()
	return nil
}

func (c *configImpl) broadcast(event *config.ChangeEvent) {
	c.mu.RLock()
	handlers := make([]config.ChangeHandler, len(c.handlers))
	copy(handlers, c.handlers)
	c.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

// changeListener 实现 agollo 的 ChangeListener 接口
type changeListener struct {
	impl *configImpl
}

func (l *changeListener) OnChange(event *storage.ChangeEvent) {
	var changes []config.Change
	for key, change := range event.Changes {
		var ct config.ChangeType
		switch change.ChangeType {
		case storage.ADDED:
			ct = config.ChangeTypeAdded
		case storage.MODIFIED:
			ct = config.ChangeTypeModified
		case storage.DELETED:
			ct = config.ChangeTypeDeleted
		}
		changes = append(changes, config.Change{
			Key:        key,
			OldValue:   change.OldValue,
			NewValue:   change.NewValue,
			ChangeType: ct,
		})
	}
	l.impl.broadcast(&config.ChangeEvent{Changes: changes})
}

func (l *changeListener) OnNewestChange(_ *storage.FullChangeEvent) {}
