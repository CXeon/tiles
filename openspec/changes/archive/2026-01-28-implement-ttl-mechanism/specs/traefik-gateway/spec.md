# Capability: traefik-gateway

## ADDED Requirements

### Requirement: Automatic Cleanup via TTL Mechanism

The Traefik client SHALL support automatic cleanup of service instance configurations using TTL mechanism to prevent stale data after service crashes.

#### Scenario: Register service instance with TTL
- **WHEN** `Register` is called with `endpoint.TTL > 0`
- **THEN** the Service instance configurations (URL, Weight) MUST be stored with the specified TTL
- **AND** Router configurations MUST be stored without TTL (permanent until explicitly deleted)
- **AND** an automatic renewal goroutine MUST be started (if `autoRenew=true`, which is the default)

#### Scenario: Register service instance without TTL
- **WHEN** `Register` is called with `endpoint.TTL = 0`
- **THEN** all configurations MUST be stored without TTL (preserving existing behavior)
- **AND** no renewal goroutine should be started

#### Scenario: Service instance configuration expiration
- **WHEN** a Service instance with TTL is registered and the TTL expires without refresh
- **THEN** the instance's URL and Weight keys MUST be automatically deleted from the KV store
- **AND** Router configurations MUST remain in the KV store
- **AND** other instances' configurations MUST remain unaffected

---

### Requirement: Automatic Renewal Mechanism

The Traefik client SHALL provide automatic renewal mechanism to keep service instance configurations alive by default.

#### Scenario: Automatic renewal enabled (default)
- **WHEN** `Register` is called with `endpoint.TTL > 0` and `autoRenew=true` (default)
- **THEN** a background goroutine MUST be started to automatically refresh the instance configuration
- **AND** the goroutine MUST call `handler.Refresh` every `TTL/3` seconds
- **AND** if refresh fails, the goroutine MUST attempt to re-register the instance
- **AND** the goroutine MUST continue until `Deregister` is called or the client is closed

#### Scenario: Automatic renewal disabled
- **WHEN** `NewClient` is called with `WithAutoRenew(false)` option
- **THEN** automatic renewal MUST be disabled
- **AND** users MUST manually call `KeepAlive` to refresh instance configurations

---

### Requirement: Manual Renewal via KeepAlive

The Traefik client SHALL provide a manual KeepAlive method for advanced users who disable auto-renewal.

#### Scenario: Manual KeepAlive for registered instance
- **WHEN** `KeepAlive` is called with a valid endpoint
- **THEN** the system MUST locate the instance by its IP:Port
- **AND** refresh the TTL of the instance's URL and Weight keys
- **AND** use the TTL value from the endpoint

#### Scenario: KeepAlive for non-existent instance
- **WHEN** `KeepAlive` is called for an instance that is not registered
- **THEN** the system MUST return an error indicating the instance needs to be re-registered

#### Scenario: KeepAlive without TTL
- **WHEN** `KeepAlive` is called for an endpoint with `TTL = 0`
- **THEN** the system SHOULD return immediately without error (no-op behavior)

---

### Requirement: TTL Preservation Across Lifecycle Operations

The Traefik client SHALL preserve TTL behavior across lifecycle operations.

#### Scenario: Update service with TTL
- **WHEN** `Update` is called for a service registered with TTL
- **THEN** the new Service instance configurations MUST be stored with the same TTL
- **AND** the TTL timer MUST be reset (equivalent to re-registration)

#### Scenario: Deregister service with TTL
- **WHEN** `Deregister` is called for a service registered with TTL
- **THEN** the automatic renewal goroutine MUST be stopped
- **AND** the instance configurations MUST be deleted immediately (not waiting for TTL expiration)
- **AND** the deletion MUST succeed regardless of TTL state

#### Scenario: Close client with active renewals
- **WHEN** `Close` is called on the client
- **THEN** all automatic renewal goroutines MUST be stopped
- **AND** no goroutine leaks should occur

---

### Requirement: Configuration Lifecycle Separation

The Traefik client SHALL distinguish between shared and instance-specific configurations when applying TTL.

#### Scenario: Configuration TTL assignment
- **WHEN** a service is registered with TTL
- **THEN** the following configurations MUST have TTL:
  - Service instance URL (`traefik/http/services/<name>/loadbalancer/servers/<index>/url`)
  - Service instance Weight (`traefik/http/services/<name>/loadbalancer/servers/<index>/weight`)
- **AND** the following configurations MUST NOT have TTL:
  - Router Rule (`traefik/http/routers/<name>/rule`)
  - Router Service reference (`traefik/http/routers/<name>/service`)
  - Router Middlewares (`traefik/http/routers/<name>/middlewares/<index>`)
  - Service HealthCheck path (`traefik/http/services/<name>/loadbalancer/healthcheck/path`)

#### Scenario: Last instance cleanup with TTL
- **WHEN** the last instance of a service with TTL expires
- **THEN** only the instance-specific configurations MUST be deleted
- **AND** Router configurations MUST remain (allowing new instances to register without recreating routers)
- **AND** Traefik MUST return 503 for requests to the service (no available instances)

---

### Requirement: Endpoint TTL Field Documentation

The Endpoint structure SHALL clearly document the TTL field with unit and behavior.

#### Scenario: TTL field documentation
- **WHEN** Endpoint.TTL field is documented
- **THEN** the comment MUST state: "TTL 生存时间（单位：秒）。0 表示永不过期，> 0 表示 N 秒后自动删除实例配置。"
- **AND** the unit MUST be clearly specified as seconds (秒)
- **AND** the behavior for TTL=0 MUST be explicitly documented
