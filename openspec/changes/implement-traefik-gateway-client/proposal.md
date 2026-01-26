# Change: Implement Traefik Gateway Client

## Why
The project requires a gateway client to register, deregister, and update service instances in Traefik 3.6. This is essential for dynamic service discovery and routing management via Traefik's KV configuration backend.

## What Changes
- Implement `gateway.Client` interface in `gateway/traefik/traefik.go`.
- Finish implementation of `gateway/traefik/handler.go` for core logic.
- Implement KV store backends (Redis, Consul, Etcd, ZooKeeper) in `gateway/traefik/kv_store/`.
- Add support for dynamic configuration via options (Router Middlewares, Service Weights, Healthchecks).
- Add parameter normalization (e.g., HTTPS to HTTP) in the Traefik decision layer.
- Ensure proper lifecycle management of handlers and KV stores.

## Impact
- Affected specs: `gateway-client`, `traefik-gateway`
- Affected code: `gateway/traefik/*`, `gateway/traefik/kv_store/*`
