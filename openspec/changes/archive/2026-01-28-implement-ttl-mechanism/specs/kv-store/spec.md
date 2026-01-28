# Capability: kv-store

## ADDED Requirements

### Requirement: TTL Support via Optional Parameter in Put Method

The KV Store SHALL support time-to-live (TTL) for key-value pairs via optional parameter in Put method.

#### Scenario: Put key with TTL
- **WHEN** `Put` is called with `expired` parameter (unit: seconds)
- **THEN** the key MUST expire and be automatically deleted after the specified duration
- **AND** the expiration precision SHOULD be within ±1 second (allowing for backend clock precision)

#### Scenario: Put key without TTL
- **WHEN** `Put` is called without `expired` parameter or with `expired=0`
- **THEN** the key MUST persist indefinitely (永不过期)

#### Scenario: Mix TTL and non-TTL keys
- **WHEN** both `Put` with TTL and without TTL are used in the same store
- **THEN** both types of keys MUST coexist without interference

---

### Requirement: TTL Renewal via KeepAlive Method

The KV Store SHALL support refreshing the TTL of existing keys via KeepAlive method.

#### Scenario: KeepAlive with explicit TTL
- **WHEN** `KeepAlive` is called with a key and optional TTL duration (unit: seconds)
- **THEN** the key's expiration MUST be reset to the new TTL from the current time

#### Scenario: KeepAlive with default TTL
- **WHEN** `KeepAlive` is called with a key but without TTL parameter
- **THEN** the key's expiration MUST be reset using a default or previously-set TTL value

#### Scenario: KeepAlive non-existent key
- **WHEN** `KeepAlive` is called for a key that does not exist
- **THEN** the operation MUST return an error (not create the key)

---

### Requirement: Batch TTL Renewal via BatchKeepAlive Method

The KV Store SHALL support batch refreshing TTL of multiple keys to ensure synchronized expiration times.

#### Scenario: BatchKeepAlive for multiple keys
- **WHEN** `BatchKeepAlive` is called with multiple keys and a TTL duration
- **THEN** all keys' expiration MUST be reset to the same TTL value
- **AND** for Redis implementation, all keys MUST have nearly identical expiration timestamps (within milliseconds)
- **AND** the operation SHOULD complete in a single round-trip (e.g., Redis Pipeline)

#### Scenario: BatchKeepAlive with empty key list
- **WHEN** `BatchKeepAlive` is called with an empty key list
- **THEN** the operation SHOULD return immediately without error (no-op)

#### Scenario: BatchKeepAlive for Session/Lease-based backends
- **WHEN** `BatchKeepAlive` is called for Consul/Etcd/ZooKeeper backends
- **THEN** the implementation MAY internally call a single Session/Lease renewal
- **AND** all keys sharing the same Session/Lease MUST be renewed together

---

### Requirement: Universal TTL Semantics Across Backends

The KV Store SHALL provide unified TTL semantics across all backend implementations (Redis, Consul, Etcd, ZooKeeper).

#### Scenario: Redis backend TTL implementation
- **WHEN** Redis backend is used with TTL
- **THEN** the implementation MUST use `SET key value EX seconds` for Put and `EXPIRE key seconds` for KeepAlive

#### Scenario: Consul backend TTL implementation
- **WHEN** Consul backend is used with TTL
- **THEN** the implementation MUST use Consul Session mechanism
- **AND** all TTL keys MUST share a single global Session (创建一次，复用)
- **AND** KeepAlive MUST use Session Renew

#### Scenario: Etcd backend TTL implementation
- **WHEN** Etcd backend is used with TTL
- **THEN** the implementation MUST use Etcd Lease mechanism
- **AND** Lease MUST be automatically renewed via `client.KeepAlive()` upon creation
- **AND** KeepAlive MAY be a No-op or call `KeepAliveOnce` for manual refresh

#### Scenario: ZooKeeper backend TTL implementation
- **WHEN** ZooKeeper backend is used with TTL
- **THEN** the implementation MUST use Ephemeral Nodes
- **AND** the `expired` parameter MAY be ignored (Ephemeral Nodes rely on connection liveness)
- **AND** code comments MUST clearly explain the limitation: "ZooKeeper 临时节点无法设置精确 TTL，依赖连接存活。当客户端连接断开时，临时节点自动删除。"
- **AND** KeepAlive SHOULD check node existence as a health check (No-op for actual renewal)

#### Scenario: Read operations on TTL keys
- **WHEN** `Get` or `GetByPrefix` is called on keys with TTL
- **THEN** the methods MUST return the values normally (TTL does not affect read operations before expiration)
