# Nginx Ingress SPOF — Task Checklist

Last Updated: 2026-02-19

## Phase 1: Make Nginx Resilient (Quick Win)

- [x] **1.1** Add liveness and readiness probes to nginx deployment in `atlas-ingress.yml`
- [x] **1.2** Scale nginx to 2 replicas in `atlas-ingress.yml`
- [x] **1.3** Add PodDisruptionBudget (minAvailable: 1) to `atlas-ingress.yml`
- [x] **1.4** Add preferred pod anti-affinity to spread across nodes
- [x] **1.5** Reduce proxy timeouts from 1800s to 30s (except WebSocket HMR path)

## Phase 2: Direct Service-to-Service Communication

- [ ] **2.1** Build complete domain-to-service URL mapping (audit all `RootUrl()` calls)
- [ ] **2.2** Audit each service's HTTP route registration to confirm path compatibility
- [ ] **2.3** Add per-service `_SERVICE_URL` env vars to `atlas-env.yaml`
- [ ] **2.4** Verify tenant headers propagate correctly on direct calls (no nginx)
- [ ] **2.5** Pilot: Migrate `atlas-transports` → `atlas-tenants` direct call
- [ ] **2.6** Migrate low-traffic service pairs:
  - [ ] `atlas-drops` → `atlas-configurations`
  - [ ] `atlas-account` → `atlas-ban`
  - [ ] `atlas-character` → `atlas-skills`
  - [ ] `atlas-character` → `atlas-data`
  - [ ] `atlas-inventory` → `atlas-data`
  - [ ] `atlas-inventory` → `atlas-pets`
  - [ ] `atlas-buddies` → `atlas-characters`
  - [ ] `atlas-monsters` → `atlas-maps`
  - [ ] `atlas-monsters` → `atlas-data`
  - [ ] `atlas-transports` → `atlas-data`
  - [ ] `atlas-transports` → `atlas-maps`
- [ ] **2.7** Migrate high-traffic service pairs:
  - [ ] `atlas-channel` → all outbound services
  - [ ] `atlas-saga-orchestrator` → all outbound services
  - [ ] `atlas-query-aggregator` → all outbound services
  - [ ] `atlas-login` → all outbound services
  - [ ] `atlas-monster-death` → all outbound services
- [ ] **2.8** Remove nginx location blocks for fully-migrated internal-only routes

## Phase 3: Update Debug Tooling

- [ ] **3.1** Design debug approach for direct service communication
- [ ] **3.2** Update `tools/debug-start.sh` to support new architecture
- [ ] **3.3** Update `tools/debug-stop.sh` to support new architecture
- [ ] **3.4** Test debug workflow end-to-end with a direct-call service

## Phase 4: Slim Down Nginx to Edge-Only

- [ ] **4.1** Identify which API routes the UI actually calls (external-facing subset)
- [ ] **4.2** Reduce nginx config to external-only routes
- [ ] **4.3** Evaluate replacing nginx with Traefik IngressRoute CRDs
- [ ] **4.4** If Traefik migration chosen: create IngressRoute resources per external service
- [ ] **4.5** Remove atlas-ingress deployment if fully replaced by Traefik
