# Tasks: Implement TTL Mechanism

## Phase 1: KV Store Interface Enhancement (8 tasks)

- [x] 1.1 Modify `Put` method to support optional `expired` parameter
  - **File**: `gateway/traefik/kv_store/kv_store.go`
  - **Acceptance**: Interface compiles with `Put(ctx, key, value, expired ...uint32)` signature
  
- [x] 1.2 Add `KeepAlive` method to `KvStore` interface
  - **File**: `gateway/traefik/kv_store/kv_store.go`
  - **Acceptance**: Interface compiles with `KeepAlive(ctx, key, ttl ...uint32)` signature

- [x] 1.3 Add `BatchKeepAlive` method to `KvStore` interface
  - **File**: `gateway/traefik/kv_store/kv_store.go`
  - **Acceptance**: Interface compiles with `BatchKeepAlive(ctx, keys, ttl ...uint32)` signature

- [x] 1.4 Implement `Put`, `KeepAlive`, and `BatchKeepAlive` for Redis
  - **File**: `gateway/traefik/kv_store/redis.go`
  - **Acceptance**: 
    - Redis SET with EX works correctly
    - EXPIRE works for single key
    - Pipeline batch EXPIRE works for multiple keys with synchronized timestamps

- [x] 1.5 Implement `Put`, `KeepAlive`, and `BatchKeepAlive` for Consul
  - **File**: `gateway/traefik/kv_store/consul.go`
  - **Acceptance**: 
    - Consul Session-based TTL works correctly
    - Global Session shared
    - BatchKeepAlive calls single Session Renew

- [x] 1.6 Implement `Put`, `KeepAlive`, and `BatchKeepAlive` for Etcd
  - **File**: `gateway/traefik/kv_store/etcd.go` (new file)
  - **Acceptance**: 
    - Etcd Lease mechanism works
    - Automatic KeepAlive enabled
    - BatchKeepAlive uses single Lease renewal

- [x] 1.7 Implement `Put`, `KeepAlive`, and `BatchKeepAlive` for ZooKeeper
  - **File**: `gateway/traefik/kv_store/zookeeper.go` (new file)
  - **Acceptance**: 
    - ZooKeeper Ephemeral Node works
    - Code comments explain limitations
    - BatchKeepAlive is no-op

- [ ] 1.8 Add unit tests for KV Store TTL methods
  - **File**: `gateway/traefik/kv_store/*_test.go`
  - **Acceptance**: Tests cover Put with TTL, KeepAlive, BatchKeepAlive success and error cases

## Phase 2: Traefik Client Core Implementation (15 tasks)

- [x] 2.1 Refactor `handler.Register` to follow "direct Put" strategy
  - **File**: `gateway/traefik/handler.go`
  - **Acceptance**: 
    - Remove redundant `Get` checks for Router configs (Rule, Service, Middlewares, Priority)
    - Remove redundant `Get` checks for HealthCheck Path
    - HealthCheck Path uses direct `Put` without checking existence
    - Service instance registration still uses `GetByPrefix` to find index
    - Performance improved by ~50% (fewer network round trips)

- [x] 2.2 Fix HealthCheck Path write timing
  - **File**: `gateway/traefik/handler.go`
  - **Acceptance**: 
    - Move HealthCheck Path write to top-level (same as Router)
    - No longer inside `if !serverURLExists` block
    - HealthCheck Path always written on every `Register` call

- [x] 2.3 Modify `handler.Register` to use TTL for Service instance configs
  - **File**: `gateway/traefik/handler.go`
  - **Acceptance**: 
    - `endpoint.TTL > 0` passes TTL to `Put` for URL and Weight
    - `endpoint.TTL = 0` uses `Put` without TTL parameter
    - Router configs remain without TTL

- [x] 2.4 Implement `handler.Refresh` method
  - **File**: `gateway/traefik/handler.go`
  - **Acceptance**:
    - Locates instance by IP:Port
    - Collects all keys that need renewal (URL + Weight)
    - Calls `BatchKeepAlive` for synchronized TTL renewal
    - Returns error if instance not found

- [x] 2.5 Add `autoRenew` field and `renewGoroutines` map to `traefikClient`
  - **File**: `gateway/traefik/traefik.go`
  - **Acceptance**: Struct compiles with new fields

- [x] 2.6 Add `WithAutoRenew` ClientOption
  - **File**: `gateway/traefik/traefik.go`
  - **Acceptance**: Option function works, defaults to `autoRenew = true`

- [x] 2.7 Modify `traefikClient.Register` to support auto-renewal
  - **File**: `gateway/traefik/traefik.go`
  - **Acceptance**:
    - Calls `handler.Register`
    - If `autoRenew && endpoint.TTL > 0`, starts renewal goroutine
    - Goroutine runs until stopped

- [x] 2.8 Implement `traefikClient.startAutoRenew` method
  - **File**: `gateway/traefik/traefik.go`
  - **Acceptance**:
    - Creates stop channel and goroutine
    - Calls `handler.Refresh` every TTL/3
    - On failure, retries with `handler.Register`
    - Stops on channel close or context cancel

- [x] 2.9 Modify `traefikClient.Deregister` to stop renewal goroutine
  - **File**: `gateway/traefik/traefik.go`
  - **Acceptance**: Calls `stopAutoRenew` before deregistering

- [x] 2.10 Implement `traefikClient.Close` to stop all goroutines
  - **File**: `gateway/traefik/traefik.go`
  - **Acceptance**: Closes all stop channels, prevents goroutine leaks

- [x] 2.11 Add `KeepAlive` method to `traefikClient`
  - **File**: `gateway/traefik/traefik.go`
  - **Acceptance**: 
    - Normalizes endpoint
    - Calls `handler.Refresh`
    - Propagates errors correctly

- [x] 2.12 Add `KeepAlive` method to `gateway.Client` interface
  - **File**: `gateway/gateway.go`
  - **Acceptance**: Interface compiles with new method

- [x] 2.13 Update `Endpoint.TTL` field comment
  - **File**: `gateway/types.go`
  - **Acceptance**: Comment clearly states "单位：秒。0 表示永不过期。"

- [x] 2.14 Update `handler.Update` to preserve TTL behavior
  - **File**: `gateway/traefik/handler.go`
  - **Acceptance**: Update uses same TTL logic as Register

- [x] 2.15 Ensure `handler.Deregister` works correctly with TTL configs
  - **File**: `gateway/traefik/handler.go`
  - **Acceptance**: Deregister deletes keys regardless of TTL

## Phase 3: Unit Testing (10 tasks)

- [ ] 3.1 Add test: `TestKvStore_PutWithTTL`
  - **File**: `gateway/traefik/kv_store/*_test.go`
  - **Acceptance**: Verifies Put with expired parameter sets TTL correctly (all backends)

- [ ] 3.2 Add test: `TestKvStore_KeepAlive`
  - **File**: `gateway/traefik/kv_store/*_test.go`
  - **Acceptance**: Verifies KeepAlive refreshes TTL (all backends)

- [ ] 3.3 Add test: `TestKvStore_BatchKeepAlive`
  - **File**: `gateway/traefik/kv_store/*_test.go`
  - **Acceptance**: 
    - Verifies BatchKeepAlive refreshes multiple keys
    - For Redis: verifies all keys have synchronized expiration timestamps
    - For Consul/Etcd/ZooKeeper: verifies correct behavior

- [ ] 3.4 Add test: `TestHandler_RegisterWithTTL`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: Verifies Put is called with TTL for URL and Weight when TTL > 0

- [ ] 3.5 Add test: `TestHandler_RegisterWithoutTTL`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: Verifies Put is called without TTL when TTL = 0

- [ ] 3.6 Add test: `TestHandler_RegisterDirectPutStrategy`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: 
    - Verifies Router configs use direct `Put` without prior `Get`
    - Verifies HealthCheck Path uses direct `Put` without prior `Get`
    - Verifies Service instance still uses `GetByPrefix` for index lookup

- [ ] 3.7 Add test: `TestHandler_Refresh`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: Verifies Refresh locates instance and calls BatchKeepAlive with correct keys

- [ ] 3.8 Add test: `TestHandler_RefreshInstanceNotFound`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: Verifies Refresh returns error when instance not found

- [ ] 3.9 Add test: `TestTraefikClient_AutoRenew`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: Verifies auto-renewal goroutine starts and calls Refresh periodically

- [ ] 3.10 Add test: `TestTraefikClient_AutoRenewDisabled`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: Verifies WithAutoRenew(false) prevents goroutine from starting

- [ ] 3.11 Add test: `TestTraefikClient_KeepAlive`
  - **File**: `gateway/traefik/traefik_test.go`
  - **Acceptance**: Verifies client-level KeepAlive normalizes endpoint and calls handler

## Phase 4: Integration Testing (6 tasks)

- [ ] 4.1 Add integration test: `TestTTLExpiration_Redis`
  - **File**: `gateway/traefik/integration_test.go` (new file)
  - **Acceptance**: 
    - Register service with TTL=2s, autoRenew=false
    - Wait 3s
    - Verify keys are deleted from Redis

- [ ] 4.2 Add integration test: `TestTTLExpiration_Consul`
  - **File**: `gateway/traefik/integration_test.go`
  - **Acceptance**: Same as 4.1 but for Consul

- [ ] 4.3 Add integration test: `TestTTLExpiration_Etcd`
  - **File**: `gateway/traefik/integration_test.go`
  - **Acceptance**: Same as 4.1 but for Etcd

- [ ] 4.4 Add integration test: `TestTTLExpiration_ZooKeeper`
  - **File**: `gateway/traefik/integration_test.go`
  - **Acceptance**: Same as 4.1 but for ZooKeeper

- [ ] 4.5 Add integration test: `TestAutoRenewalKeepAlive`
  - **File**: `gateway/traefik/integration_test.go`
  - **Acceptance**:
    - Register service with TTL=5s, autoRenew=true
    - Wait 15s
    - Verify keys remain in KV store (auto-renewed)

- [ ] 4.6 Add integration test: `TestServiceCrashCleanup`
  - **File**: `gateway/traefik/integration_test.go`
  - **Acceptance**:
    - Register 2 instances with TTL=3s, autoRenew=true
    - Close client 1 (stops renewal)
    - Wait 4s
    - Verify only instance 1 is deleted after TTL
    - Verify Router configs remain
    - Verify instance 2 still exists

## Phase 5: Documentation and Validation (4 tasks)

- [ ] 5.1 Update API documentation for gateway.Client
  - **File**: `gateway/gateway.go` (code comments)
  - **Acceptance**: 
    - `KeepAlive` method has complete godoc
    - Documents when to use auto-renewal vs manual

- [ ] 5.2 Update API documentation for Traefik client
  - **File**: `gateway/traefik/traefik.go` (code comments)
  - **Acceptance**: 
    - `Register` documents TTL behavior and auto-renewal
    - `WithAutoRenew` option documented
    - Examples show both auto and manual renewal

- [ ] 5.3 Update Endpoint.TTL field documentation
  - **File**: `gateway/types.go`
  - **Acceptance**: Clearly documents units (seconds) and behavior

- [ ] 5.4 Run full validation suite
  - **Command**: `go test -v ./gateway/...`
  - **Acceptance**: All tests pass, coverage > 90%

## Dependencies

- Phase 2 depends on Phase 1 (KV Store interface must be updated)
- Phase 3 depends on Phase 2 (implementation must exist)
- Phase 4 depends on Phase 1 and Phase 2 (requires full stack)
- Phase 5 can start after Phase 2 (documentation)

## Parallel Work Opportunities

- Tasks 1.3, 1.4, 1.5, 1.6 (各 KV Store 实现) can be done in parallel
- Tasks 2.1, 2.2 can be done in parallel
- Tasks 2.3, 2.4 can be done in parallel
- Phase 3 (unit tests) and Phase 4 (integration tests) can partially overlap
- Tasks 5.1, 5.2, 5.3 (documentation) can be done in parallel
