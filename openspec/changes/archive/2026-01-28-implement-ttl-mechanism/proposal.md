# Proposal: Implement TTL Mechanism for Traefik Gateway Client

## Why

当前 Traefik 网关客户端存在严重的配置残留问题：

1. **服务崩溃数据残留**：当服务实例异常崩溃（进程被 kill、宿主机断电、网络中断等）无法调用 `Deregister` 方法时，注册到 KV Store 的配置会永久残留，导致 Traefik 继续将流量路由到已崩溃的服务实例，造成请求失败。

2. **健康检查无法清理配置**：虽然 Traefik 的健康检查机制可以标记不健康的实例并跳过路由，但无法删除 KV Store 中的僵尸配置，导致存储空间持续膨胀。

3. **依赖手动清理**：目前只能通过显式调用 `Deregister` 或手动删除 KV Store 数据来清理配置，在大规模微服务环境中运维成本高且容易遗漏。

4. **缺少业界最佳实践**：主流服务注册中心（Consul、Etcd、Eureka）均支持 TTL 机制实现自动清理，当前实现缺少这一关键能力。

## What Changes

实现基于 TTL（Time To Live）的自动清理机制，通过以下改进解决配置残留问题：

### KV Store 层增强
- 修改 `Put` 方法：增加可选参数 `expired`（单位：秒），支持设置键值对的过期时间
- 新增 `KeepAlive` 方法：支持刷新已存在键的 TTL（用于心跳续期）
- Redis、Consul、Etcd、ZooKeeper 实现统一支持 TTL 语义（通过各自的机制：Redis 原生 TTL、Consul Session、Etcd Lease、ZooKeeper 临时节点）

### Traefik Client 增强
- 新增 `KeepAlive` 方法：手动刷新服务实例配置的 TTL（供高级用户使用）
- 修改 `Register` 方法：根据 `Endpoint.TTL` 字段自动使用 TTL 机制注册实例配置，并默认启动自动续约协程
- 新增 `WithAutoRenew` 选项：允许用户禁用自动续约，完全控制心跳逻辑
- 新增 Handler 层 `Refresh` 实现：定位并刷新当前实例的 URL 和 Weight 配置
- 修改 `Deregister` 方法：自动停止续约协程
- 修改 `Close` 方法：停止所有续约协程，防止泄漏

### 配置生命周期分离
- **Service 实例配置**（如 URL、Weight）：使用 TTL，服务崩溃后自动过期清理
- **Router 配置**（如 Rule、Middleware）：不使用 TTL，因为多实例共享，应保持稳定
- **Service 全局配置**（如 HealthCheck）：不使用 TTL，全局共享配置

### 向后兼容
- `Endpoint.TTL = 0` 时退化为无 TTL 模式，保持现有行为
- 不影响未设置 TTL 的现有服务
- `WithAutoRenew(false)` 可禁用自动续约，允许用户自行管理心跳
- 其他网关客户端实现者可选择不实现自动续约

## Success Metrics

1. **自动清理有效性**：服务崩溃后在 TTL 时间内自动清理配置，无需手动干预
2. **心跳机制稳定性**：自动续约成功率 > 99.9%，`KeepAlive` 方法延迟 < 10ms
3. **性能影响**：心跳刷新操作延迟 < 10ms（P99）
4. **向后兼容性**：现有服务无需修改代码即可正常运行
5. **测试覆盖率**：TTL 相关代码测试覆盖率 > 90%
6. **写入性能优化**：
   - Handler 对 Router、HealthCheck 等公共配置使用直接 Put 策略，无冗余 Get 检查
   - 服务注册性能提升约 50%（网络往返次数减半）

## Risks and Mitigations

### 风险 1：KV Store 不支持 TTL
- **概率**：中
- **影响**：高
- **缓解措施**：
  - Redis 和 Consul 均原生支持 TTL
  - 提供降级方案：TTL = 0 时使用原有逻辑
  - 文档明确说明 KV Store 支持情况

### 风险 2：心跳失败导致配置过期
- **概率**：低
- **影响**：高
- **缓解措施**：
  - 心跳间隔设置为 TTL/3，允许 2 次失败
  - `Refresh` 失败自动重新 `Register`
  - 建议 TTL 设置为 30s 以上，给予充足容错时间

### 风险 3：时钟不同步
- **概率**：极低
- **影响**：中
- **缓解措施**：
  - TTL 由 KV Store 服务端控制，与客户端时钟无关
  - 文档建议使用 NTP 同步时间

## Dependencies

- **KV Store 接口扩展**：需要修改 `Put` 方法签名添加可选 `expired` 参数，新增 `KeepAlive` 方法
- **Redis/Consul/Etcd/ZooKeeper 实现**：需要各 KV Store 实现支持 TTL 语义
- **Traefik Gateway 规范更新**：需要更新 spec 定义 TTL 行为和自动续约机制
- **Gateway 接口扩展**：需要在 `gateway.Client` 接口添加 `KeepAlive` 方法

## Open Questions

1. **默认 TTL 值**：是否需要提供默认 TTL 值（如 30s）？
   - 建议：不提供默认值，强制用户显式设置，避免误用
   
2. **自动续约失败重试策略**：续约失败后是否需要退避重试？
   - 已确定：失败后立即尝试重新 `Register`，如果仍失败则记录日志，继续下一次心跳
   
5. **自动续约 vs 手动续约**：是否应该默认启用自动续约？
   - 已确定：默认启用自动续约（`autoRenew = true`），用户可通过 `WithAutoRenew(false)` 禁用

3. **Router 配置清理**：当所有实例 TTL 过期后，Router 配置是否需要自动清理？
   - 建议：不自动清理，保持 Router 配置稳定，避免频繁创建/删除

4. **多实例竞争**：多个实例同时刷新 TTL 时是否有竞争问题？
   - 答案：无竞争，每个实例只刷新自己的配置（通过索引隔离）
