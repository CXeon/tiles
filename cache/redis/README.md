# Cache - Redis Implementation

基于 [go-redis/v9](https://github.com/redis/go-redis) 的 Redis 缓存实现，覆盖 Redis 常用数据类型与高级特性。

## 特性

- **完整数据类型支持**：String、Hash、List、Set、Sorted Set
- **Pipeline**：批量命令单次 RTT 发送，显著减少网络开销
- **事务 Pipeline**：`TxPipeline` 以 `MULTI/EXEC` 包裹，保证原子执行
- **Pub/Sub**：频道消息发布与订阅，支持跨进程通信
- **游标扫描**：`Scan` 分页遍历 key，避免 `KEYS` 阻塞生产环境
- **错误归一**：`redis.Nil` 统一转换为 `cache.ErrKeyNotFound`，`WRONGTYPE` 转换为 `cache.ErrWrongType`
- **原始客户端暴露**：通过 `Client()` 获取底层 `*goredis.Client`，支持 Lua 脚本、Watch/CAS 等高级场景

## 快速开始

```go
import (
    "context"
    "time"

    "github.com/CXeon/tiles/cache"
    redisCache "github.com/CXeon/tiles/cache/redis"
)

ctx := context.Background()

// 创建 Redis 缓存客户端
c := redisCache.New(redisCache.Config{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
defer c.Close()

// String
c.Set(ctx, "name", "alice", time.Hour)
name, _ := c.Get(ctx, "name")

// Hash
c.HSet(ctx, "user:1", "name", "alice")
c.HSet(ctx, "user:1", "age", "30")
fields, _ := c.HGetAll(ctx, "user:1")

// Sorted Set（排行榜）
c.ZAdd(ctx, "leaderboard",
    cache.Z{Score: 100, Member: "alice"},
    cache.Z{Score: 200, Member: "bob"},
)
top, _ := c.ZRevRange(ctx, "leaderboard", 0, 9)
```

## 配置说明

```go
type Config struct {
    Addr         string        // Redis 地址，如 "localhost:6379"
    Password     string        // 认证密码，无密码留空
    DB           int           // 数据库编号，默认 0
    PoolSize     int           // 连接池最大连接数
    MinIdleConns int           // 最小空闲连接数
    DialTimeout  time.Duration // 建连超时
    ReadTimeout  time.Duration // 读超时
    WriteTimeout time.Duration // 写超时
}
```

## Pipeline

Pipeline 将多条命令打包成一次网络请求发送，适合批量读写场景：

```go
cmds, err := c.Pipeline(ctx, func(pipe redisCache.Pipeliner) error {
    pipe.Set(ctx, "k1", "v1", time.Hour)
    pipe.Set(ctx, "k2", "v2", time.Hour)
    pipe.Incr(ctx, "counter")
    return nil
})
// 所有命令在函数返回后一次性发送并执行
// cmds[i].Err() 可查看每条命令的执行结果
```

### 事务 Pipeline（MULTI/EXEC）

```go
_, err := c.TxPipeline(ctx, func(pipe redisCache.Pipeliner) error {
    pipe.Decr(ctx, "stock")
    pipe.Set(ctx, "order:"+id, data, 24*time.Hour)
    return nil
})
// 两条命令原子执行，中间不会被其他客户端插入
```

### Pipeline 批量读取

```go
var getCmd1, getCmd2 *goredis.StringCmd

c.Pipeline(ctx, func(pipe redisCache.Pipeliner) error {
    getCmd1 = pipe.Get(ctx, "k1")
    getCmd2 = pipe.Get(ctx, "k2")
    return nil
})

v1, _ := getCmd1.Result()
v2, _ := getCmd2.Result()
```

## Pub/Sub

```go
// 订阅频道
pubsub := c.Subscribe(ctx, "notifications")
defer pubsub.Close()

// 接收消息（阻塞）
go func() {
    ch := pubsub.Channel()
    for msg := range ch {
        fmt.Printf("channel: %s, payload: %s\n", msg.Channel, msg.Payload)
    }
}()

// 发布消息
c.Publish(ctx, "notifications", "hello")
```

## Scan 分页遍历

生产环境禁止直接使用 `KEYS *`，应使用 `Scan` 分页扫描：

```go
var cursor uint64
var allKeys []string

for {
    keys, next, err := c.Scan(ctx, cursor, "user:*", 100)
    if err != nil {
        break
    }
    allKeys = append(allKeys, keys...)
    cursor = next
    if cursor == 0 {
        break // 遍历完成
    }
}
```

## 高级用法（原始客户端）

通过 `Client()` 获取底层 `*goredis.Client`，使用 Lua 脚本、Watch/CAS 等未封装的功能：

```go
rdb := c.Client()

// Lua 脚本（原子操作）
script := goredis.NewScript(`
    local val = redis.call("GET", KEYS[1])
    if val == ARGV[1] then
        return redis.call("DEL", KEYS[1])
    end
    return 0
`)
result, err := script.Run(ctx, rdb, []string{"lock:key"}, "token123").Result()

// Watch/CAS（乐观锁）
err = rdb.Watch(ctx, func(tx *goredis.Tx) error {
    n, err := tx.Get(ctx, "counter").Int()
    if err != nil {
        return err
    }
    _, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
        pipe.Set(ctx, "counter", n+1, 0)
        return nil
    })
    return err
}, "counter")
```

## 与内存缓存互换

`RedisCache` 实现了 `cache.Cache` 接口，可与内存缓存无缝切换：

```go
var c cache.Cache

if os.Getenv("REDIS_ADDR") != "" {
    c = redisCache.New(redisCache.Config{Addr: os.Getenv("REDIS_ADDR")})
} else {
    c = memCache.New(memCache.Config{})
}

// 所有业务代码使用 cache.Cache，无感知切换
c.Set(ctx, "key", "value", time.Hour)

// 需要 Redis 独有功能时，类型断言获取扩展接口
if rc, ok := c.(redisCache.RedisCache); ok {
    rc.Pipeline(ctx, func(pipe redisCache.Pipeliner) error {
        // ...
        return nil
    })
}
```

## 错误处理

| Redis 错误 | 转换后 |
|-----------|--------|
| `redis.Nil`（key 不存在） | `cache.ErrKeyNotFound` |
| `WRONGTYPE ...` | `cache.ErrWrongType` |
| `ERR value is not an integer` | `cache.ErrNotInteger` |
| 其他错误 | 原始 Redis 错误 |

```go
val, err := c.Get(ctx, "missing")
if errors.Is(err, cache.ErrKeyNotFound) {
    // key 不存在
}
```

## 运行测试

Redis 集成测试需要运行中的 Redis 实例，通过环境变量指定地址：

```bash
# 无 Redis 时自动 skip
go test github.com/CXeon/tiles/cache/redis/... -v

# 有 Redis 时运行完整集成测试
REDIS_TEST_ADDR=localhost:6379 go test github.com/CXeon/tiles/cache/redis/... -v
```

## 相关链接

- [Cache 接口定义](../cache.go)
- [Memory 实现](../memory/README.md)
- [go-redis 官方文档](https://github.com/redis/go-redis)
- [Redis 命令参考](https://redis.io/commands)
