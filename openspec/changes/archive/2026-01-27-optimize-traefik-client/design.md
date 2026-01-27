# Design: Optimize Traefik Client

## Architecture Overview

本次优化聚焦于 Traefik 客户端的三个层次：

1. **Client 层** (`traefik.go`): 消除代码重复，简化路径处理
2. **Handler 层** (`handler.go`): 引入批量操作，改进错误处理
3. **Constructor 层** (`constructor.go`): 增强批量删除能力

## Detailed Design

### 1. Client 层重构 (traefik.go)

#### 1.1 提取公共函数

**`normalizeEndpoint(endpoint *gateway.Endpoint) (gateway.Endpoint, error)`**

职责：验证并归一化 endpoint
- 检查 endpoint 是否为 nil
- 验证 Protocol 字段
- 执行 HTTPS → HTTP 协议转换
- 返回归一化后的 endpoint 副本

```go
// 伪代码示例
func normalizeEndpoint(endpoint *gateway.Endpoint) (gateway.Endpoint, error) {
    if endpoint == nil {
        return gateway.Endpoint{}, fmt.Errorf("endpoint is nil")
    }
    if err := endpoint.Protocol.Validate(); err != nil {
        return gateway.Endpoint{}, err
    }
    
    normalized := *endpoint
    if normalized.Protocol == gateway.ProtocolTypeHttps {
        normalized.Protocol = gateway.ProtocolTypeHttp
    }
    return normalized, nil
}
```

**`buildHandlerOptions(c *traefikClient) HandlerOptions`**

职责：构造 HandlerOptions
- 直接使用 `c.excludeAuthPaths`（不再重新构造）
- 使用 `c.middlewares` 和 `c.healthCheckPath`

```go
// 伪代码示例
func (c *traefikClient) buildHandlerOptions() HandlerOptions {
    return HandlerOptions{
        Middlewares:      c.middlewares,
        ExcludeAuthPaths: c.excludeAuthPaths,
        HealthCheckPath:  c.healthCheckPath,
    }
}
```

#### 1.2 重构 Register/Update/Deregister

**Register**:
```go
func (c *traefikClient) Register(ctx context.Context, endpoint *gateway.Endpoint) error {
    normalized, err := normalizeEndpoint(endpoint)
    if err != nil {
        return err
    }
    opts := c.buildHandlerOptions()
    return c.handler.Register(normalized, opts)
}
```

**Update**:
```go
func (c *traefikClient) Update(ctx context.Context, endpoint *gateway.Endpoint) error {
    normalized, err := normalizeEndpoint(endpoint)
    if err != nil {
        return err
    }
    opts := c.buildHandlerOptions()
    return c.handler.Update(normalized, opts)
}
```

**Deregister**: 保持简单，只需归一化
```go
func (c *traefikClient) Deregister(ctx context.Context, endpoint *gateway.Endpoint) error {
    normalized, err := normalizeEndpoint(endpoint)
    if err != nil {
        return err
    }
    return c.handler.Deregister(normalized)
}
```

### 2. Handler 层重构 (handler.go)

#### 2.1 Update 方法：先删后增

**设计原因**：
- 当前 Update 只是调用 Register，无法处理配置减少的情况（如中间件从 3 个减到 2 个）
- 先删除所有旧配置，再写入新配置，确保配置完全一致

**实现逻辑**：
```go
func (h *handler) Update(endpoint gateway.Endpoint, opts ...HandlerOptions) error {
    constructor := NewConstructor()
    
    // 1. 删除所有 router 配置（protected + public）
    routerPrefix := constructor.GenRouterPrefixAll(endpoint)
    if err := h.store.DeleteByPrefix(h.ctx, routerPrefix); err != nil {
        if !errors.Is(err, kv_store.ErrKeyNotFound) {
            return fmt.Errorf("failed to delete old router config: %w", err)
        }
    }
    
    // 2. 删除 service 配置（但保留其他实例的 URL）
    // 先获取当前实例的索引，只删除对应的 servers/[index]
    // 如果是最后一个实例，再删除 healthcheck 配置
    
    // 3. 调用 Register 重新注册
    return h.Register(endpoint, opts...)
}
```

#### 2.2 Deregister 方法：完全清除

**设计原因**：
- 当前只删除 service URL，router 配置残留
- 需要判断是否为最后一个实例，决定是否删除整个 service

**实现逻辑**：
```go
func (h *handler) Deregister(endpoint gateway.Endpoint, opts ...HandlerOptions) error {
    constructor := NewConstructor()
    
    // 1. 检查 service 下还有多少实例
    loadbalancerPrefix := constructor.GenServiceLoadbalancerServiceKeyPrefix(endpoint)
    serverMap, err := h.store.GetByPrefix(h.ctx, loadbalancerPrefix)
    // 错误处理：区分连接失败和键不存在
    
    // 2. 找到当前实例的索引并删除
    currentURL := fmt.Sprintf("%s://%s:%d", endpoint.Protocol, endpoint.Ip, endpoint.Port)
    instanceIndex := findInstanceIndex(serverMap, currentURL)
    if instanceIndex >= 0 {
        // 删除 servers/[index]/ 下的所有配置
        instancePrefix := constructor.GenServiceInstancePrefix(instanceIndex, endpoint)
        h.store.DeleteByPrefix(h.ctx, instancePrefix)
    }
    
    // 3. 如果是最后一个实例，删除所有配置
    remainingInstances := countRemainingInstances(serverMap, currentURL)
    if remainingInstances == 0 {
        // 删除所有 router 配置
        routerPrefix := constructor.GenRouterPrefixAll(endpoint)
        h.store.DeleteByPrefix(h.ctx, routerPrefix)
        
        // 删除整个 service 配置
        servicePrefix := constructor.GenServicePrefix(endpoint)
        h.store.DeleteByPrefix(h.ctx, servicePrefix)
    }
    
    return nil
}
```

#### 2.3 错误处理改进

在所有 KV Store 调用处增加错误类型判断：

```go
// 示例：Put 操作
if err := h.store.Put(h.ctx, key, value); err != nil {
    if errors.Is(err, kv_store.ErrConnectionFailed) {
        return fmt.Errorf("kv store connection failed: %w", err)
    }
    return fmt.Errorf("failed to put key %s: %w", key, err)
}

// 示例：Get 操作
value, err := h.store.Get(h.ctx, key)
if err != nil {
    if errors.Is(err, kv_store.ErrConnectionFailed) {
        return fmt.Errorf("kv store connection failed: %w", err)
    }
    if errors.Is(err, kv_store.ErrKeyNotFound) {
        // 根据业务逻辑决定是否继续
        // 可能是正常场景，需要新增
    }
    return fmt.Errorf("failed to get key %s: %w", key, err)
}
```

### 3. Constructor 层增强 (constructor.go)

#### 3.1 新增批量操作方法

**`GenRouterPrefixAll(endpoint gateway.Endpoint) string`**

返回所有 router 的共同前缀，用于删除 protected 和 public router：

```go
func (con *constructor) GenRouterPrefixAll(endpoint gateway.Endpoint) string {
    protocol := strings.ToLower(string(endpoint.Protocol))
    if protocol == "https" {
        protocol = "http"
    }
    // 返回到 routers/ 层级，不包含具体的 router name
    // 例如：traefik/http/routers/dev.china.testco.testprj.testsvc.http.blue
    //       traefik/http/routers/dev.china.testco.testprj.testsvc.http.blue.public
    // 共同前缀：traefik/http/routers/dev.china.testco.testprj.testsvc.http.blue
    baseName := fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s", 
        endpoint.Env, endpoint.Cluster, endpoint.Company, 
        endpoint.Project, endpoint.Service, protocol, endpoint.Color)
    return fmt.Sprintf("%s/%s/%s/%s", con.prefix, protocol, "routers", baseName)
}
```

**`GenServicePrefix(endpoint gateway.Endpoint) string`**

返回整个 service 的前缀：

```go
func (con *constructor) GenServicePrefix(endpoint gateway.Endpoint) string {
    return con.genDefaultServicePrefix(&endpoint)
}
```

**`GenServiceInstancePrefix(index int, endpoint gateway.Endpoint) string`**

返回单个实例的前缀（用于删除特定实例）：

```go
func (con *constructor) GenServiceInstancePrefix(index int, endpoint gateway.Endpoint) string {
    return con.GenServiceLoadbalancerServiceKeyPrefix(endpoint) + strconv.Itoa(index) + "/"
}
```

## Data Flow

### Register 流程
```
Client.Register
  ↓ normalizeEndpoint
  ↓ buildHandlerOptions
  ↓
Handler.Register
  ↓ upsertRouter (protected)
  ↓ upsertRouter (public, if needed)
  ↓ 检查并添加 service URL
  ↓ Put: service/loadbalancer/servers/[index]/url
  ↓ Put: service/loadbalancer/servers/[index]/weight
  ↓ Put: service/loadbalancer/healthcheck/path
```

### Update 流程
```
Client.Update
  ↓ normalizeEndpoint
  ↓ buildHandlerOptions
  ↓
Handler.Update
  ↓ DeleteByPrefix: router prefix (删除所有 router)
  ↓ 查找当前实例索引
  ↓ DeleteByPrefix: servers/[index]/ (删除当前实例配置)
  ↓
  ↓ 调用 Handler.Register (重新写入)
```

### Deregister 流程
```
Client.Deregister
  ↓ normalizeEndpoint
  ↓
Handler.Deregister
  ↓ GetByPrefix: servers/ (查找所有实例)
  ↓ 查找当前实例索引
  ↓ DeleteByPrefix: servers/[index]/ (删除当前实例)
  ↓
  ↓ 如果是最后一个实例:
      ↓ DeleteByPrefix: router prefix
      ↓ DeleteByPrefix: service prefix
```

## Error Handling Strategy

### 连接失败 (ErrConnectionFailed)
- 立即返回错误
- 不进行任何重试或补偿

### 键不存在 (ErrKeyNotFound)
- **在 Register 中**：正常场景，继续执行新增操作
- **在 Update 中**：DeleteByPrefix 不存在时可以忽略（说明之前没有配置）
- **在 Deregister 中**：GetByPrefix 不存在时说明已经被删除，可以忽略

### 其他错误
- 包装错误信息，返回给调用方

## Testing Strategy

### 单元测试更新

1. **traefik_test.go**:
   - 更新 Mock 以支持 `DeleteByPrefix` 调用
   - 测试 `normalizeEndpoint` 函数
   - 测试 `buildHandlerOptions` 函数
   - 更新 `TestTraefikClient_ExcludeAuthPaths`：使用完整路径

2. **handler_test.go** (新增):
   - 测试 Update 的先删后增逻辑
   - 测试 Deregister 的完全清除逻辑
   - 测试错误处理（连接失败 vs 键不存在）

## Backward Compatibility

### Breaking Change: excludeAuthPaths

**影响范围**：所有使用 `WithExcludeAuthPaths` 的代码

**迁移示例**：
```go
// 旧代码
client, _ := traefik.NewClient(ctx, provider, 
    traefik.WithExcludeAuthPaths([]string{"/public", "health"}))

// 新代码
client, _ := traefik.NewClient(ctx, provider, 
    traefik.WithExcludeAuthPaths([]string{
        "/company/project/service/public",
        "/company/project/service/health",
    }))
```

### Non-Breaking Changes

- Register 行为保持不变
- Deregister 行为改进（更彻底地清除配置），但不影响调用方
- Update 行为改进（避免配置残留），符合用户预期

## Performance Considerations

### 改进点

1. **批量删除**：从 O(N) 次 Delete 变为 O(1) 次 DeleteByPrefix
2. **减少网络调用**：Update 时不需要逐个检查旧配置

### 权衡点

1. **Update 增加删除操作**：但整体性能提升，且语义更清晰
2. **DeleteByPrefix 可能删除更多数据**：但这正是我们想要的（彻底清除）

## Security Considerations

- 批量删除操作需要确保前缀计算正确，避免误删其他服务的配置
- 错误处理需要明确区分连接失败和业务错误，避免泄露敏感信息
