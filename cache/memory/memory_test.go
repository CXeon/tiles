package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CXeon/tiles/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCache(t *testing.T) MemoryCache {
	t.Helper()
	c := New(Config{CleanupInterval: time.Minute})
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// ---- String ----

func TestMemoryCache_SetGet(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", "v", 0))

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)
}

func TestMemoryCache_Get_NotFound(t *testing.T) {
	c := newCache(t)
	_, err := c.Get(context.Background(), "missing")
	assert.True(t, errors.Is(err, cache.ErrKeyNotFound))
}

func TestMemoryCache_SetOverwritesType(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.HSet(ctx, "k", "f", "v"))
	// Set must overwrite regardless of previous type.
	require.NoError(t, c.Set(ctx, "k", "string", 0))
	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "string", got)
}

func TestMemoryCache_Delete(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "a", "1", 0))
	require.NoError(t, c.Set(ctx, "b", "2", 0))
	require.NoError(t, c.Delete(ctx, "a", "b", "nonexistent"))

	ok, _ := c.Exists(ctx, "a")
	assert.False(t, ok)
}

func TestMemoryCache_Expire_TTL(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", "v", 200*time.Millisecond))

	ttl, err := c.TTL(ctx, "k")
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))

	time.Sleep(250 * time.Millisecond)

	_, err = c.Get(ctx, "k")
	assert.True(t, errors.Is(err, cache.ErrKeyNotFound))
}

func TestMemoryCache_Expire_NoExpiry(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", "v", 0))
	ttl, err := c.TTL(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, cache.NoExpiration, ttl)
}

func TestMemoryCache_Expire_UpdateTTL(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", "v", time.Minute))
	require.NoError(t, c.Expire(ctx, "k", 200*time.Millisecond))

	time.Sleep(250 * time.Millisecond)
	_, err := c.Get(ctx, "k")
	assert.True(t, errors.Is(err, cache.ErrKeyNotFound))
}

func TestMemoryCache_SetNX(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	ok, err := c.SetNX(ctx, "k", "first", 0)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = c.SetNX(ctx, "k", "second", 0)
	require.NoError(t, err)
	assert.False(t, ok)

	got, _ := c.Get(ctx, "k")
	assert.Equal(t, "first", got)
}

func TestMemoryCache_GetDel(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", "v", 0))
	got, err := c.GetDel(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	_, err = c.Get(ctx, "k")
	assert.True(t, errors.Is(err, cache.ErrKeyNotFound))
}

// ---- Atomic ----

func TestMemoryCache_Incr(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	n, err := c.Incr(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = c.IncrBy(ctx, "counter", 9)
	require.NoError(t, err)
	assert.Equal(t, int64(10), n)

	n, err = c.Decr(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(9), n)

	n, err = c.DecrBy(ctx, "counter", 4)
	require.NoError(t, err)
	assert.Equal(t, int64(5), n)
}

func TestMemoryCache_Incr_NotInteger(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "k", "not-a-number", 0))
	_, err := c.Incr(ctx, "k")
	assert.True(t, errors.Is(err, cache.ErrNotInteger))
}

// ---- Hash ----

func TestMemoryCache_Hash(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.HSet(ctx, "h", "f1", "v1"))
	require.NoError(t, c.HSet(ctx, "h", "f2", "v2"))

	v, err := c.HGet(ctx, "h", "f1")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	all, err := c.HGetAll(ctx, "h")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"f1": "v1", "f2": "v2"}, all)

	ok, _ := c.HExists(ctx, "h", "f1")
	assert.True(t, ok)

	n, _ := c.HLen(ctx, "h")
	assert.Equal(t, int64(2), n)

	require.NoError(t, c.HDel(ctx, "h", "f1"))
	n, _ = c.HLen(ctx, "h")
	assert.Equal(t, int64(1), n)
}

func TestMemoryCache_HGetAll_MissingKey(t *testing.T) {
	c := newCache(t)
	all, err := c.HGetAll(context.Background(), "missing")
	require.NoError(t, err)
	assert.Empty(t, all)
}

// ---- List ----

func TestMemoryCache_List_LPush_RPush(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	// LPush "a","b","c" → head is "c"
	require.NoError(t, c.LPush(ctx, "l", "a", "b", "c"))
	all, err := c.LRange(ctx, "l", 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "b", "a"}, all)

	require.NoError(t, c.RPush(ctx, "l", "d", "e"))
	all, _ = c.LRange(ctx, "l", 0, -1)
	assert.Equal(t, []string{"c", "b", "a", "d", "e"}, all)

	n, _ := c.LLen(ctx, "l")
	assert.Equal(t, int64(5), n)
}

func TestMemoryCache_List_Pop(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.RPush(ctx, "l", "a", "b", "c"))

	head, err := c.LPop(ctx, "l")
	require.NoError(t, err)
	assert.Equal(t, "a", head)

	tail, err := c.RPop(ctx, "l")
	require.NoError(t, err)
	assert.Equal(t, "c", tail)
}

func TestMemoryCache_List_NegativeIndex(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.RPush(ctx, "l", "a", "b", "c", "d"))
	sub, err := c.LRange(ctx, "l", 1, -2)
	require.NoError(t, err)
	assert.Equal(t, []string{"b", "c"}, sub)
}

// ---- Set ----

func TestMemoryCache_Set_Ops(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.SAdd(ctx, "s", "a", "b", "c"))
	n, _ := c.SCard(ctx, "s")
	assert.Equal(t, int64(3), n)

	ok, _ := c.SIsMember(ctx, "s", "b")
	assert.True(t, ok)

	require.NoError(t, c.SRem(ctx, "s", "b"))
	ok, _ = c.SIsMember(ctx, "s", "b")
	assert.False(t, ok)

	members, err := c.SMembers(ctx, "s")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "c"}, members)
}

// ---- Sorted Set ----

func TestMemoryCache_ZSet(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.ZAdd(ctx, "z",
		cache.Z{Score: 3, Member: "c"},
		cache.Z{Score: 1, Member: "a"},
		cache.Z{Score: 2, Member: "b"},
	))

	members, err := c.ZRange(ctx, "z", 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, members)

	rev, err := c.ZRevRange(ctx, "z", 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "b", "a"}, rev)

	score, err := c.ZScore(ctx, "z", "b")
	require.NoError(t, err)
	assert.Equal(t, float64(2), score)

	rank, err := c.ZRank(ctx, "z", "b")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rank)

	n, _ := c.ZCard(ctx, "z")
	assert.Equal(t, int64(3), n)

	// Update score
	require.NoError(t, c.ZAdd(ctx, "z", cache.Z{Score: 10, Member: "a"}))
	members, _ = c.ZRange(ctx, "z", 0, -1)
	assert.Equal(t, []string{"b", "c", "a"}, members)

	require.NoError(t, c.ZRem(ctx, "z", "b"))
	n, _ = c.ZCard(ctx, "z")
	assert.Equal(t, int64(2), n)
}

func TestMemoryCache_ZRangeWithScores(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.ZAdd(ctx, "z",
		cache.Z{Score: 1, Member: "a"},
		cache.Z{Score: 2, Member: "b"},
	))

	zs, err := c.ZRangeWithScores(ctx, "z", 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []cache.Z{{Score: 1, Member: "a"}, {Score: 2, Member: "b"}}, zs)
}

// ---- Keys ----

func TestMemoryCache_Keys_Glob(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "user:1", "a", 0))
	require.NoError(t, c.Set(ctx, "user:2", "b", 0))
	require.NoError(t, c.Set(ctx, "order:1", "c", 0))

	keys, err := c.Keys(ctx, "user:*")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"user:1", "user:2"}, keys)

	all, err := c.Keys(ctx, "*")
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

// ---- WrongType ----

func TestMemoryCache_WrongType(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.HSet(ctx, "k", "f", "v"))

	_, err := c.Get(ctx, "k")
	assert.True(t, errors.Is(err, cache.ErrWrongType))

	_, err = c.LLen(ctx, "k")
	assert.True(t, errors.Is(err, cache.ErrWrongType))
}

// ---- Flush / ItemCount ----

func TestMemoryCache_Flush(t *testing.T) {
	c := newCache(t)
	ctx := context.Background()

	require.NoError(t, c.Set(ctx, "a", "1", 0))
	require.NoError(t, c.Set(ctx, "b", "2", 0))
	assert.Equal(t, 2, c.ItemCount())

	require.NoError(t, c.Flush())
	assert.Equal(t, 0, c.ItemCount())
}
