package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrKeyNotFound is returned when the requested key does not exist or has expired.
	ErrKeyNotFound = errors.New("tiles/cache: key not found")
	// ErrWrongType is returned when an operation is performed on a key holding the wrong data type.
	ErrWrongType = errors.New("tiles/cache: WRONGTYPE operation against a key holding the wrong kind of value")
	// ErrNotInteger is returned when an atomic operation is attempted on a non-integer value.
	ErrNotInteger = errors.New("tiles/cache: value is not an integer or out of range")
)

// NoExpiration indicates that a key should never expire.
const NoExpiration = time.Duration(0)

// Z represents a sorted set member with its score.
type Z struct {
	Score  float64
	Member string
}

// Cache is the base interface implemented by all cache backends.
//
// It covers the common denominator of Redis features that can be faithfully
// simulated in memory, enabling seamless backend substitution. Use a type
// assertion to access backend-specific extensions:
//
//	if rc, ok := c.(redis.RedisCache); ok {
//	    rc.Pipeline(ctx, func(p redis.Pipeliner) error { ... })
//	}
type Cache interface {
	// --- String ---

	// Set stores a string value with an optional TTL. ttl=0 means no expiration.
	// Overwrites any existing value and type for the key.
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Get retrieves a string value. Returns ErrKeyNotFound if absent or expired.
	Get(ctx context.Context, key string) (string, error)
	// Delete removes one or more keys. Non-existent keys are silently ignored.
	Delete(ctx context.Context, keys ...string) error
	// Exists reports whether the key exists and has not expired.
	Exists(ctx context.Context, key string) (bool, error)
	// Expire sets or updates the TTL of an existing key.
	// Returns ErrKeyNotFound if the key is absent.
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// TTL returns the remaining lifetime of a key.
	// Returns NoExpiration if the key has no expiry, ErrKeyNotFound if absent.
	TTL(ctx context.Context, key string) (time.Duration, error)
	// SetNX sets the value only if the key does not already exist.
	// Returns true if the value was set, false if the key already existed.
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	// GetDel atomically retrieves and deletes a key.
	// Returns ErrKeyNotFound if absent.
	GetDel(ctx context.Context, key string) (string, error)

	// --- Atomic counters (value stored as decimal string, same as Redis) ---

	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	DecrBy(ctx context.Context, key string, value int64) (int64, error)

	// --- Hash ---

	// HSet sets a single field in a hash. Creates the hash if it does not exist.
	HSet(ctx context.Context, key, field, value string) error
	// HGet retrieves a field value. Returns ErrKeyNotFound if key or field is absent.
	HGet(ctx context.Context, key, field string) (string, error)
	// HGetAll returns all field-value pairs. Returns an empty map if key is absent.
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	// HDel removes fields from a hash. Non-existent fields are ignored.
	HDel(ctx context.Context, key string, fields ...string) error
	// HExists reports whether a field exists in the hash.
	HExists(ctx context.Context, key, field string) (bool, error)
	// HLen returns the number of fields in the hash.
	HLen(ctx context.Context, key string) (int64, error)

	// --- List ---

	// LPush inserts values at the head of the list.
	// Equivalent to Redis LPUSH: LPush(ctx, key, "a","b","c") → list head is "c".
	LPush(ctx context.Context, key string, values ...string) error
	// RPush appends values to the tail of the list.
	RPush(ctx context.Context, key string, values ...string) error
	// LPop removes and returns the head element. Returns ErrKeyNotFound if empty.
	LPop(ctx context.Context, key string) (string, error)
	// RPop removes and returns the tail element. Returns ErrKeyNotFound if empty.
	RPop(ctx context.Context, key string) (string, error)
	// LRange returns a sub-range of the list (0-based, negative indices count from tail).
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	// LLen returns the length of the list.
	LLen(ctx context.Context, key string) (int64, error)

	// --- Set ---

	SAdd(ctx context.Context, key string, members ...string) error
	SRem(ctx context.Context, key string, members ...string) error
	// SMembers returns all members. Returns an empty slice if key is absent.
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key, member string) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)

	// --- Sorted Set ---

	// ZAdd adds or updates members. Maintains ascending score order.
	ZAdd(ctx context.Context, key string, members ...Z) error
	ZRem(ctx context.Context, key string, members ...string) error
	// ZScore returns the score of a member. Returns ErrKeyNotFound if absent.
	ZScore(ctx context.Context, key, member string) (float64, error)
	// ZRange returns members in ascending score order (0-based, negative indices supported).
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	// ZRevRange returns members in descending score order.
	ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	// ZRangeWithScores is like ZRange but includes scores.
	ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]Z, error)
	// ZCard returns the number of members.
	ZCard(ctx context.Context, key string) (int64, error)
	// ZRank returns the 0-based rank of a member in ascending order.
	// Returns ErrKeyNotFound if the member is absent.
	ZRank(ctx context.Context, key, member string) (int64, error)

	// --- Keys ---

	// Keys returns all keys matching the glob pattern.
	// Supports * (any sequence) and ? (any single char).
	Keys(ctx context.Context, pattern string) ([]string, error)

	Close() error
}
