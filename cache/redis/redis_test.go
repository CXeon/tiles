package redis

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CXeon/tiles/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These are integration tests that require a running Redis instance.
// Set REDIS_TEST_ADDR (e.g. "localhost:6379") to enable them.
func newTestCache(t *testing.T) RedisCache {
	t.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("REDIS_TEST_ADDR not set, skipping Redis integration tests")
	}
	c := New(Config{Addr: addr})
	ctx := context.Background()
	if err := c.Client().Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable at %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// keyPrefix returns a unique key prefix for each test to avoid cross-test pollution.
func kp(t *testing.T, key string) string {
	return t.Name() + ":" + key
}

func TestRedisCache_SetGet(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "k")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	require.NoError(t, c.Set(ctx, k, "hello", 0))
	got, err := c.Get(ctx, k)
	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestRedisCache_Get_NotFound(t *testing.T) {
	c := newTestCache(t)
	_, err := c.Get(context.Background(), kp(t, "missing"))
	assert.True(t, errors.Is(err, cache.ErrKeyNotFound))
}

func TestRedisCache_Expire_TTL(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "k")

	require.NoError(t, c.Set(ctx, k, "v", 300*time.Millisecond))
	ttl, err := c.TTL(ctx, k)
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))

	time.Sleep(400 * time.Millisecond)
	_, err = c.Get(ctx, k)
	assert.True(t, errors.Is(err, cache.ErrKeyNotFound))
}

func TestRedisCache_SetNX(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "k")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	ok, err := c.SetNX(ctx, k, "first", 0)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = c.SetNX(ctx, k, "second", 0)
	require.NoError(t, err)
	assert.False(t, ok)

	got, _ := c.Get(ctx, k)
	assert.Equal(t, "first", got)
}

func TestRedisCache_GetDel(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "k")

	require.NoError(t, c.Set(ctx, k, "v", 0))
	got, err := c.GetDel(ctx, k)
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	_, err = c.Get(ctx, k)
	assert.True(t, errors.Is(err, cache.ErrKeyNotFound))
}

func TestRedisCache_Atomic(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "counter")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	n, err := c.Incr(ctx, k)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = c.IncrBy(ctx, k, 9)
	require.NoError(t, err)
	assert.Equal(t, int64(10), n)

	n, err = c.Decr(ctx, k)
	require.NoError(t, err)
	assert.Equal(t, int64(9), n)

	n, err = c.DecrBy(ctx, k, 4)
	require.NoError(t, err)
	assert.Equal(t, int64(5), n)
}

func TestRedisCache_Hash(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "h")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	require.NoError(t, c.HSet(ctx, k, "f1", "v1"))
	require.NoError(t, c.HSet(ctx, k, "f2", "v2"))

	v, err := c.HGet(ctx, k, "f1")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	all, err := c.HGetAll(ctx, k)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"f1": "v1", "f2": "v2"}, all)

	n, _ := c.HLen(ctx, k)
	assert.Equal(t, int64(2), n)

	require.NoError(t, c.HDel(ctx, k, "f1"))
	ok, _ := c.HExists(ctx, k, "f1")
	assert.False(t, ok)
}

func TestRedisCache_List(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "l")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	// LPush "a","b","c" → Redis list head is "c"
	require.NoError(t, c.LPush(ctx, k, "a", "b", "c"))
	all, err := c.LRange(ctx, k, 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "b", "a"}, all)

	require.NoError(t, c.RPush(ctx, k, "d"))
	n, _ := c.LLen(ctx, k)
	assert.Equal(t, int64(4), n)

	head, _ := c.LPop(ctx, k)
	assert.Equal(t, "c", head)

	tail, _ := c.RPop(ctx, k)
	assert.Equal(t, "d", tail)
}

func TestRedisCache_Set_Ops(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "s")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	require.NoError(t, c.SAdd(ctx, k, "a", "b", "c"))
	n, _ := c.SCard(ctx, k)
	assert.Equal(t, int64(3), n)

	ok, _ := c.SIsMember(ctx, k, "b")
	assert.True(t, ok)

	require.NoError(t, c.SRem(ctx, k, "b"))
	members, err := c.SMembers(ctx, k)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "c"}, members)
}

func TestRedisCache_ZSet(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "z")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	require.NoError(t, c.ZAdd(ctx, k,
		cache.Z{Score: 3, Member: "c"},
		cache.Z{Score: 1, Member: "a"},
		cache.Z{Score: 2, Member: "b"},
	))

	members, err := c.ZRange(ctx, k, 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, members)

	rev, err := c.ZRevRange(ctx, k, 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "b", "a"}, rev)

	score, _ := c.ZScore(ctx, k, "b")
	assert.Equal(t, float64(2), score)

	rank, _ := c.ZRank(ctx, k, "b")
	assert.Equal(t, int64(1), rank)

	n, _ := c.ZCard(ctx, k)
	assert.Equal(t, int64(3), n)

	zs, err := c.ZRangeWithScores(ctx, k, 0, 1)
	require.NoError(t, err)
	assert.Len(t, zs, 2)
	assert.Equal(t, float64(1), zs[0].Score)

	require.NoError(t, c.ZRem(ctx, k, "a"))
	n, _ = c.ZCard(ctx, k)
	assert.Equal(t, int64(2), n)
}

func TestRedisCache_Pipeline(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k1, k2 := kp(t, "k1"), kp(t, "k2")
	t.Cleanup(func() { _ = c.Delete(ctx, k1, k2) })

	cmds, err := c.Pipeline(ctx, func(pipe Pipeliner) error {
		pipe.Set(ctx, k1, "v1", 0)
		pipe.Set(ctx, k2, "v2", 0)
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, cmds, 2)

	v1, _ := c.Get(ctx, k1)
	v2, _ := c.Get(ctx, k2)
	assert.Equal(t, "v1", v1)
	assert.Equal(t, "v2", v2)
}

func TestRedisCache_TxPipeline(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()
	k := kp(t, "k")
	t.Cleanup(func() { _ = c.Delete(ctx, k) })

	_, err := c.TxPipeline(ctx, func(pipe Pipeliner) error {
		pipe.Set(ctx, k, "atomic", 0)
		return nil
	})
	require.NoError(t, err)

	v, _ := c.Get(ctx, k)
	assert.Equal(t, "atomic", v)
}

func TestRedisCache_Scan(t *testing.T) {
	c := newTestCache(t)
	ctx := context.Background()

	prefix := kp(t, "scan")
	keys := []string{prefix + ":1", prefix + ":2", prefix + ":3"}
	for _, k := range keys {
		require.NoError(t, c.Set(ctx, k, "v", time.Minute))
	}
	t.Cleanup(func() { _ = c.Delete(ctx, keys...) })

	var found []string
	var cursor uint64
	for {
		batch, next, err := c.Scan(ctx, cursor, prefix+":*", 10)
		require.NoError(t, err)
		found = append(found, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	assert.ElementsMatch(t, keys, found)
}
