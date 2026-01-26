package kv_store

// KvStore kv类型provider存储接口
type KvStore interface {
	// Put 保存数据
	Put(key string, value []byte) error

	// Add 向某个key添加数据,如果值类型是list,则会添加到list后面
	Add(key string, value []byte) error

	// Get 获取数据
	Get(key string) ([]byte, error)

	// GetByPrefix 通过前缀获取数据
	//  返回map的key是匹配到的key
	GetByPrefix(prefix string) (map[string][]byte, error)

	// Delete 删除数据
	Delete(key string) error

	// Close 关闭连接
	Close() error
}
