package viper

import (
	"context"
	"sync"

	"github.com/CXeon/tiles/config"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 是 Viper 实现的配置参数
type Config struct {
	// ConfigPaths 是配置文件搜索路径，默认为 ["."]
	ConfigPaths []string
	// ConfigName 是配置文件名（不含扩展名），如 "config"
	ConfigName string
	// ConfigType 是文件类型："yaml"、"json"、"env" 等，默认 "yaml"
	ConfigType string
	// EnvPrefix 是环境变量前缀，如 "APP" -> APP_DATABASE_HOST
	EnvPrefix string
	// AutoEnv 是否自动绑定所有环境变量，默认 true
	AutoEnv bool
}

type configImpl struct {
	cfg      Config
	v        *viper.Viper
	mu       sync.RWMutex
	handlers []config.ChangeHandler
	snapshot map[string]any
	loaded   bool
}

// New 创建一个基于 Viper 的 Config 实现
func New(cfg Config) config.Config {
	if len(cfg.ConfigPaths) == 0 {
		cfg.ConfigPaths = []string{"."}
	}
	if cfg.ConfigType == "" {
		cfg.ConfigType = "yaml"
	}
	if !cfg.AutoEnv {
		// 零值为 false，但默认应为 true，故只在调用方显式设置 false 时保留
	}
	return &configImpl{
		cfg: cfg,
		v:   viper.New(),
	}
}

func (c *configImpl) Load() error {
	for _, p := range c.cfg.ConfigPaths {
		c.v.AddConfigPath(p)
	}
	if c.cfg.ConfigName != "" {
		c.v.SetConfigName(c.cfg.ConfigName)
	}
	c.v.SetConfigType(c.cfg.ConfigType)

	if c.cfg.EnvPrefix != "" {
		c.v.SetEnvPrefix(c.cfg.EnvPrefix)
	}
	if c.cfg.AutoEnv {
		c.v.AutomaticEnv()
	}

	if err := c.v.ReadInConfig(); err != nil {
		return err
	}

	c.mu.Lock()
	c.snapshot = c.buildSnapshot()
	c.loaded = true
	c.mu.Unlock()

	return nil
}

func (c *configImpl) Get(key string) any {
	return c.v.Get(key)
}

func (c *configImpl) GetString(key string) string {
	return c.v.GetString(key)
}

func (c *configImpl) GetInt(key string) int {
	return c.v.GetInt(key)
}

func (c *configImpl) GetBool(key string) bool {
	return c.v.GetBool(key)
}

func (c *configImpl) GetFloat64(key string) float64 {
	return c.v.GetFloat64(key)
}

func (c *configImpl) Unmarshal(v any) error {
	return c.v.Unmarshal(v)
}

func (c *configImpl) UnmarshalKey(key string, v any) error {
	return c.v.UnmarshalKey(key, v)
}

func (c *configImpl) Watch(handler config.ChangeHandler) error {
	c.mu.Lock()
	if !c.loaded {
		c.mu.Unlock()
		return config.ErrNotLoaded
	}
	c.handlers = append(c.handlers, handler)
	c.mu.Unlock()

	c.v.WatchConfig()
	c.v.OnConfigChange(func(_ fsnotify.Event) {
		c.mu.Lock()
		oldSnapshot := c.snapshot
		newSnapshot := c.buildSnapshot()
		c.snapshot = newSnapshot
		handlers := make([]config.ChangeHandler, len(c.handlers))
		copy(handlers, c.handlers)
		c.mu.Unlock()

		event := buildChangeEvent(oldSnapshot, newSnapshot)
		for _, h := range handlers {
			h(event)
		}
	})

	return nil
}

func (c *configImpl) Close(_ context.Context) error {
	c.mu.Lock()
	c.handlers = nil
	c.mu.Unlock()
	return nil
}

// buildSnapshot 将当前所有 key-value 保存为快照
func (c *configImpl) buildSnapshot() map[string]any {
	keys := c.v.AllKeys()
	snap := make(map[string]any, len(keys))
	for _, k := range keys {
		snap[k] = c.v.Get(k)
	}
	return snap
}

// buildChangeEvent 比较新旧快照，生成变更事件
func buildChangeEvent(old, new map[string]any) *config.ChangeEvent {
	var changes []config.Change

	for k, newVal := range new {
		oldVal, exists := old[k]
		if !exists {
			changes = append(changes, config.Change{
				Key:        k,
				OldValue:   nil,
				NewValue:   newVal,
				ChangeType: config.ChangeTypeAdded,
			})
		} else if oldVal != newVal {
			changes = append(changes, config.Change{
				Key:        k,
				OldValue:   oldVal,
				NewValue:   newVal,
				ChangeType: config.ChangeTypeModified,
			})
		}
	}

	for k, oldVal := range old {
		if _, exists := new[k]; !exists {
			changes = append(changes, config.Change{
				Key:        k,
				OldValue:   oldVal,
				NewValue:   nil,
				ChangeType: config.ChangeTypeDeleted,
			})
		}
	}

	return &config.ChangeEvent{Changes: changes}
}
