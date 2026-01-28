# Design: TTL Mechanism for Traefik Gateway Client

## Overview

本设计实现基于 TTL（Time To Live）的自动清理机制，解决服务实例崩溃后配置残留问题。核心思路是：
- **Service 实例配置**使用 TTL，服务崩溃后自动过期
- **Router 配置**不使用 TTL，多实例共享，保持稳定
- **自动续约机制**保持服务存活期间配置不过期（默认启用，可禁用）
- **通用化设计**支持 Redis、Consul、Etcd、ZooKeeper 等多种 KV Store

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      User Application                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  方案 A：默认自动续约（推荐）                       │   │
│  │  1. client.Register(endpoint) [TTL=30s]             │   │
│  │  2. 后台自动续约，用户无感知                      │   │
│  │  3. defer client.Deregister(endpoint)                │   │
│  │                                                         │   │
│  │  方案 B：禁用自动续约（高级用户）                 │   │
│  │  1. client = NewClient(WithAutoRenew(false))        │   │
│  │  2. client.Register(endpoint)                       │   │
│  │  3. go heartbeat(client.KeepAlive)  // 用户自己管理  │   │
│  │  4. defer client.Deregister(endpoint)                │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    Traefik Gateway Client                    │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Register(endpoint)                                   │   │
│  │  ├─> Router 配置 (无 TTL)                            │   │
│  │  ├─> Service 实例配置 (带 TTL)                       │   │
│  │  └─> 自动启动续约协程 (如果 autoRenew=true)        │   │
│  │                                                         │   │
│  │  KeepAlive(endpoint)  [新增 - 手动续约]            │   │
│  │  └─> 刷新 Service 实例配置 TTL                       │   │
│  │                                                         │   │
│  │  Deregister(endpoint)                                 │   │
│  │  ├─> 停止续约协程                                  │   │
│  │  └─> 立即删除配置                                    │   │
│  │                                                         │   │
│  │  Close()                                              │   │
│  │  └─> 停止所有续约协程，防止泄漏                   │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                       KV Store Layer                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Put(key, value, expired...)       [修改]          │   │
│  │  KeepAlive(key, ttl...)            [新增]          │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    KV Store Backends                         │
│  ┌──────────────┐              ┌──────────────┐            │
│  │    Redis     │              │   Consul     │            │
│  │  SET k v EX  │              │ Session TTL  │            │
│  │  EXPIRE k    │              │ Session Renew│            │
│  └──────────────┘              └──────────────┘            │
│                                                               │
│  ┌──────────────┐              ┌──────────────┐            │
│  │    Etcd      │              │  ZooKeeper  │            │
│  │ Lease+Auto  │              │ Ephemeral   │            │
│  │  KeepAlive  │              │    Node     │            │
│  └──────────────┘              └──────────────┘            │
└─────────────────────────────────────────────────────────────┘
```

## Data Model

### Configuration Lifecycle

| 配置类型 | 路径示例 | TTL 策略 | 原因 |
|---------|---------|---------|------|
| **Router Rule** | `traefik/http/routers/<name>/rule` | 无 TTL | 多实例共享，稳定 |
| **Router Service** | `traefik/http/routers/<name>/service` | 无 TTL | 多实例共享，稳定 |
| **Router Middlewares** | `traefik/http/routers/<name>/middlewares/0` | 无 TTL | 多实例共享，稳定 |
| **Service Instance URL** | `traefik/http/services/<name>/loadbalancer/servers/0/url` | **带 TTL** | 实例特有，需自动清理 |
| **Service Instance Weight** | `traefik/http/services/<name>/loadbalancer/servers/0/weight` | **带 TTL** | 实例特有，需自动清理 |
| **Service HealthCheck** | `traefik/http/services/<name>/loadbalancer/healthcheck/path` | 无 TTL | 全局配置，稳定 |

### TTL Calculation

```
推荐配置：
- TTL: 30 秒
- 心跳间隔: TTL / 3 = 10 秒
- 容错次数: 2 次（允许 2 次心跳失败）

时间线示例：
T=0s    Register (TTL=30s)
T=10s   Refresh (TTL=30s)  // 第 1 次心跳
T=20s   Refresh (TTL=30s)  // 第 2 次心跳
T=30s   Refresh (TTL=30s)  // 第 3 次心跳
...
T=25s   服务崩溃（假设）
T=55s   配置自动过期清理（30s TTL）
```

## API Design

### KV Store Interface

```go
// 修改后的接口
type KvStore interface {
    // 现有方法
    Get(ctx context.Context, key string) ([]byte, error)
    GetByPrefix(ctx context.Context, prefix string) (map[string][]byte, error)
    Delete(ctx context.Context, key string) error
    DeleteByPrefix(ctx context.Context, prefix string) error
    Close() error
    
    // 修改：Put 方法增加可选 expired 参数（单位：秒）
    // expired: 可选参数，过期时间（单位：秒）。0 表示永不过期。
    Put(ctx context.Context, key string, value []byte, expired ...uint32) error
    
    // 新增：KeepAlive 续期单个 key 的生命周期
    // ttl: 可选参数，续期的 TTL 值（单位：秒）。如果不传，使用默认 TTL。
    KeepAlive(ctx context.Context, key string, ttl ...uint32) error
    
    // 新增：BatchKeepAlive 批量续期多个 key，确保它们的 TTL 同步
    // 重要：解决 Redis 单个 key 续期导致的 TTL 时间漂移问题
    // 对于 Redis：使用 Pipeline 批量 EXPIRE，确保所有 key 的过期时间一致
    // 对于 Consul/Etcd/ZooKeeper：内部调用一次 Session/Lease 续约即可
    BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error
}
```

### Traefik Client Interface

```go
// 修改后的接口
type Client interface {
    // 现有方法
    Register(ctx context.Context, endpoint *gateway.Endpoint) error
    Update(ctx context.Context, endpoint *gateway.Endpoint) error
    Deregister(ctx context.Context, endpoint *gateway.Endpoint) error
    Close() error
    
    // 新增：手动续约 TTL（供高级用户使用）
    KeepAlive(ctx context.Context, endpoint *gateway.Endpoint) error
}

// 新增：客户端配置选项
type ClientOption func(*traefikClient)

// WithAutoRenew 配置是否自动续约（默认 true）
func WithAutoRenew(enabled bool) ClientOption {
    return func(c *traefikClient) {
        c.autoRenew = enabled
    }
}
```

### Endpoint 结构体增强

```go
type Endpoint struct {
    // ... 其他字段 ...
    
    // TTL 生存时间（单位：秒）
    // 0 表示永不过期，> 0 表示 N 秒后自动删除实例配置
    TTL uint32
    
    // ... 其他字段 ...
}
```

## Implementation Details

### 0. Handler 写入策略原则

**核心原则：公共配置直接覆盖，实例配置检查索引**

#### 规则一：公共配置（Router、HealthCheck）- 直接 Put，无需检查

**理由**：
1. **幂等性**：同一服务的配置通常不变，重复写入结果相同
2. **性能优化**：避免额外的 `Get` 操作，减少 50% 网络往返
3. **代码简洁**：减少分支逻辑和错误处理路径

**适用范围**：
- Router Rule、Router Service、Router Middlewares、Router Entrypoints
- Router Priority、HealthCheck Path 等全局配置

**实现示例**：
```go
// ✅ 正确：直接 Put
ruleKey := constructor.GenRouterRuleKey(endpoint)
err := h.store.Put(h.ctx, ruleKey, []byte(rule))

// ❌ 错误：先检查再写入（冗余）
exists, _ := h.store.Get(h.ctx, ruleKey)
if exists == nil {
    h.store.Put(h.ctx, ruleKey, []byte(rule))
}
```

#### 规则二：实例配置（Service URL/Weight）- 查找索引后写入

**理由**：
1. **索引管理**：需要确定实例在 loadbalancer 中的序号（0, 1, 2...）
2. **避免重复**：同一个 IP:Port 不应该注册两次
3. **支持更新**：已存在的实例可以更新 Weight 等属性

**实现示例**：
```go
// ✅ 正确：查找索引，避免重复
loadbalancerServerMap, err := h.store.GetByPrefix(h.ctx, loadbalancerServiceKeyPrefix)
serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)

instanceIndex := findInstanceIndex(loadbalancerServerMap, serverURL)
if instanceIndex < 0 {
    // 新实例，使用下一个可用索引
    instanceIndex = maxIndex + 1
    h.store.Put(h.ctx, serviceURLKey, []byte(serverURL), endpoint.TTL)
}
```

#### 性能对比

假设注册一个服务（1 Router Rule + 1 Router Service + 2 Middlewares + 1 HealthCheck）：

| 方式 | 操作次数 | 网络往返 |
|------|---------|----------|
| **当前（检查后写入）** | 5 Get + 5 Put | 10 次 |
| **优化（直接写入）** | 0 Get + 5 Put | 5 次 |
| **性能提升** | - | **50%** |

### 1. Register 方法修改（带自动续约）

```go
func (h *handler) Register(endpoint gateway.Endpoint, opts ...HandlerOptions) error {
    // 1. Router 配置（直接 Put，无需检查）
    err := h.upsertRouter(endpoint, "", rule, middlewares, 0)
    
    // 2. HealthCheck 配置（直接 Put，无需检查）
    if opt.HealthCheckPath != "" {
        healthCheckKey := constructor.GenServiceHealthCheckPathKey(endpoint)
        err = h.store.Put(h.ctx, healthCheckKey, []byte(opt.HealthCheckPath))
    }
    
    // 3. Service 实例注册（需要查找索引，避免重复）
    loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
    loadbalancerServerMap, err := h.store.GetByPrefix(h.ctx, loadbalancerServiceKeyPrefix)
    
    serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
    instanceIndex := findInstanceIndex(loadbalancerServerMap, serverURL)
    
    if instanceIndex < 0 {
        // 新实例，使用下一个可用索引
        instanceIndex = maxIndex + 1
        serviceURLKey := constructor.GenServiceUrlKey(instanceIndex, endpoint)
        
        // 关键：使用可选参数传递 TTL
        if endpoint.TTL > 0 {
            err = h.store.Put(h.ctx, serviceURLKey, 
                []byte(serverURL), 
                endpoint.TTL)  // 传递 TTL 参数
        } else {
            err = h.store.Put(h.ctx, serviceURLKey, []byte(serverURL))
        }
        
        // Weight 也使用相同策略
        if endpoint.Weight > 0 {
            weightKey := constructor.GenServiceWeightKey(currentMaxServicesURLIndex, endpoint)
            if endpoint.TTL > 0 {
                err = h.store.Put(h.ctx, weightKey, 
                    []byte(strconv.Itoa(int(endpoint.Weight))), endpoint.TTL)
            } else {
                err = h.store.Put(h.ctx, weightKey, 
                    []byte(strconv.Itoa(int(endpoint.Weight))))
            }
        }
    }
    
    return nil
}
```

### 2. 自动续约机制实现

```go
type traefikClient struct {
    handler          *handler
    autoRenew        bool                    // 是否自动续约
    renewGoroutines  map[string]chan struct{} // endpoint ID -> stop channel
    renewMutex       sync.RWMutex
    // ... 其他字段
}

func (c *traefikClient) Register(ctx context.Context, endpoint *gateway.Endpoint) error {
    normalized, err := normalizeEndpoint(endpoint)
    if err != nil {
        return err
    }
    
    opts := c.buildHandlerOptions()
    if err := c.handler.Register(normalized, opts); err != nil {
        return err
    }
    
    // 如果启用自动续约且 TTL > 0，启动续约协程
    if c.autoRenew && normalized.TTL > 0 {
        c.startAutoRenew(ctx, normalized)
    }
    
    return nil
}

func (c *traefikClient) startAutoRenew(ctx context.Context, endpoint gateway.Endpoint) {
    stopChan := make(chan struct{})
    
    c.renewMutex.Lock()
    c.renewGoroutines[endpoint.ID()] = stopChan
    c.renewMutex.Unlock()
    
    go func() {
        interval := time.Duration(endpoint.TTL) * time.Second / 3 // TTL/3
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                if err := c.handler.Refresh(endpoint); err != nil {
                    // 续约失败，尝试重新注册
                    log.Printf("[Traefik] KeepAlive failed for %s, re-registering: %v", endpoint.ID(), err)
                    if err := c.handler.Register(endpoint, c.buildHandlerOptions()); err != nil {
                        log.Printf("[Traefik] Re-register failed for %s: %v", endpoint.ID(), err)
                    }
                }
            case <-stopChan:
                return
            case <-ctx.Done():
                return
            }
        }
    }()
}

func (c *traefikClient) Deregister(ctx context.Context, endpoint *gateway.Endpoint) error {
    normalized, err := normalizeEndpoint(endpoint)
    if err != nil {
        return err
    }
    
    // 停止续约协程
    c.stopAutoRenew(normalized.ID())
    
    return c.handler.Deregister(normalized)
}

func (c *traefikClient) stopAutoRenew(endpointID string) {
    c.renewMutex.Lock()
    defer c.renewMutex.Unlock()
    
    if stopChan, exists := c.renewGoroutines[endpointID]; exists {
        close(stopChan)
        delete(c.renewGoroutines, endpointID)
    }
}

func (c *traefikClient) Close() error {
    // 停止所有续约协程
    c.renewMutex.Lock()
    for _, stopChan := range c.renewGoroutines {
        close(stopChan)
    }
    c.renewGoroutines = make(map[string]chan struct{})
    c.renewMutex.Unlock()
    
    return c.handler.Close()
}
```

### 3. Handler Refresh 方法实现（使用 BatchKeepAlive）

```go
func (h *handler) Refresh(endpoint gateway.Endpoint) error {
    if endpoint.TTL == 0 {
        return nil // 无 TTL，无需刷新
    }
    
    constructor := NewConstructor()
    
    // 1. 查找当前实例索引
    loadbalancerServiceKeyPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
    loadbalancerServerMap, err := h.store.GetByPrefix(h.ctx, loadbalancerServiceKeyPrefix)
    if err != nil {
        return fmt.Errorf("failed to find instance: %w", err)
    }
    
    serverURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
    instanceIndex := findInstanceIndex(loadbalancerServerMap, loadbalancerServiceKeyPrefix, serverURL)
    
    if instanceIndex < 0 {
        return fmt.Errorf("instance not found, need re-register")
    }
    
    // 2. 收集需要续期的所有 key
    keys := []string{
        constructor.GenServiceUrlKey(instanceIndex, endpoint),
    }
    
    if endpoint.Weight > 0 {
        keys = append(keys, constructor.GenServiceWeightKey(instanceIndex, endpoint))
    }
    
    // 3. 批量续期，确保所有 key 的 TTL 同步
    // 重要：这解决了 Redis 单个 key 续期导致的时间漂移问题
    return h.store.BatchKeepAlive(h.ctx, keys, endpoint.TTL)
}
```

### 4. Redis 实现

```go
func (r *redisStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
    if len(expired) > 0 && expired[0] > 0 {
        ttl := time.Duration(expired[0]) * time.Second
        return r.client.Set(ctx, key, value, ttl).Err()
    }
    return r.client.Set(ctx, key, value, 0).Err() // 0 表示永不过期
}

func (r *redisStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
    var duration time.Duration
    if len(ttl) > 0 && ttl[0] > 0 {
        duration = time.Duration(ttl[0]) * time.Second
    } else {
        // 默认 TTL（如 30 秒）
        duration = 30 * time.Second
    }
    return r.client.Expire(ctx, key, duration).Err()
}

func (r *redisStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
    if len(keys) == 0 {
        return nil
    }
    
    var duration time.Duration
    if len(ttl) > 0 && ttl[0] > 0 {
        duration = time.Duration(ttl[0]) * time.Second
    } else {
        duration = 30 * time.Second
    }
    
    // 使用 Pipeline 批量操作，确保原子性和时间一致性
    // 所有 EXPIRE 命令在极短时间内执行，几乎同时设置过期时间
    pipe := r.client.Pipeline()
    for _, key := range keys {
        pipe.Expire(ctx, key, duration)
    }
    _, err := pipe.Exec(ctx)
    return err
}
```

### 5. Consul 实现

```go
type consulStore struct {
    client    *api.Client
    sessionID string  // 全局共享的 Session ID
    sessionMutex sync.RWMutex
}

// ensureSession 确保 Session 存在
func (c *consulStore) ensureSession(ttl uint32) (string, error) {
    c.sessionMutex.Lock()
    defer c.sessionMutex.Unlock()
    
    // 如果已存在 Session，直接返回
    if c.sessionID != "" {
        return c.sessionID, nil
    }
    
    // 创建全局共享的 Session
    sessionOpts := &api.SessionEntry{
        Behavior: "delete",  // Session 过期时删除 key
        TTL:      fmt.Sprintf("%ds", ttl),
    }
    sessionID, _, err := c.client.Session().Create(sessionOpts, nil)
    if err != nil {
        return "", err
    }
    
    c.sessionID = sessionID
    return sessionID, nil
}

func (c *consulStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
    if len(expired) > 0 && expired[0] > 0 {
        // 使用 Session 实现 TTL
        sessionID, err := c.ensureSession(expired[0])
        if err != nil {
            return err
        }
        
        // 关联 Session 到 KV
        p := &api.KVPair{
            Key:     key,
            Value:   value,
            Session: sessionID,
        }
        _, err = c.client.KV().Put(p, nil)
        return err
    }
    
    // 无 TTL，普通 Put
    p := &api.KVPair{
        Key:   key,
        Value: value,
    }
    _, err := c.client.KV().Put(p, nil)
    return err
}

func (c *consulStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
    // Consul 通过 Session Renew 实现 TTL 刷新
    c.sessionMutex.RLock()
    sessionID := c.sessionID
    c.sessionMutex.RUnlock()
    
    if sessionID == "" {
        return fmt.Errorf("no session found, need re-register")
    }
    
    _, _, err := c.client.Session().Renew(sessionID, nil)
    return err
}

func (c *consulStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
    // Consul 全局共享 Session，所有 key 关联同一个 Session
    // 只需要 Renew 一次 Session，所有 key 同时续期
    return c.KeepAlive(ctx, "", ttl...)
}
```

### 6. Etcd 实现（基于 Lease 机制）

```go
type etcdStore struct {
    client      *clientv3.Client
    leases      map[string]clientv3.LeaseID  // key -> LeaseID 映射
    leaseMutex  sync.RWMutex
}

func (e *etcdStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
    if len(expired) > 0 && expired[0] > 0 {
        // 创建 Lease
        leaseResp, err := e.client.Grant(ctx, int64(expired[0]))
        if err != nil {
            return err
        }
        
        // 存储 LeaseID
        e.leaseMutex.Lock()
        e.leases[key] = leaseResp.ID
        e.leaseMutex.Unlock()
        
        // 关联 Lease 到 key
        _, err = e.client.Put(ctx, key, string(value), clientv3.WithLease(leaseResp.ID))
        if err != nil {
            return err
        }
        
        // 启动自动续约 (重要：Etcd 的 Lease 自动续约)
        _, err = e.client.KeepAlive(ctx, leaseResp.ID)
        return err
    }
    
    // 无 TTL，普通 Put
    _, err := e.client.Put(ctx, key, string(value))
    return err
}

func (e *etcdStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
    // 获取 LeaseID
    e.leaseMutex.RLock()
    leaseID, exists := e.leases[key]
    e.leaseMutex.RUnlock()
    
    if !exists {
        return fmt.Errorf("no lease found for key, need re-register")
    }
    
    // 注意：Etcd 的 KeepAlive 已经在 Put 时启动了自动续约
    // 这里可以为 No-op，或者手动刷新一次
    _, err := e.client.KeepAliveOnce(ctx, leaseID)
    return err
}

func (e *etcdStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
    if len(keys) == 0 {
        return nil
    }
    
    // Etcd 所有 key 关联同一个 Lease
    // 只需要刷新一次 Lease，所有 key 同时续期
    // 使用第一个 key 获取 LeaseID
    return e.KeepAlive(ctx, keys[0], ttl...)
}
```

### 7. ZooKeeper 实现（基于临时节点）

```go
func (z *zkStore) Put(ctx context.Context, key string, value []byte, expired ...uint32) error {
    if len(expired) > 0 && expired[0] > 0 {
        // 使用临时节点
        flags := int32(zk.FlagEphemeral)
        
        // 注意：ZooKeeper 临时节点无法设置精确 TTL，依赖连接存活。
        // expired 参数会被忽略。
        // 当客户端连接断开时，临时节点自动删除。
        _, err := z.conn.Create(key, value, flags, zk.WorldACL(zk.PermAll))
        return err
    }
    
    // 持久节点
    flags := int32(0)
    _, err := z.conn.Create(key, value, flags, zk.WorldACL(zk.PermAll))
    return err
}

func (z *zkStore) KeepAlive(ctx context.Context, key string, ttl ...uint32) error {
    // 注意：ZooKeeper 临时节点自动维护，无需手动续约。
    // 这里为 No-op，只要连接存活即可。
    // 可以检查节点是否存在作为健康检查。
    exists, _, err := z.conn.Exists(key)
    if err != nil {
        return err
    }
    if !exists {
        return fmt.Errorf("key not found, need re-register")
    }
    return nil
}

func (z *zkStore) BatchKeepAlive(ctx context.Context, keys []string, ttl ...uint32) error {
    // ZooKeeper 临时节点自动维护，无需手动续约
    // 只要连接存活，所有临时节点就存活
    // 这里为 No-op
    return nil
}
```

## Error Handling

### 自动续约失败处理策略

自动续约已在 `startAutoRenew` 中实现，失败后自动重新注册。

### 用户手动续约示例（禁用自动续约）

```go
// 用户侧实现示例
func heartbeat(client gateway.Client, endpoint *gateway.Endpoint) {
    ticker := time.NewTicker(time.Duration(endpoint.TTL) * time.Second / 3)
    defer ticker.Stop()
    
    for range ticker.C {
        if err := client.KeepAlive(ctx, endpoint); err != nil {
            log.Printf("KeepAlive failed: %v, re-registering...", err)
            
            // 策略：KeepAlive 失败，尝试重新 Register
            if err := client.Register(ctx, endpoint); err != nil {
                log.Printf("Re-register failed: %v", err)
                // 记录监控指标，由运维系统决定是否重启服务
            }
        }
    }
}
```

## Testing Strategy

### Unit Tests
- `TestKvStore_PutWithTTL`: 验证 Put 方法的 expired 参数生效
- `TestKvStore_KeepAlive`: 验证 KeepAlive 续约正确
- `TestHandler_RegisterWithTTL`: 验证 Register 使用 TTL
- `TestHandler_Refresh`: 验证 Refresh 定位并刷新正确的实例
- `TestAutoRenew`: 验证自动续约机制
- `TestAutoRenewDisabled`: 验证 WithAutoRenew(false) 生效

### Integration Tests
- `TestTTLExpiration`: 验证配置在 TTL 后自动清理
- `TestHeartbeatKeepAlive`: 验证自动续约保持配置不过期
- `TestServiceCrashCleanup`: 验证服务崩溃后配置自动清理
- `TestMultiBackendTTL`: 验证 Redis/Consul/Etcd/ZooKeeper 各后端 TTL 实现

### Backward Compatibility Tests
- `TestRegisterWithoutTTL`: 验证 TTL=0 时行为不变
- `TestExistingServices`: 验证现有服务不受影响

## Rollout Plan

### Phase 1: KV Store 接口扩展
1. 修改 `Put` 方法签名，增加可选 `expired` 参数
2. 新增 `KeepAlive` 方法
3. Redis/Consul/Etcd/ZooKeeper 实现
4. 单元测试

### Phase 2: Traefik Client 增强
1. 修改 `Register` 支持 TTL 和自动续约
2. 新增 `KeepAlive` 方法
3. 新增 `WithAutoRenew` 配置选项
4. Handler 层 `Refresh` 实现
5. 单元测试

### Phase 3: 集成测试
1. TTL 过期清理测试
2. 自动续约测试
3. 多后端 TTL 测试
4. 向后兼容性测试

### Phase 4: 文档和示例
1. API 文档
2. 用户指南（如何使用 TTL 和自动续约）
3. 最佳实践（TTL 值推荐、续约策略）

## Performance Considerations

- **续约开销**：每次 KeepAlive 只刷新 2 个 key（URL + Weight），网络开销 < 1ms
- **自动续约频率**：推荐 TTL/3，对于 TTL=30s，续约频率为 10s，负载可接受
- **协程开销**：每个实例一个续约协程，内存开销极小
- **批量刷新**：如果有多个实例，可以批量刷新（未来优化）
- **Etcd 自动续约**：Etcd Lease 自带自动续约，无额外开销
