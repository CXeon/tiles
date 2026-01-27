# Tasks: Optimize Traefik Client

## Phase 1: Constructor 层增强 (2 tasks)

- [x] 1.1 在 `constructor.go` 中新增 `GenRouterPrefixAll(endpoint)` 方法
  - 返回所有 router 的共同前缀（用于删除 protected 和 public router）
  - 单元测试验证前缀生成的正确性

- [x] 1.2 在 `constructor.go` 中新增 `GenServiceInstancePrefix(index, endpoint)` 方法
  - 返回单个服务实例的前缀（用于删除特定实例）
  - 导出现有的 `genDefaultServicePrefix` 为 `GenServicePrefix`

## Phase 2: Client 层重构 (4 tasks)

- [x] 2.1 在 `traefik.go` 中提取 `normalizeEndpoint` 函数
  - 验证 endpoint 是否为 nil
  - 验证 Protocol 字段
  - 执行 HTTPS → HTTP 协议转换
  - 编写单元测试

- [x] 2.2 在 `traefik.go` 中提取 `buildHandlerOptions` 方法
  - 直接使用 `c.excludeAuthPaths`（移除路径拼接逻辑）
  - 构造 HandlerOptions
  - 更新 `Register` 和 `Update` 方法使用此函数

- [x] 2.3 重构 `Register` 方法使用公共函数
  - 调用 `normalizeEndpoint`
  - 调用 `buildHandlerOptions`
  - 删除重复代码（85-96 行简化为 3-4 行）

- [x] 2.4 重构 `Update` 方法使用公共函数
  - 调用 `normalizeEndpoint`
  - 调用 `buildHandlerOptions`
  - 删除重复代码（115-143 行简化为 3-4 行）

## Phase 3: Handler 层批量操作 (5 tasks)

- [x] 3.1 重构 `handler.Update` 实现先删后增逻辑
  - 使用 `DeleteByPrefix` 删除所有 router 配置
  - 查找当前实例的索引
  - 使用 `DeleteByPrefix` 删除当前实例的 service 配置
  - 调用 `Register` 重新写入新配置
  - 验证配置不残留的场景（中间件减少、权重变化等）

- [x] 3.2 重构 `handler.Deregister` 实现完全清除
  - 使用 `GetByPrefix` 查找所有服务实例
  - 查找并删除当前实例（使用 `DeleteByPrefix`）
  - 判断是否为最后一个实例
  - 如果是最后一个实例，删除所有 router 和 service 配置
  - 验证完全清除的场景

- [x] 3.3 在 `handler.Register` 中增加错误类型判断
  - 对所有 `store.Put` 调用增加错误判断
  - 区分 `ErrConnectionFailed` 和其他错误
  - 包装错误信息

- [x] 3.4 在 `handler.Update` 中增加错误类型判断
  - 对 `DeleteByPrefix` 调用增加错误判断
  - `ErrKeyNotFound` 可以忽略（说明之前没有配置）
  - `ErrConnectionFailed` 立即返回

- [x] 3.5 在 `handler.Deregister` 中增加错误类型判断
  - 对 `GetByPrefix` 和 `DeleteByPrefix` 调用增加错误判断
  - `ErrKeyNotFound` 可以忽略（已被删除）
  - `ErrConnectionFailed` 立即返回

## Phase 4: 单元测试更新 (6 tasks)

- [x] 4.1 更新 `traefik_test.go` 中的 Mock
  - 为 `mockKvStore` 增加 `DeleteByPrefix` 方法
  - 确保所有测试的 Mock 预期正确

- [x] 4.2 新增 `TestNormalizeEndpoint` 测试
  - 测试 nil endpoint
  - 测试无效 Protocol
  - 测试 HTTPS → HTTP 转换
  - 测试正常的 HTTP endpoint

- [x] 4.3 新增 `TestBuildHandlerOptions` 测试
  - 测试使用完整路径的 excludeAuthPaths
  - 测试中间件列表
  - 测试健康检查路径

- [x] 4.4 更新 `TestTraefikClient_ExcludeAuthPaths` 测试
  - 将相对路径改为完整路径
  - 验证不再有路径拼接逻辑
  - 确保 Mock 预期匹配

- [x] 4.5 新增 `TestHandlerUpdate` 测试（handler 层）
  - 测试 Update 调用 DeleteByPrefix
  - 测试 Update 删除后重新 Register
  - 测试配置变更场景（中间件数量变化、权重变化）
  - 测试错误处理

- [x] 4.6 新增 `TestHandlerDeregister` 测试（handler 层）
  - 测试单实例删除场景（完全清除）
  - 测试多实例删除场景（只删除当前实例）
  - 测试 DeleteByPrefix 调用
  - 测试错误处理

## Phase 5: 集成验证 (3 tasks)

- [x] 5.1 运行所有单元测试
  - `cd gateway/traefik && go test -v`
  - 确保所有测试通过

- [ ] 5.2 代码审查
  - 验证代码重复减少 50% 以上
  - 验证批量操作正确使用
  - 验证错误处理完整

- [ ] 5.3 更新文档注释
  - 为 `WithExcludeAuthPaths` 添加完整路径的说明
  - 为 `Update` 方法添加“先删后增”的说明
  - 为新增的 constructor 方法添加注释

## Dependencies

- Phase 2 依赖 Phase 1 完成（需要新的 constructor 方法）
- Phase 3 依赖 Phase 1 完成（需要新的 constructor 方法）
- Phase 4 依赖 Phase 2 和 Phase 3 完成（需要新的实现）
- Phase 5 依赖所有前置阶段完成

## Estimated Effort

- Phase 1: 1 小时
- Phase 2: 2 小时
- Phase 3: 4 小时
- Phase 4: 3 小时
- Phase 5: 1 小时

**Total**: ~11 小时

## Success Criteria

- ✅ 所有单元测试通过
- ✅ `Register` 和 `Update` 代码重复减少 50% 以上
- ✅ Update 操作不会残留旧配置
- ✅ Deregister 操作完全清除所有相关配置
- ✅ 错误处理能够区分连接失败和键不存在
