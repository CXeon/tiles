# Cache（缓存模块）

提供统一的缓存抽象接口，覆盖 Redis 常用数据类型操作，支持 Memory 和 Redis 两种实现，可无感知切换。

## 接口定义

`Cache` 接口覆盖 Redis 常用操作的最大公约数，Memory 实现可作为 Redis 的开发/测试替代品。

```go
type Cache interface {
    // String
    Set(ctx, key, value string, ttl time.Duration) error
    Get(ctx, key string) (string, error)
    Delete(ctx, keys ...string) error
    Exists(ctx, key string) (bool, error)
    Expire(ctx, key string, ttl time.Duration) error
    TTL(ctx, key string) (time.Duration, error)
    SetNX(ctx, key, value string, ttl time.Duration) (bool, error)
    GetDel(ctx, key string) (string, error)

    // 原子计数器
    Incr(ctx, key string) (int64, error)
    IncrBy(ctx, key string, value int64) (int64, error)
    Decr(ctx, key string) (int64, error)
    DecrBy(ctx, key string, value int64) (int64, error)

    // Hash
    HSet / HGet / HGetAll / HDel / HExists / HLen

    // List
    LPush / RPush / LPop / RPop / LRange / LLen

    // Set
    SAdd / SRem / SMembers / SIsMember / SCard

    // Sorted Set
    ZAdd / ZRem / ZScore / ZRange / ZRevRange / ZRangeWithScores / ZCard / ZRank

    // Keys
    Keys(ctx, pattern string) ([]string, error)

    Close() error
}
```

## 错误变量

| 变量 | 说明 |
|------|------|
| `ErrKeyNotFound` | key 不存在或已过期 |
| `ErrWrongType` | 操作类型与 key 持有的数据类型不匹配 |
| `ErrNotInteger` | 原子操作的 value 不是整数 |

## 常量

```go
const NoExpiration = time.Duration(0)  // 不设过期时间
```

## Sorted Set 成员

```go
type Z struct {
    Score  float64
    Member string
}
```

## 快速开始

```go
import (
    "github.com/CXeon/tiles/cache"
    memCache "github.com/CXeon/tiles/cache/memory"
)

c := memCache.New(memCache.Config{})
defer c.Close()

// String
c.Set(ctx, "name", "alice", time.Hour)
name, _ := c.Get(ctx, "name")

// Hash
c.HSet(ctx, "user:1", "name", "alice")
fields, _ := c.HGetAll(ctx, "user:1")

// Sorted Set（排行榜）
c.ZAdd(ctx, "rank",
    cache.Z{Score: 100, Member: "alice"},
    cache.Z{Score: 200, Member: "bob"},
)
top, _ := c.ZRevRange(ctx, "rank", 0, 9)
```

## 切换实现

```go
var c cache.Cache

if os.Getenv("REDIS_ADDR") != "" {
    c = redisCache.New(redisCache.Config{Addr: os.Getenv("REDIS_ADDR")})
} else {
    c = memCache.New(memCache.Config{})
}

// 业务代码统一使用 cache.Cache 接口，无感知切换
```

## 可用实现

| 实现 | 包路径 | 特点 | 文档 |
|------|--------|------|------|
| **Memory** | `github.com/CXeon/tiles/cache/memory` | 无外部依赖，适合开发/测试 | [文档](memory/README.md) |
| **Redis** | `github.com/CXeon/tiles/cache/redis` | 基于 go-redis/v9，支持 Pipeline、Pub/Sub、Scan | [文档](redis/README.md) |

## 相关链接

- [Memory 实现](memory/README.md)
- [Redis 实现](redis/README.md)
- [tiles 项目主页](../README.md)
