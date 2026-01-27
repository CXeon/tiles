# Design: Traefik Gateway Client

## Context
The system needs to integrate with Traefik 3.6 using its KV configuration provider. The integration must be flexible enough to support multiple KV backends (Redis, Consul, etc.) while maintaining a clean separation of concerns.

## Goals
- Support Register, Deregister, Update, and Close operations.
- Abstract KV store operations from the business logic.
- Support multiple KV backends.
- Implement Traefik 3.6 specific configuration structures (Routers, Services, LoadBalancers).

## Non-Goals
- Support for non-KV Traefik providers (e.g., Docker, Kubernetes) in this specific implementation.
- Complex service mesh features like mTLS or circuit breaking (to be added later if needed).

## Architecture
The implementation follows a three-tier architecture:

1. **Traefik Layer (`traefik.go`)**: 
   - Implements `gateway.Client`.
   - Handles high-level decisions and parameter validation.
   - Normalizes protocols (e.g., `https` -> `http` for internal Traefik routing).
   - Manages the `handler` lifecycle.

2. **Handler Layer (`handler.go`)**:
   - The primary worker performing the registration/deregistration flow.
   - Coordinates between the `constructor` and `kv_store`.
   - Manages the `kv_store` connection lifecycle.

3. **Utility Layer**:
   - **Constructor (`constructor.go`)**: Stateless logic to generate Traefik-specific KV keys.
   - **KV Store (`kv_store/`)**: Interface and implementations for various KV backends.

## Decisions
- **Protocol Normalization**: When an endpoint uses `https`, it will be converted to `http` before passing to the handler, as Traefik routers/services usually communicate with backends over HTTP or handle TLS at the entrypoint level.
- **KV Store Factory**: A factory pattern will be used in `NewHandler` to instantiate the appropriate `KvStore` implementation based on the `ProviderType`.
- **LifeCycle**: The `KvStore` and `handler` will share the same lifecycle as the `Traefik` client.

## Risks / Trade-offs
- **KV Consistency**: Rapid updates to KV stores might lead to race conditions if not handled carefully. Mitigation: Use atomic operations where supported by the backend.
- **Traefik Versioning**: Changes in Traefik 3.6+ configuration schema might break the constructor. Mitigation: Keep the constructor modular and well-tested.

## Open Questions
- Should we implement TTL at the KV store level or via a separate heartbeat mechanism? (Deferred for now).
- How to handle global Traefik settings (Entrypoints, Middlewares) that might be shared across services?
