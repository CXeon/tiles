# Cache - Memory Implementation

基于 Go 标准库实现的内存缓存，支持完整的 Redis 常用操作语义，可作为 Redis 的开发/测试替代品。

## 特性

- **Redis 语义兼容**：String、Hash、List、Set、Sorted Set 操作与 Redis 行为一致
- **TTL 支持**：懒惰过期��访问时检查）+ 后台定期清理双重机制
- **并发安全**：全局 `sync.RWMutex` 保护，读多写少场景性能友好
- **类型隔离**：同一 key 只能持有一种数据类型，操作类型不匹配返回 `ErrWrongType`
- **自动清理空容器**：List / Set / ZSet / Hash 为空时自动删除 key，与 Redis 行为一致
- **Glob 模式匹配**：`Keys` 方法支持 `*`（任意字符序列）和 `?`（单个字符）通配符

## 快速开始

```go
import (
    "context"
    "time"

    "github.com/CXeon/tiles/cache"
    memCache "github.com/CXeon/tiles/cache/memory"
)

ctx := context.Background()

// 创建内存缓存
c := memCache.New(memCache.Config{
    CleanupInterval: 5 * time.Minute, // 后台清理间隔，默认 5 分钟
})
defer c.Close()

// String
c.Set(ctx, "name", "alice", time.Hour)
name, _ := c.Get(ctx, "name")

// Hash
c.HSet(ctx, "user:1", "name", "alice")
c.HSet(ctx, "user:1", "age", "30")
fields, _ := c.HGetAll(ctx, "user:1")

// 计数器
c.Incr(ctx, "pv")
c.IncrBy(ctx, "pv", 10)

// Sorted Set（排行榜）
c.ZAdd(ctx, "leaderboard",
    cache.Z{Score: 100, Member: "alice"},
    cache.Z{Score: 200, Member: "bob"},
)
top, _ := c.ZRevRange(ctx, "leaderboard", 0, 9) // 前 10 名
```

## 配置说明

```go
type Config struct {
    // CleanupInterval 是后台清理协程的运行间隔
    // 过期 key 在访问时会即时失效（懒惰过期），后台协程负责回收内存
    // 默认值：5 分钟
    CleanupInterval time.Duration
}
```

## 扩展接口

`MemoryCache` 在 `cache.Cache` 基础上提供内存特有操作：

```go
// 清空所有数据（适用于测试用例隔离）
c.Flush()

// 查询当前条目总数（含已过期但未清理的条目）
count := c.ItemCount()
```

## 使用场景

### 作为 Redis 的临时替代

```go
var c cache.Cache

if redisAvailable {
    c = redisCache.New(redisCache.Config{Addr: "localhost:6379"})
} else {
    c = memCache.New(memCache.Config{})
}

// 业务代码使用 cache.Cache 接口，无需关心底层实现
c.Set(ctx, "key", "value", time.Hour)
```

### 单元测试

```go
func TestMyService(t *testing.T) {
    c := memCache.New(memCache.Config{})
    defer c.Close()

    svc := NewMyService(c) // 注入 cache.Cache 接口
    // ...

    // 测试用例间隔离
    c.Flush()
}
```

### SetNX 分布式锁模拟

> 注意：内存实现仅保证单进程内的原子性，不适用于跨进程场景。

```go
ok, _ := c.SetNX(ctx, "lock:order:123", "1", 30*time.Second)
if !ok {
    return errors.New("获取锁失败")
}
defer c.Delete(ctx, "lock:order:123")
```

### Keys 模式查询

```go
// 查询所有用户 key
keys, _ := c.Keys(ctx, "user:*")

// 查询特定格式 key
keys, _ = c.Keys(ctx, "order:202?:*")
```

## 数据类型操作对照

| 类型 | 操作 | 说明 |
|------|------|------|
| String | `Set` / `Get` / `SetNX` / `GetDel` | 基础键值存储 |
| Atomic | `Incr` / `IncrBy` / `Decr` / `DecrBy` | 整数值原子操作，值以字符串存储 |
| Hash | `HSet` / `HGet` / `HGetAll` / `HDel` / `HExists` / `HLen` | 字段-值映射 |
| List | `LPush` / `RPush` / `LPop` / `RPop` / `LRange` / `LLen` | 双端队列，支持负索引 |
| Set | `SAdd` / `SRem` / `SMembers` / `SIsMember` / `SCard` | 无序不重复集合 |
| ZSet | `ZAdd` / `ZRem` / `ZScore` / `ZRange` / `ZRevRange` / `ZCard` / `ZRank` | 按���数排序集合 |

### LPush 顺序说明

与 Redis 行为一致：`LPush(ctx, key, "a", "b", "c")` 后，列表从头到尾为 `[c, b, a]`。

### 负索引说明

`LRange` / `ZRange` 等支持负索引：`-1` 代表最后一个元素，`-2` 代表倒数第二个，以此类推。

## 运行测试

```bash
go test github.com/CXeon/tiles/cache/memory/... -v
```

## 相关链接

- [Cache 接口定义](../cache.go)
- [Redis 实现](../redis/README.md)
- [go-redis](https://github.com/redis/go-redis)
