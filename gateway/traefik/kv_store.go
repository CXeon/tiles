package traefik

// KvStore kv类型provider存储接口
type KvStore interface {
	// Put 保存数据
	Put(key string, value []byte) error
	// Get 获取数据
	Get(key string) ([]byte, error)
	// GetByPrefix 通过前缀获取数据
	//  返回map的key是匹配到的key
	GetByPrefix(prefix string) (map[string][]byte, error)
}
