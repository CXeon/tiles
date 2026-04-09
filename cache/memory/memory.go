package memory

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/CXeon/tiles/cache"
)

// MemoryCache extends cache.Cache with in-memory-specific operations.
type MemoryCache interface {
	cache.Cache
	// Flush removes all keys from the cache.
	Flush() error
	// ItemCount returns the total number of stored entries,
	// including expired items not yet cleaned up.
	ItemCount() int
}

// Config holds configuration for the in-memory cache.
type Config struct {
	// CleanupInterval controls how often the background goroutine actively
	// removes expired keys. Defaults to 5 minutes if zero.
	CleanupInterval time.Duration
}

type valueKind int8

const (
	kindString valueKind = iota
	kindHash
	kindList
	kindSet
	kindZSet
)

type entry struct {
	kind      valueKind
	str       string
	hash      map[string]string
	list      []string
	set       map[string]struct{}
	zset      []cache.Z // maintained in ascending score order, then ascending member
	expiresAt time.Time
	hasExpiry bool
}

func (e *entry) expired() bool {
	return e.hasExpiry && time.Now().After(e.expiresAt)
}

type memCache struct {
	mu     sync.RWMutex
	items  map[string]*entry
	stopCh chan struct{}
}

// New creates a new in-memory cache and starts the background cleanup goroutine.
func New(cfg Config) MemoryCache {
	interval := cfg.CleanupInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	c := &memCache{
		items:  make(map[string]*entry),
		stopCh: make(chan struct{}),
	}
	go c.runCleanup(interval)
	return c
}

func (c *memCache) runCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *memCache) deleteExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.items {
		if e.hasExpiry && now.After(e.expiresAt) {
			delete(c.items, k)
		}
	}
}

// getEntry returns the live entry for key, or nil if missing or expired.
// Caller must hold at least a read lock.
func (c *memCache) getEntry(key string) *entry {
	e, ok := c.items[key]
	if !ok || e.expired() {
		return nil
	}
	return e
}

// getTyped returns a live entry of the expected kind.
// Returns ErrKeyNotFound or ErrWrongType on mismatch.
// Caller must hold at least a read lock.
func (c *memCache) getTyped(key string, kind valueKind) (*entry, error) {
	e := c.getEntry(key)
	if e == nil {
		return nil, cache.ErrKeyNotFound
	}
	if e.kind != kind {
		return nil, cache.ErrWrongType
	}
	return e, nil
}

// getOrCreate returns a live entry of the expected kind, creating one if absent.
// Returns ErrWrongType if the key holds a different type.
// Caller must hold the write lock.
func (c *memCache) getOrCreate(key string, kind valueKind) (*entry, error) {
	if e, ok := c.items[key]; ok && !e.expired() {
		if e.kind != kind {
			return nil, cache.ErrWrongType
		}
		return e, nil
	}
	e := &entry{kind: kind}
	switch kind {
	case kindHash:
		e.hash = make(map[string]string)
	case kindSet:
		e.set = make(map[string]struct{})
	}
	c.items[key] = e
	return e, nil
}

// ---- String ----

func (c *memCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := &entry{kind: kindString, str: value}
	if ttl > 0 {
		e.hasExpiry = true
		e.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = e
	return nil
}

func (c *memCache) Get(_ context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, err := c.getTyped(key, kindString)
	if err != nil {
		return "", err
	}
	return e.str, nil
}

func (c *memCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.items, k)
	}
	return nil
}

func (c *memCache) Exists(_ context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.getEntry(key) != nil, nil
}

func (c *memCache) Expire(_ context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.getEntry(key)
	if e == nil {
		return cache.ErrKeyNotFound
	}
	if ttl <= 0 {
		e.hasExpiry = false
	} else {
		e.hasExpiry = true
		e.expiresAt = time.Now().Add(ttl)
	}
	return nil
}

func (c *memCache) TTL(_ context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return 0, cache.ErrKeyNotFound
	}
	if !e.hasExpiry {
		return cache.NoExpiration, nil
	}
	remaining := time.Until(e.expiresAt)
	if remaining < 0 {
		return 0, cache.ErrKeyNotFound
	}
	return remaining, nil
}

func (c *memCache) SetNX(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok && !e.expired() {
		return false, nil
	}
	e := &entry{kind: kindString, str: value}
	if ttl > 0 {
		e.hasExpiry = true
		e.expiresAt = time.Now().Add(ttl)
	}
	c.items[key] = e
	return true, nil
}

func (c *memCache) GetDel(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getTyped(key, kindString)
	if err != nil {
		return "", err
	}
	val := e.str
	delete(c.items, key)
	return val, nil
}

// ---- Atomic ----

func (c *memCache) incrBy(key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || e.expired() {
		ne := &entry{kind: kindString, str: strconv.FormatInt(delta, 10)}
		c.items[key] = ne
		return delta, nil
	}
	if e.kind != kindString {
		return 0, cache.ErrWrongType
	}
	n, err := strconv.ParseInt(e.str, 10, 64)
	if err != nil {
		return 0, cache.ErrNotInteger
	}
	n += delta
	e.str = strconv.FormatInt(n, 10)
	return n, nil
}

func (c *memCache) Incr(_ context.Context, key string) (int64, error) {
	return c.incrBy(key, 1)
}

func (c *memCache) IncrBy(_ context.Context, key string, value int64) (int64, error) {
	return c.incrBy(key, value)
}

func (c *memCache) Decr(_ context.Context, key string) (int64, error) {
	return c.incrBy(key, -1)
}

func (c *memCache) DecrBy(_ context.Context, key string, value int64) (int64, error) {
	return c.incrBy(key, -value)
}

// ---- Hash ----

func (c *memCache) HSet(_ context.Context, key, field, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getOrCreate(key, kindHash)
	if err != nil {
		return err
	}
	e.hash[field] = value
	return nil
}

func (c *memCache) HGet(_ context.Context, key, field string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, err := c.getTyped(key, kindHash)
	if err != nil {
		return "", err
	}
	v, ok := e.hash[field]
	if !ok {
		return "", cache.ErrKeyNotFound
	}
	return v, nil
}

func (c *memCache) HGetAll(_ context.Context, key string) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return map[string]string{}, nil
	}
	if e.kind != kindHash {
		return nil, cache.ErrWrongType
	}
	result := make(map[string]string, len(e.hash))
	for k, v := range e.hash {
		result[k] = v
	}
	return result, nil
}

func (c *memCache) HDel(_ context.Context, key string, fields ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getTyped(key, kindHash)
	if errors.Is(err, cache.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, f := range fields {
		delete(e.hash, f)
	}
	if len(e.hash) == 0 {
		delete(c.items, key)
	}
	return nil
}

func (c *memCache) HExists(_ context.Context, key, field string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return false, nil
	}
	if e.kind != kindHash {
		return false, cache.ErrWrongType
	}
	_, ok := e.hash[field]
	return ok, nil
}

func (c *memCache) HLen(_ context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return 0, nil
	}
	if e.kind != kindHash {
		return 0, cache.ErrWrongType
	}
	return int64(len(e.hash)), nil
}

// ---- List ----

func (c *memCache) LPush(_ context.Context, key string, values ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getOrCreate(key, kindList)
	if err != nil {
		return err
	}
	// Redis LPUSH semantics: each value is pushed to the head left-to-right,
	// so the last value in the argument list ends up at the head.
	newList := make([]string, len(values)+len(e.list))
	for i, v := range values {
		newList[len(values)-1-i] = v
	}
	copy(newList[len(values):], e.list)
	e.list = newList
	return nil
}

func (c *memCache) RPush(_ context.Context, key string, values ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getOrCreate(key, kindList)
	if err != nil {
		return err
	}
	e.list = append(e.list, values...)
	return nil
}

func (c *memCache) LPop(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getTyped(key, kindList)
	if err != nil {
		return "", err
	}
	if len(e.list) == 0 {
		return "", cache.ErrKeyNotFound
	}
	val := e.list[0]
	e.list = e.list[1:]
	if len(e.list) == 0 {
		delete(c.items, key)
	}
	return val, nil
}

func (c *memCache) RPop(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getTyped(key, kindList)
	if err != nil {
		return "", err
	}
	if len(e.list) == 0 {
		return "", cache.ErrKeyNotFound
	}
	last := len(e.list) - 1
	val := e.list[last]
	e.list = e.list[:last]
	if len(e.list) == 0 {
		delete(c.items, key)
	}
	return val, nil
}

func (c *memCache) LRange(_ context.Context, key string, start, stop int64) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return []string{}, nil
	}
	if e.kind != kindList {
		return nil, cache.ErrWrongType
	}
	return sliceRange(e.list, start, stop), nil
}

func (c *memCache) LLen(_ context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return 0, nil
	}
	if e.kind != kindList {
		return 0, cache.ErrWrongType
	}
	return int64(len(e.list)), nil
}

// ---- Set ----

func (c *memCache) SAdd(_ context.Context, key string, members ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getOrCreate(key, kindSet)
	if err != nil {
		return err
	}
	for _, m := range members {
		e.set[m] = struct{}{}
	}
	return nil
}

func (c *memCache) SRem(_ context.Context, key string, members ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getTyped(key, kindSet)
	if errors.Is(err, cache.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, m := range members {
		delete(e.set, m)
	}
	if len(e.set) == 0 {
		delete(c.items, key)
	}
	return nil
}

func (c *memCache) SMembers(_ context.Context, key string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return []string{}, nil
	}
	if e.kind != kindSet {
		return nil, cache.ErrWrongType
	}
	result := make([]string, 0, len(e.set))
	for m := range e.set {
		result = append(result, m)
	}
	return result, nil
}

func (c *memCache) SIsMember(_ context.Context, key, member string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return false, nil
	}
	if e.kind != kindSet {
		return false, cache.ErrWrongType
	}
	_, ok := e.set[member]
	return ok, nil
}

func (c *memCache) SCard(_ context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return 0, nil
	}
	if e.kind != kindSet {
		return 0, cache.ErrWrongType
	}
	return int64(len(e.set)), nil
}

// ---- Sorted Set ----

func (c *memCache) ZAdd(_ context.Context, key string, members ...cache.Z) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getOrCreate(key, kindZSet)
	if err != nil {
		return err
	}
	for _, m := range members {
		updated := false
		for i, z := range e.zset {
			if z.Member == m.Member {
				e.zset[i].Score = m.Score
				updated = true
				break
			}
		}
		if !updated {
			e.zset = append(e.zset, m)
		}
	}
	sort.Slice(e.zset, func(i, j int) bool {
		if e.zset[i].Score != e.zset[j].Score {
			return e.zset[i].Score < e.zset[j].Score
		}
		return e.zset[i].Member < e.zset[j].Member
	})
	return nil
}

func (c *memCache) ZRem(_ context.Context, key string, members ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, err := c.getTyped(key, kindZSet)
	if errors.Is(err, cache.ErrKeyNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	toRemove := make(map[string]struct{}, len(members))
	for _, m := range members {
		toRemove[m] = struct{}{}
	}
	filtered := e.zset[:0]
	for _, z := range e.zset {
		if _, rem := toRemove[z.Member]; !rem {
			filtered = append(filtered, z)
		}
	}
	e.zset = filtered
	if len(e.zset) == 0 {
		delete(c.items, key)
	}
	return nil
}

func (c *memCache) ZScore(_ context.Context, key, member string) (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, err := c.getTyped(key, kindZSet)
	if err != nil {
		return 0, err
	}
	for _, z := range e.zset {
		if z.Member == member {
			return z.Score, nil
		}
	}
	return 0, cache.ErrKeyNotFound
}

func (c *memCache) ZRange(_ context.Context, key string, start, stop int64) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return []string{}, nil
	}
	if e.kind != kindZSet {
		return nil, cache.ErrWrongType
	}
	members := make([]string, len(e.zset))
	for i, z := range e.zset {
		members[i] = z.Member
	}
	return sliceRange(members, start, stop), nil
}

func (c *memCache) ZRevRange(_ context.Context, key string, start, stop int64) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return []string{}, nil
	}
	if e.kind != kindZSet {
		return nil, cache.ErrWrongType
	}
	reversed := make([]string, len(e.zset))
	for i, z := range e.zset {
		reversed[len(e.zset)-1-i] = z.Member
	}
	return sliceRange(reversed, start, stop), nil
}

func (c *memCache) ZRangeWithScores(_ context.Context, key string, start, stop int64) ([]cache.Z, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return []cache.Z{}, nil
	}
	if e.kind != kindZSet {
		return nil, cache.ErrWrongType
	}
	sub := sliceRange(e.zset, start, stop)
	result := make([]cache.Z, len(sub))
	copy(result, sub)
	return result, nil
}

func (c *memCache) ZCard(_ context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := c.getEntry(key)
	if e == nil {
		return 0, nil
	}
	if e.kind != kindZSet {
		return 0, cache.ErrWrongType
	}
	return int64(len(e.zset)), nil
}

func (c *memCache) ZRank(_ context.Context, key, member string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, err := c.getTyped(key, kindZSet)
	if err != nil {
		return 0, err
	}
	for i, z := range e.zset {
		if z.Member == member {
			return int64(i), nil
		}
	}
	return 0, cache.ErrKeyNotFound
}

// ---- Keys ----

func (c *memCache) Keys(_ context.Context, pattern string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var result []string
	for k, e := range c.items {
		if !e.expired() && matchGlob(pattern, k) {
			result = append(result, k)
		}
	}
	return result, nil
}

// ---- Lifecycle ----

func (c *memCache) Close() error {
	close(c.stopCh)
	return nil
}

// ---- MemoryCache extras ----

func (c *memCache) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*entry)
	return nil
}

func (c *memCache) ItemCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// ---- Helpers ----

// sliceRange applies Redis-style index slicing: 0-based, negative indices count
// from the tail (-1 is the last element).
func sliceRange[T any](s []T, start, stop int64) []T {
	n := int64(len(s))
	if n == 0 {
		return []T{}
	}
	if start < 0 {
		start += n
	}
	if stop < 0 {
		stop += n
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop {
		return []T{}
	}
	result := make([]T, stop-start+1)
	copy(result, s[start:stop+1])
	return result
}

// matchGlob reports whether key matches the glob pattern.
// Supports * (any sequence of characters) and ? (any single character).
func matchGlob(pattern, key string) bool {
	px, kx := 0, 0
	starPx, starKx := -1, 0
	for kx < len(key) {
		if px < len(pattern) && (pattern[px] == '?' || pattern[px] == key[kx]) {
			px++
			kx++
		} else if px < len(pattern) && pattern[px] == '*' {
			starPx = px
			starKx = kx
			px++
		} else if starPx >= 0 {
			starKx++
			kx = starKx
			px = starPx + 1
		} else {
			return false
		}
	}
	for px < len(pattern) && pattern[px] == '*' {
		px++
	}
	return px == len(pattern)
}
