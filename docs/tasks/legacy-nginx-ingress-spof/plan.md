# Eliminate Single Nginx Ingress as SPOF

Last Updated: 2026-02-19

## Executive Summary

All inter-service REST communication in Atlas (~50 microservices) routes through a single-replica nginx pod (`atlas-ingress`). This creates a single point of failure — if the pod crashes, restarts, or is evicted, all service-to-service HTTP calls fail simultaneously. The fix is a two-phase approach: first make the ingress resilient with replicas + health checks (quick win), then eliminate the extra hop for internal traffic by switching to direct service-to-service communication.

## Current State Analysis

### Architecture
- **55 services** in the `atlas` namespace, ~50 of which communicate via REST
- **1 nginx pod** (`atlas-ingress`) handles all routing via ~70 regex location blocks
- Every service uses `BASE_SERVICE_URL=http://atlas-ingress.atlas.svc.cluster.local:80/api/` for outbound REST calls
- The `_SERVICE_URL` override mechanism exists in `libs/atlas-rest/requests/url.go` but is **unused** — no service defines a domain-specific URL
- External traffic flows through **two proxies**: Traefik Ingress -> nginx -> service
- Debug tooling (`tools/debug-start.sh`) depends on rewriting the nginx ConfigMap

### Problems
1. **Single point of failure**: 1 replica means any pod disruption kills all inter-service REST
2. **No health checks**: No liveness or readiness probes on the nginx deployment
3. **Unnecessary latency**: Every internal REST call adds an extra network hop through nginx
4. **No rate limiting or circuit breaking**: A misbehaving service can cascade failures through nginx
5. **Regex ordering fragility**: ~70 location blocks with regex matching are order-sensitive and error-prone
6. **30-minute proxy timeouts**: `proxy_read_timeout 1800` masks slow/hung backends

### What Works Well
- Centralized routing is simple to reason about
- Debug tooling leverages the central proxy effectively
- Tenant header forwarding is consistent across all routes
- The `_SERVICE_URL` escape hatch was already designed into the system

## Proposed Future State

### Target Architecture
```
External requests:  Traefik Ingress -> nginx (UI + external API only) -> services
Internal requests:  Service A -> K8s DNS -> Service B (direct)
```

### Key Changes
1. Nginx becomes an **edge proxy only** — serves external traffic (UI, external API consumers)
2. Internal service-to-service calls use **direct Kubernetes DNS** via per-service `_SERVICE_URL` env vars
3. Nginx runs with **2+ replicas, health checks, and a PodDisruptionBudget**
4. Debug tooling is updated to work with both direct and proxied traffic

## Implementation Phases

### Phase 1: Make Nginx Resilient (Quick Win)
**Goal**: Eliminate the immediate SPOF risk without any code changes.

#### 1.1 Add health checks to nginx deployment [S]
Add liveness and readiness probes to `atlas-ingress.yml`:
- Readiness: HTTP GET `/` → 200 (nginx default)
- Liveness: HTTP GET `/` → 200, with initial delay
- **Acceptance**: Pods restart automatically when nginx becomes unresponsive

#### 1.2 Scale nginx to 2 replicas [S]
Change `spec.replicas: 1` to `spec.replicas: 2` in `atlas-ingress.yml`.
- K8s Service already load-balances across pods with ClusterIP
- **Acceptance**: `kubectl get pods -l app=atlas-ingress` shows 2 running pods

#### 1.3 Add PodDisruptionBudget [S]
Create a PDB requiring at least 1 nginx pod available during node drains/upgrades.
- **Acceptance**: `kubectl get pdb` shows the budget; voluntary disruptions respect it

#### 1.4 Add anti-affinity to spread across nodes [S]
Add `podAntiAffinity` (preferred) to avoid scheduling both replicas on the same node.
- **Acceptance**: Pods land on different nodes when possible

#### 1.5 Reduce proxy timeouts to sane values [S]
Change from 1800s (30 min) to values matching the HTTP client timeout (30s for service calls). Keep a longer timeout only for specific paths that need it (e.g., WebSocket HMR).
- **Acceptance**: Hung backends time out in seconds, not minutes

### Phase 2: Direct Service-to-Service Communication
**Goal**: Eliminate nginx from the internal call path entirely. Services call each other directly via Kubernetes DNS.

#### 2.1 Build the domain-to-service URL mapping [M]
Create a reference mapping from every `RootUrl("DOMAIN")` call to its corresponding Kubernetes Service DNS name and the URL path prefix. For example:
```
CHARACTERS  -> http://atlas-character.atlas.svc.cluster.local:8080/
DATA        -> http://atlas-data.atlas.svc.cluster.local:8080/
SKILLS      -> http://atlas-skills.atlas.svc.cluster.local:8080/
INVENTORY   -> http://atlas-inventory.atlas.svc.cluster.local:8080/
...
```
- Cross-reference every `RootUrl()` domain key against the nginx location blocks to derive the correct backend
- **Acceptance**: Complete mapping document; every domain key resolves to exactly one K8s Service

#### 2.2 Add per-service URL env vars to atlas-env.yaml [M]
Add entries like:
```yaml
CHARACTERS_SERVICE_URL: "http://atlas-character.atlas.svc.cluster.local:8080/"
DATA_SERVICE_URL: "http://atlas-data.atlas.svc.cluster.local:8080/"
SKILLS_SERVICE_URL: "http://atlas-skills.atlas.svc.cluster.local:8080/"
```
Keep `BASE_SERVICE_URL` as a fallback for any unmapped domains.
- **Acceptance**: All domain keys used in `RootUrl()` have a corresponding `_SERVICE_URL` env var

#### 2.3 Verify URL path construction [L]
The current URL pattern is `BASE_SERVICE_URL + "characters/123"` which becomes `http://ingress:80/api/characters/123`. With direct URLs, each service's `_SERVICE_URL` must produce the correct path. Audit every `requests.go` file to confirm the path concatenation works when the base URL changes from `http://ingress/api/` to `http://atlas-character:8080/`.

Key concern: Some services use path prefixes like `/api/data/...` which nginx strips. With direct calls, the service must handle the path as-is.

- Review how each service registers its HTTP routes (what prefix does it listen on?)
- Verify that `_SERVICE_URL + path` produces the correct final URL
- **Acceptance**: Every service's URL construction produces valid paths for direct calls

#### 2.4 Ensure tenant headers propagate without nginx [M]
Currently nginx forwards `TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION` headers. The `TenantHeaderDecorator` in `libs/atlas-rest` already injects these headers on outbound requests, so they should propagate correctly without nginx. Verify this.
- **Acceptance**: Tenant headers present on all inter-service requests when bypassing nginx

#### 2.5 Roll out direct URLs incrementally [L]
Deploy one service pair at a time to validate:
1. Start with a low-risk, low-traffic pair (e.g., `atlas-transports` calling `atlas-tenants`)
2. Add the `TENANTS_SERVICE_URL` env var
3. Verify requests succeed with correct tenant context
4. Monitor for errors, then proceed to the next pair
- **Acceptance**: Each migrated pair shows 0 error rate in logs; tenant context intact

#### 2.6 Remove nginx location blocks for migrated services [M]
As services are migrated to direct calls, remove their corresponding nginx location blocks. The blocks are only needed for:
- External API access from the UI
- External clients
- Debug tooling
- **Acceptance**: nginx config shrinks; only external-facing routes remain

### Phase 3: Update Debug Tooling
**Goal**: Ensure `tools/debug-start.sh` works in the new architecture.

#### 3.1 Update debug scripts for direct communication [M]
When services call each other directly, the debug scripts can no longer just rewrite nginx. Options:
- **Option A**: Use a K8s Service override — patch the target service's K8s Service to point at a debug endpoint instead of the deployment
- **Option B**: Keep a debug-mode flag that temporarily switches a service back to `BASE_SERVICE_URL` routing through nginx, where the debug redirect is applied
- **Option C**: Use `kubectl port-forward` or a dedicated debug sidecar
- **Acceptance**: Developer can redirect traffic to local machine for any service

### Phase 4: Slim Down Nginx to Edge-Only
**Goal**: nginx serves only external traffic.

#### 4.1 Separate external-only nginx config [M]
Reduce the nginx config to only routes needed for external access:
- `/` → atlas-ui (frontend)
- `/api/*` → only routes the UI actually calls (likely a subset of all routes)
- `/_next/webpack-hmr` → atlas-ui (dev websocket)
- **Acceptance**: nginx config is dramatically smaller; internal-only routes removed

#### 4.2 Consider replacing nginx with Traefik IngressRoutes [M]
Since K3s already provides Traefik, the external routing could be done entirely with Traefik IngressRoute CRDs, eliminating the double-proxy (Traefik -> nginx -> service) for external requests. This would reduce the external path to Traefik -> service.
- **Acceptance**: External requests reach services through a single proxy hop

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Path construction breaks during migration | Medium | High | Audit every `requests.go`; deploy incrementally per service pair |
| Tenant headers lost on direct calls | Low | High | `TenantHeaderDecorator` already handles this; verify with integration test |
| Debug tooling breaks | Medium | Medium | Update debug scripts before full rollout; Option B provides a safe fallback |
| DNS resolution failures | Low | Medium | K8s CoreDNS is reliable; services already resolve infra services directly |
| Increased env var sprawl | Low | Low | Centralized in `atlas-env.yaml`; `BASE_SERVICE_URL` provides fallback |

## Success Metrics

1. **Zero downtime from nginx pod disruption** (Phase 1 — immediate)
2. **Reduced inter-service latency** by eliminating the nginx hop (Phase 2 — measurable via OpenTelemetry traces)
3. **No single pod whose failure breaks all REST communication** (Phase 2 — complete)
4. **Simplified nginx config** with only external routes (Phase 4)
5. **External requests traverse a single proxy** instead of two (Phase 4)

## Required Resources and Dependencies

- **atlas-rest library**: Already has the `_SERVICE_URL` override mechanism — no library changes needed
- **atlas-env.yaml**: Central ConfigMap for new env vars
- **atlas-ingress.yml**: Deployment manifest for nginx changes
- **OpenTelemetry**: Use existing tracing to compare latency before/after
- **tools/debug-start.sh, debug-stop.sh**: Must be updated for Phase 3

## Effort Estimates

| Phase | Effort | Description |
|-------|--------|-------------|
| Phase 1: Resilient nginx | S | Manifest-only changes, no code |
| Phase 2: Direct communication | L | Audit all request paths, add env vars, incremental rollout |
| Phase 3: Debug tooling | M | Rework debug scripts for new architecture |
| Phase 4: Edge-only nginx | M | Config cleanup, optional Traefik migration |

**Recommended order**: Phase 1 first (immediate risk reduction), then Phase 2 incrementally alongside normal development.
