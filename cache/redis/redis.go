package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CXeon/tiles/cache"
	goredis "github.com/redis/go-redis/v9"
)

// Pipeliner is an alias for go-redis Pipeliner.
// Commands queued on a Pipeliner are sent to Redis in a single round trip
// when the enclosing Pipeline or TxPipeline function returns.
type Pipeliner = goredis.Pipeliner

// RedisCache extends cache.Cache with Redis-specific features that cannot be
// meaningfully simulated in other backends.
type RedisCache interface {
	cache.Cache

	// Pipeline sends all commands queued by fn in a single round trip (no MULTI/EXEC).
	// Returns the results of all commands after execution.
	Pipeline(ctx context.Context, fn func(pipe Pipeliner) error) ([]goredis.Cmder, error)

	// TxPipeline wraps commands in MULTI/EXEC for atomic execution.
	TxPipeline(ctx context.Context, fn func(pipe Pipeliner) error) ([]goredis.Cmder, error)

	// Publish sends a message to a channel.
	Publish(ctx context.Context, channel string, message any) error

	// Subscribe returns a PubSub handle for the given channels.
	// The caller is responsible for closing the returned PubSub.
	Subscribe(ctx context.Context, channels ...string) *goredis.PubSub

	// Scan iterates over keys matching match using a cursor.
	// Returns the next cursor and the keys found in this page.
	// Use cursor=0 to start; iteration is complete when the returned cursor is 0.
	Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error)

	// Client exposes the underlying go-redis client for advanced use cases
	// such as Lua scripts, WATCH/CAS transactions, or commands not covered
	// by this interface.
	Client() *goredis.Client
}

// Config holds configuration for the Redis cache client.
type Config struct {
	Addr         string        // e.g. "localhost:6379"
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type redisCache struct {
	client *goredis.Client
}

// New creates a Redis cache client. The connection is lazy; use Client().Ping
// or any cache operation to verify connectivity.
func New(cfg Config) RedisCache {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
	return &redisCache{client: rdb}
}

// wrapErr normalises common Redis errors to cache sentinel errors.
func wrapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, goredis.Nil) {
		return cache.ErrKeyNotFound
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "WRONGTYPE") {
		return cache.ErrWrongType
	}
	if strings.Contains(msg, "not an integer") {
		return cache.ErrNotInteger
	}
	return err
}

// toGoZ converts []cache.Z to []goredis.Z.
func toGoZ(members []cache.Z) []goredis.Z {
	out := make([]goredis.Z, len(members))
	for i, m := range members {
		out[i] = goredis.Z{Score: m.Score, Member: m.Member}
	}
	return out
}

// toAny converts a []string to []any for go-redis variadic APIs.
func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// fromGoZ converts []goredis.Z to []cache.Z.
func fromGoZ(members []goredis.Z) []cache.Z {
	out := make([]cache.Z, len(members))
	for i, m := range members {
		out[i] = cache.Z{Score: m.Score, Member: fmt.Sprint(m.Member)}
	}
	return out
}

// ---- String ----

func (r *redisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return wrapErr(r.client.Set(ctx, key, value, ttl).Err())
}

func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	v, err := r.client.Get(ctx, key).Result()
	return v, wrapErr(err)
}

func (r *redisCache) Delete(ctx context.Context, keys ...string) error {
	return wrapErr(r.client.Del(ctx, keys...).Err())
}

func (r *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, wrapErr(err)
}

func (r *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	ok, err := r.client.Expire(ctx, key, ttl).Result()
	if err != nil {
		return wrapErr(err)
	}
	if !ok {
		return cache.ErrKeyNotFound
	}
	return nil
}

func (r *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, wrapErr(err)
	}
	// go-redis returns -2 for missing keys, -1 for keys with no expiry.
	if d == -2*time.Second {
		return 0, cache.ErrKeyNotFound
	}
	if d == -1*time.Second {
		return cache.NoExpiration, nil
	}
	return d, nil
}

func (r *redisCache) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, key, value, ttl).Result()
	return ok, wrapErr(err)
}

func (r *redisCache) GetDel(ctx context.Context, key string) (string, error) {
	v, err := r.client.GetDel(ctx, key).Result()
	return v, wrapErr(err)
}

// ---- Atomic ----

func (r *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	n, err := r.client.Incr(ctx, key).Result()
	return n, wrapErr(err)
}

func (r *redisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	n, err := r.client.IncrBy(ctx, key, value).Result()
	return n, wrapErr(err)
}

func (r *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	n, err := r.client.Decr(ctx, key).Result()
	return n, wrapErr(err)
}

func (r *redisCache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	n, err := r.client.DecrBy(ctx, key, value).Result()
	return n, wrapErr(err)
}

// ---- Hash ----

func (r *redisCache) HSet(ctx context.Context, key, field, value string) error {
	return wrapErr(r.client.HSet(ctx, key, field, value).Err())
}

func (r *redisCache) HGet(ctx context.Context, key, field string) (string, error) {
	v, err := r.client.HGet(ctx, key, field).Result()
	return v, wrapErr(err)
}

func (r *redisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	m, err := r.client.HGetAll(ctx, key).Result()
	return m, wrapErr(err)
}

func (r *redisCache) HDel(ctx context.Context, key string, fields ...string) error {
	return wrapErr(r.client.HDel(ctx, key, fields...).Err())
}

func (r *redisCache) HExists(ctx context.Context, key, field string) (bool, error) {
	ok, err := r.client.HExists(ctx, key, field).Result()
	return ok, wrapErr(err)
}

func (r *redisCache) HLen(ctx context.Context, key string) (int64, error) {
	n, err := r.client.HLen(ctx, key).Result()
	return n, wrapErr(err)
}

// ---- List ----

func (r *redisCache) LPush(ctx context.Context, key string, values ...string) error {
	return wrapErr(r.client.LPush(ctx, key, toAny(values)...).Err())
}

func (r *redisCache) RPush(ctx context.Context, key string, values ...string) error {
	return wrapErr(r.client.RPush(ctx, key, toAny(values)...).Err())
}

func (r *redisCache) LPop(ctx context.Context, key string) (string, error) {
	v, err := r.client.LPop(ctx, key).Result()
	return v, wrapErr(err)
}

func (r *redisCache) RPop(ctx context.Context, key string) (string, error) {
	v, err := r.client.RPop(ctx, key).Result()
	return v, wrapErr(err)
}

func (r *redisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vs, err := r.client.LRange(ctx, key, start, stop).Result()
	return vs, wrapErr(err)
}

func (r *redisCache) LLen(ctx context.Context, key string) (int64, error) {
	n, err := r.client.LLen(ctx, key).Result()
	return n, wrapErr(err)
}

// ---- Set ----

func (r *redisCache) SAdd(ctx context.Context, key string, members ...string) error {
	return wrapErr(r.client.SAdd(ctx, key, toAny(members)...).Err())
}

func (r *redisCache) SRem(ctx context.Context, key string, members ...string) error {
	return wrapErr(r.client.SRem(ctx, key, toAny(members)...).Err())
}

func (r *redisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	vs, err := r.client.SMembers(ctx, key).Result()
	return vs, wrapErr(err)
}

func (r *redisCache) SIsMember(ctx context.Context, key, member string) (bool, error) {
	ok, err := r.client.SIsMember(ctx, key, member).Result()
	return ok, wrapErr(err)
}

func (r *redisCache) SCard(ctx context.Context, key string) (int64, error) {
	n, err := r.client.SCard(ctx, key).Result()
	return n, wrapErr(err)
}

// ---- Sorted Set ----

func (r *redisCache) ZAdd(ctx context.Context, key string, members ...cache.Z) error {
	return wrapErr(r.client.ZAdd(ctx, key, toGoZ(members)...).Err())
}

func (r *redisCache) ZRem(ctx context.Context, key string, members ...string) error {
	return wrapErr(r.client.ZRem(ctx, key, toAny(members)...).Err())
}

func (r *redisCache) ZScore(ctx context.Context, key, member string) (float64, error) {
	s, err := r.client.ZScore(ctx, key, member).Result()
	return s, wrapErr(err)
}

func (r *redisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vs, err := r.client.ZRange(ctx, key, start, stop).Result()
	return vs, wrapErr(err)
}

func (r *redisCache) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vs, err := r.client.ZRevRange(ctx, key, start, stop).Result()
	return vs, wrapErr(err)
}

func (r *redisCache) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]cache.Z, error) {
	zs, err := r.client.ZRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, wrapErr(err)
	}
	return fromGoZ(zs), nil
}

func (r *redisCache) ZCard(ctx context.Context, key string) (int64, error) {
	n, err := r.client.ZCard(ctx, key).Result()
	return n, wrapErr(err)
}

func (r *redisCache) ZRank(ctx context.Context, key, member string) (int64, error) {
	n, err := r.client.ZRank(ctx, key, member).Result()
	return n, wrapErr(err)
}

// ---- Keys ----

func (r *redisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	vs, err := r.client.Keys(ctx, pattern).Result()
	return vs, wrapErr(err)
}

// ---- Redis-only ----

func (r *redisCache) Pipeline(ctx context.Context, fn func(pipe Pipeliner) error) ([]goredis.Cmder, error) {
	return r.client.Pipelined(ctx, fn)
}

func (r *redisCache) TxPipeline(ctx context.Context, fn func(pipe Pipeliner) error) ([]goredis.Cmder, error) {
	return r.client.TxPipelined(ctx, fn)
}

func (r *redisCache) Publish(ctx context.Context, channel string, message any) error {
	return wrapErr(r.client.Publish(ctx, channel, message).Err())
}

func (r *redisCache) Subscribe(ctx context.Context, channels ...string) *goredis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

func (r *redisCache) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	keys, next, err := r.client.Scan(ctx, cursor, match, count).Result()
	return keys, next, wrapErr(err)
}

func (r *redisCache) Client() *goredis.Client {
	return r.client
}

// ---- Lifecycle ----

func (r *redisCache) Close() error {
	return r.client.Close()
}
