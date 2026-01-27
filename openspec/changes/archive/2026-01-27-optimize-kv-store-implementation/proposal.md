# Change: Optimize KV Store Implementation

## Why
The current `KvStore` interface lacks support for batch deletion by prefix, which leads to inefficient iterative deletions in the handler layer. Additionally, the Redis implementation of `GetByPrefix` is inefficient as it performs individual `GET` calls for each key found.

## What Changes
- Remove `Add(key string, value []byte) error` from `KvStore` interface as it is unused.
- Add `DeleteByPrefix(prefix string) error` to the `KvStore` interface in `gateway/traefik/kv_store/kv_store.go`.
- Implement `DeleteByPrefix` in `Redis` backend using the standard Go library (`SCAN` + `DEL`) for better maintainability.
- Implement `DeleteByPrefix` in `Consul` backend using `DeleteTree`.
- Optimize `Redis.GetByPrefix` to use `MGET` instead of individual `GET` calls.
- Implement new `KvStore` backends for `Etcd` and `Zookeeper` to expand gateway compatibility.
- Remove `token` support from `Consul` implementation, focusing on username/password authentication.
- Standardize `KvStore` interface methods to include `context.Context` for better lifecycle management.
- Define standard error types (e.g., `ErrKeyNotFound`) in `KvStore` to decouple handler logic from backend-specific errors.
- Enhance `Provider` configuration to support connection pooling and timeout settings for better production readiness.

## Impact
- Affected specs: `kv-store`
- Affected code: `gateway/traefik/kv_store/*`
