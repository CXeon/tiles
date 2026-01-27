# Design: KV Store Implementation Optimization

## Context
The `KvStore` is a critical component for Traefik configuration management. High-performance operations and atomicity are important for maintaining gateway consistency.

## Decisions

### 1. DeleteByPrefix Interface
Adding `DeleteByPrefix` allows backends to use their native batch deletion capabilities (e.g., Consul's `DeleteTree`, Etcd's prefix range delete).

### 2. Redis Optimization
- **DeleteByPrefix**: Instead of a Lua script, we will use the standard Go client's `Scan` iterator to find keys and `Del` to remove them in batches. This avoids blocking the Redis server with `KEYS` and is easier to maintain.
- **GetByPrefix**: Switch from:
  ```go
  for _, key := range keys { val, _ := client.Get(key); result[key] = val }
  ```
  to:
  ```go
  values, _ := client.MGet(keys)
  for i, key := range keys { result[key] = values[i] }
  ```

### 3. Consul Optimization
- **DeleteByPrefix**: Use `s.client.KV().DeleteTree(prefix, nil)` which is the native way to delete by prefix.
- **Authentication**: Simplify by removing `Token` support and relying solely on `Username` and `Password` for Basic Auth, as requested for internal consistency.

### 4. Etcd Implementation
- **Driver**: `go.etcd.io/etcd/client/v3`.
- **DeleteByPrefix**: Use `client.Delete(ctx, prefix, clientv3.WithPrefix())`.
- **GetByPrefix**: Use `client.Get(ctx, prefix, clientv3.WithPrefix())`.
- **Concurrency**: Etcd's native support for prefixes makes it highly efficient for Traefik's KV model.

### 5. Zookeeper Implementation
- **Driver**: `github.com/go-zookeeper/zk`.
- **DeleteByPrefix**: Since Zookeeper doesn't have a native prefix-delete, we will implement a recursive deletion of child nodes.
- **GetByPrefix**: Implement recursive traversal of the ZNode tree starting from the prefix path.
- **Note**: Ensure path normalization (trailing slashes) for ZNode paths.

### 6. Interface Clean-up
- **Remove Add**: The `Add` method is redundant for Traefik's KV use case and will be removed.
- **Context Injection**: All methods in `KvStore` interface will now accept `context.Context` to allow for proper timeout and cancellation propagation.
- **Error Standardization**: Define `ErrKeyNotFound` and other common errors in `kv_store.go`. Backend implementations will wrap or return these standard errors.

### 7. Production Readiness
- **Connection Pooling**: Update `Provider` struct to include `PoolSize`, `MinIdleConns`, etc.
- **Timeouts**: Add `ConnectTimeout` and `ReadWriteTimeout` to the configuration.
- **Health Checks**: Implement periodic ping/health check logic within the store wrappers if not natively handled by the underlying driver.

## Risks / Trade-offs
- **Redis Batch Performance**: Using `SCAN` + `DEL` might be slightly slower than a single Lua script for very large datasets, but it is much safer for the Redis instance's stability.
- **Consul Auth**: Removing `Token` might affect environments that strictly use ACL tokens without Basic Auth.
