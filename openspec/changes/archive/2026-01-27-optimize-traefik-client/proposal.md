# Proposal: Optimize Traefik Client

## Problem Statement

当前 Traefik 客户端实现存在以下问题：

1. **代码重复严重**：`Register` 和 `Update` 方法包含大量重复代码（endpoint 验证、协议归一化、路径处理、HandlerOptions 构造等），虽然两个方法的业务语义不同（Register 用于微服务启动时注册，Update 用于运行时属性变更），但实现层面存在重复。

2. **批量操作不足**：Handler 层对 KV Store 进行过多的单次操作，由于 Traefik 的 key 前缀高度一致（如 `traefik/http/routers/[name]/`、`traefik/http/services/[name]/`），应该利用 `DeleteByPrefix` 进行批量删除，避免数据残留。

3. **路径处理不合理**：`excludeAuthPaths` 在客户端层被重新构造（添加前缀），但用户期望传入完整路径（如 `/company/project/service/public`），不应在内部修改。

4. **错误处理不精确**：Handler 操作 KV Store 时未区分连接失败和值不存在的场景，值不存在可能是正常的（需要新增），而连接失败应立即返回错误。

## Goals

1. **消除代码重复**：提取公共逻辑为辅助函数，保持 `Register` 和 `Update` 的语义清晰性
2. **引入批量操作**：在 Update/Deregister 时使用 `DeleteByPrefix` 清除旧配置，再重新写入
3. **简化路径处理**：移除 `excludeAuthPaths` 的路径拼接逻辑，用户传入完整路径
4. **改进错误处理**：区分 `kv_store.ErrConnectionFailed` 和 `kv_store.ErrKeyNotFound`

## Non-Goals

- 不增强参数验证（如必填字段检查）
- 不添加性能优化缓存（如 excludeAuthPaths 缓存）
- 不增加新的配置选项（如自定义优先级、EntryPoints）

## Proposed Solution

### 1. 消除代码重复

在 `traefik.go` 中提取以下公共函数：

- `normalizeEndpoint(*gateway.Endpoint) (gateway.Endpoint, error)`: 验证并归一化 endpoint（协议转换）
- `buildHandlerOptions(traefikClient) HandlerOptions`: 构造 HandlerOptions

### 2. 批量操作优化

#### 2.1 Update 方法改造

当前 `handler.Update` 只是调用 `Register`，无法真正处理"属性变更"场景。改为：

1. 使用 `DeleteByPrefix` 删除旧的 router 和 service 配置
2. 重新调用 `Register` 写入新配置

#### 2.2 Deregister 方法改造

当前 `Deregister` 只删除 service URL，router 配置残留。改为：

1. 使用 `DeleteByPrefix` 删除所有 router 配置（包括 protected 和 public）
2. 检查 service 下是否还有其他实例，如果是最后一个实例，删除整个 service 配置

#### 2.3 Constructor 增强

新增方法以支持批量操作：

- `GenRouterPrefixAll(endpoint)`: 返回所有 router 前缀（用于删除 protected + public）
- `GenServicePrefix(endpoint)`: 返回 service 前缀

### 3. 路径处理简化

- 移除 `traefik.go` 中的路径拼接逻辑（85-91 行和 129-135 行）
- `excludeAuthPaths` 直接传递给 Handler，用户负责传入完整路径

### 4. 错误处理改进

在 `handler.go` 中：

- 对所有 `store.Put/Get/Delete` 调用增加错误类型判断
- 如果是 `kv_store.ErrConnectionFailed`，立即返回
- 如果是 `kv_store.ErrKeyNotFound`，根据业务逻辑决定是否继续（如 Get 不存在时新增）

## Impact Analysis

### Breaking Changes

1. **API 变更**：`WithExcludeAuthPaths` 的语义变更，用户需要传入完整路径而非相对路径
   - **迁移指南**：将 `[]string{"/public", "health"}` 改为 `[]string{"/company/project/service/public", "/company/project/service/health"}`

### Compatibility

- 不影响 `Register` 和 `Deregister` 的外部调用
- `Update` 方法行为变更（从幂等变为先删后增），但符合"属性更新"的语义

### Performance

- **正向影响**：批量删除减少 KV Store 调用次数（从 N 次 Delete 变为 1 次 DeleteByPrefix）
- **负向影响**：Update 方法增加了删除操作，但整体性能提升

## Alternatives Considered

### 1. 保持 Update 和 Register 完全相同

**Rejected**：无法真正处理属性变更场景，如中间件列表变化、权重变化时，旧配置会残留。

### 2. 在 Deregister 时保留 Router 配置

**Rejected**：会导致 Router 指向不存在的 Service，Traefik 会报错。

### 3. 保留路径拼接逻辑

**Rejected**：用户传入相对路径后在内部拼接会导致理解困难，不如让用户直接传入完整路径更清晰。

## Implementation Plan

1. 在 `constructor.go` 中新增批量操作相关方法
2. 在 `traefik.go` 中提取公共函数并简化路径处理
3. 重构 `handler.Update` 实现先删后增逻辑
4. 重构 `handler.Deregister` 实现批量删除
5. 在所有 KV Store 调用处增加错误类型判断
6. 更新单元测试

## Success Criteria

- 所有单元测试通过
- `Register` 和 `Update` 代码重复减少 50% 以上
- Update 操作不会残留旧配置
- Deregister 操作完全清除所有相关配置
