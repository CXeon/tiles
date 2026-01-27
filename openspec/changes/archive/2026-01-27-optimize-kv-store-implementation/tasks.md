## 1. Interface Refactoring
- [x] 1.1 Remove `Add(key string, value []byte) error` from `KvStore` interface.
- [x] 1.2 Add `DeleteByPrefix(ctx context.Context, prefix string) error` to `KvStore` interface.
- [x] 1.3 Add `context.Context` parameter to all `KvStore` methods.
- [x] 1.4 Define standard error variables (e.g., `ErrKeyNotFound`) in `kv_store.go`.

## 2. Configuration & Types
- [x] 2.1 Update `Provider` struct in `gateway/traefik/types.go` to include `PoolSize`, `MaxIdleConns`, `ConnectTimeout`, and `ReadTimeout`.

## 3. Redis Implementation
- [x] 3.1 Implement `DeleteByPrefix` using `Scan` and `Del` in `redis.go`.
- [x] 3.2 Refactor `GetByPrefix` to use `MGET` in `redis.go`.
- [x] 3.3 Map Redis-specific errors to standard `KvStore` errors.
- [x] 3.4 Configure connection pool and timeouts using updated `Provider` settings.

## 4. Consul Implementation
- [x] 4.1 Implement `DeleteByPrefix` using `DeleteTree` in `consul.go`.
- [x] 4.2 Remove `Token` support and update `NewConsulStore` to use Basic Auth exclusively.
- [x] 4.3 Map Consul-specific errors to standard `KvStore` errors.
- [x] 4.4 Configure timeouts using updated `Provider` settings.

## 5. Etcd Implementation
- [x] 5.1 Implement `KvStore` interface in `etcd.go` using `clientv3`.
- [x] 5.2 Implement `DeleteByPrefix` and `GetByPrefix` using Etcd prefix operations.
- [x] 5.3 Implement connection pooling and timeout logic for Etcd.

## 6. Zookeeper Implementation
- [x] 6.1 Implement `KvStore` interface in `zookeeper.go` using `go-zookeeper`.
- [x] 6.2 Implement recursive `DeleteByPrefix` and `GetByPrefix` for ZNode tree.
- [x] 6.3 Implement connection management and retry logic for Zookeeper.

## 7. Verification
- [x] 7.1 Update `mockKvStore` in `traefik_test.go` to match new interface.
- [ ] 7.2 Add unit tests for error mapping across all backends.
- [ ] 7.3 Verify connection pool behavior under load (if possible).
- [ ] 7.4 Verify overall system stability with all backends.
