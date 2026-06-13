# Config-Status Projection Adoption (atlas-character-factory + atlas-world) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-12
---

## 1. Overview

`atlas-character-factory` and `atlas-world` load their per-tenant configuration
exactly once at process start, via a `sync.Once` REST fetch of
`configurations/tenants` from `atlas-configurations`
(`configuration.Init` → `requestAllTenants`). The resulting
`map[uuid.UUID]tenant.RestModel` is never refreshed for the life of the pod, and
the lookup path `GetTenantConfig(tenantId)` calls `log.Fatalf("tenant not
configured")` when the requested tenant is absent from that stale map.

This is operationally fragile. When a new tenant/version is provisioned **after**
a pod has already loaded its map (the normal case — services are long-lived,
tenants are added on demand), the first request for that tenant misses the map
and `log.Fatalf` **crashes the entire pod**. This was observed in production on
2026-06-12: creating an Evan character on the newly provisioned GMS v84.1 tenant
hit `atlas-character-factory`'s `/api/characters/seed` endpoint, missed the
stale tenant map, and crash-looped both factory replicas (`tenant not
configured`, RESTARTS incrementing). The v84 tenant was present and correct in
both `atlas-tenants` and `atlas-configurations` — the only defect was that the
already-running factory had never reloaded it.

`atlas-login` and `atlas-channel` already solved this. `atlas-configurations`
runs a transactional outbox that publishes every service/tenant config
add/change/delete to two log-compacted Kafka topics
(`EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`,
`EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`). Login and channel each run a
`configuration/projection` package that consumes those topics, maintains an
in-memory snapshot, gates readiness on a one-shot end-offset catch-up, applies
add/change/tombstone events live, and re-publishes the snapshot into a registry
whose `Get*` accessors **return errors** (`ErrTenantNotConfigured`,
`ErrNotReady`) instead of crashing.

This task adopts that same Kafka-backed projection pattern in
`atlas-character-factory` and `atlas-world`, replacing their legacy one-shot REST
load. The factory needs only the **tenant** half of the projection (it has no
socket listeners). World needs the tenant half plus a re-initialization hook for
its world-rate registry. Neither `atlas-configurations` (the producer) nor the
projection consumers in login/channel are modified.

## 2. Goals

Primary goals:

- A tenant provisioned, changed, or deleted in `atlas-configurations` **after**
  `atlas-character-factory` / `atlas-world` pods have started is reflected in
  those services **live**, without a pod restart.
- A request for a genuinely-unconfigured tenant returns a clean, client-facing
  error path — **never** `log.Fatalf` / process exit.
- Both services gate Kubernetes readiness (`/readyz`) on the projection having
  caught up to the config topic's boot end-offset, so they do not serve traffic
  against an empty snapshot.
- The implementation copy-ports the existing, proven login/channel
  `configuration/projection` pattern (tenant subset), keeping behavior and
  envelope/schema versioning consistent across services.
- The original v84-Evan failure mode is provably gone: a new GMS tenant can be
  provisioned while the factory is running and a character of that tenant can be
  created without crashing the pod.

Non-goals:

- Modifying `atlas-configurations`' producer/outbox side, the config-status
  topics, or their envelope/schema contract.
- Touching the projection implementations already living in `atlas-login` /
  `atlas-channel`.
- Adopting the **service-config** half of the projection
  (`EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`) — neither the factory nor world
  runs per-tenant socket listeners, so they need only the tenant topic.
- Extracting a shared `libs/` projection module. This task copy-ports per
  service to match the existing login/channel precedent; shared-lib extraction
  is explicitly deferred (would be a larger cross-service refactor touching four
  services).
- Any packet-layer / character-creation-encoding work (e.g. the v84 `create.go`
  subJobIndex boundary). That is a separate concern tracked elsewhere; this task
  only addresses the config-load/crash layer.

## 3. User Stories

- As an **operator**, I want to provision a new GMS version tenant while the
  factory and world services are already running, so that players on the new
  version can create and load characters immediately — without me having to
  manually restart pods.
- As an **operator**, I want a config change (e.g. world rates, character
  creation templates) to propagate to running pods within seconds, so I don't
  have to roll the deployment to apply config.
- As a **player** on a tenant that is mis- or un-configured, I want character
  creation to fail gracefully (the client shows a creation error), rather than
  the backend crash-looping and taking the whole service down for everyone.
- As an **SRE**, I want `atlas-character-factory` and `atlas-world` to report
  not-ready until their config snapshot is loaded, so Kubernetes doesn't route
  requests to a pod that would error or crash.

## 4. Functional Requirements

### 4.1 Tenant config projection (both services)

- FR-1. Each service SHALL run a Kafka consumer subscribed to the tenant
  config-status topic resolved from `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`,
  starting from `kafka.FirstOffset` so log-compacted history is replayed in
  full.
- FR-2. The consumer SHALL decode each message as a `TenantEnvelope`
  (`schema_version`, `id`, `config`, `emitted_at`) and apply it to an in-memory
  `State`:
  - non-tombstone → insert/replace tenant config for `env.Id`;
  - tombstone (nil value, key `tenant:<uuid>`) → delete tenant config for that
    uuid.
- FR-3. Envelopes with `schema_version` greater than the projection's
  `SupportedSchemaVersion` SHALL be logged and skipped (forward-compatible), not
  retried and not fatal — mirroring login/channel.
- FR-4. Decode failures and apply failures SHALL be logged at WARN and skipped;
  they MUST NOT crash the process or block the consumer.
- FR-5. The projection MUST NOT consume or require the service-config topic.
  An unset `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` SHALL be logged as a clear
  warning ("tenant config updates will not propagate live") and leave the
  service running on whatever snapshot it has (degraded, not crashed).

### 4.2 Readiness gating (both services)

- FR-6. Each service SHALL snapshot the tenant topic's end offsets at startup
  and expose a one-way `CaughtUp` gate that flips once consumed offsets reach
  the boot end-offsets (an empty topic counts as trivially caught up).
- FR-7. Each service SHALL mount a `/readyz` endpoint that reports **not-ready**
  until `CaughtUp` has flipped, and **not-ready** once graceful shutdown begins.
- FR-8. Each service's Kubernetes Deployment SHALL declare a `readinessProbe`
  targeting `/readyz` so the pod is kept out of Service rotation until the
  config snapshot is loaded.
- FR-9. Catch-up SHALL be bounded by a configurable timeout
  (`PROJECTION_CATCHUP_TIMEOUT_S`, default mirroring login). On timeout the
  service behaves as login does today (startup fails loudly / stays not-ready)
  rather than serving against an empty snapshot.

### 4.3 Registry accessors return errors (both services)

- FR-10. `GetTenantConfig(tenantId)` SHALL return `(tenant.RestModel, error)`
  with:
  - `ErrNotReady` when the projection has not yet published a first snapshot
    (transient — caller logs at DEBUG and skips);
  - `ErrTenantNotConfigured` when the tenant is absent from a ready snapshot
    (persistent — caller logs at ERROR and returns a request failure).
  - It SHALL NOT call `log.Fatalf` under any condition.
- FR-11. `atlas-world`'s `GetTenantConfigs()` (plural, all tenants) SHALL be
  converted to a snapshot read that returns an error or empty result instead of
  `log.Fatalf` on an empty map, and all callers updated accordingly.
- FR-12. On each observed change (post-catch-up), the projection SHALL
  re-publish the current snapshot into the registry so existing
  `GetTenantConfig` callers (the factory seed/preset path; the world rate and
  channel-status paths) see fresh data.

### 4.4 atlas-character-factory specifics

- FR-13. The factory's `configuration.Init` (sync.Once REST load) SHALL be
  removed and replaced by the projection wiring in `main.go`. The seed/preset
  paths (`factory/processor.go:GetTenantConfig`,
  `configuration/preset_requests.go`) SHALL consume the projection-backed
  registry unchanged at their call sites (they already handle the returned
  error — `factory/processor.go:102-105`).
- FR-14. When `GetTenantConfig` returns `ErrTenantNotConfigured` for a seed
  request, the factory SHALL surface the existing creation-failure path so
  `atlas-login` announces an `AddCharacter` error to the client (current
  behavior on a returned error), instead of crashing.

### 4.5 atlas-world specifics

- FR-15. World's per-tenant world-rate initialization (today
  `initializeRatesFromConfig`, run once inside `Init`) SHALL run when a tenant
  config is first applied AND re-run when that tenant's config changes, so rate
  updates propagate live. A tenant tombstone SHOULD be handled coherently
  (rates for a removed tenant need not be actively torn down for v1, but MUST
  NOT crash).
- FR-16. World's boot-time use of `GetTenantConfigs()` at `main.go:89`
  (`model.ForEachMap(... RequestStatus ...)`) SHALL be sequenced **after**
  projection catch-up so it operates on a populated snapshot.

## 5. API Surface

No external/JSON:API surface changes. One new operational HTTP endpoint per
service:

- `GET /readyz` — returns 200 when caught up and not shutting down, 503
  otherwise. (Mirrors `atlas-login`'s readiness route; no body contract beyond
  status code.)

Kafka consumption (inbound, existing topic — no new contract):

- Topic: `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` (log-compacted).
- Value: `TenantEnvelope { schema_version:int, id:string, config:json,
  emitted_at:string }`; nil value = tombstone; key = `tenant:<uuid>`.

## 6. Data Model

No database schema or migration changes. The only "data model" is the in-memory
projection `State`:

- `State.tenants: map[uuid.UUID]tenant.RestModel` (per service's own
  `tenant.RestModel`), guarded by `sync.RWMutex`.
- `State.ApplyTenant(env)` unmarshals `env.Config` into the service's
  `tenant.RestModel` and assigns the parsed `env.Id` (note: each service's
  `tenant.RestModel.Id` is `json:"-"`, populated separately — verify the
  envelope `config` payload unmarshals cleanly into the factory/world
  `tenant.RestModel`, since the REST path previously populated `Id` via
  JSON:API `SetID` whereas the projection uses plain `json.Unmarshal` + explicit
  id assignment).

## 7. Service Impact

- **atlas-character-factory** — copy-port the tenant subset of the projection
  package (`envelope.go`, `caughtup.go`, tenant-only `state.go`, tenant-only
  `subscriber.go`, plus a lightweight snapshot→`PublishSnapshot` bridge in place
  of login's listener `ApplyLoop`). Rewrite `configuration/registry.go` to the
  error-returning, readiness-gated form. Remove `configuration.Init` /
  `requestAllTenants`. Wire projection startup, `/readyz`, and graceful-shutdown
  not-ready into `main.go`. Add `readinessProbe` to
  `deploy/k8s/base/atlas-character-factory.yaml`.
- **atlas-world** — same copy-port and registry rewrite, plus: re-run
  `initializeRatesFromConfig` on tenant apply/change (FR-15); sequence the
  `main.go:89` boot status sweep after catch-up (FR-16); convert
  `GetTenantConfigs()` off `log.Fatalf` (FR-11). Add `readinessProbe` to
  `deploy/k8s/base/atlas-world.yaml`.
- **atlas-configurations** — no code change (already emits the outbox + topic).
- **atlas-login / atlas-channel** — no change (reference implementations only).
- **Deployment env** — `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` is already
  injected into both services via `envFrom: configMapRef: atlas-env`
  (`deploy/k8s/base/env-configmap.yaml`), so no new env key is required for the
  topic. `PROJECTION_CATCHUP_TIMEOUT_S` may be added if a non-default is wanted.

## 8. Non-Functional Requirements

- **Multi-tenancy** — projection is keyed by tenant UUID; one
  unconfigured/broken tenant MUST NOT affect requests for other tenants
  (directly fixes the all-tenants-down crash).
- **Observability** — structured logs for: projection start, catch-up complete,
  each applied add/change/tombstone (DEBUG), decode/apply failures (WARN),
  `ErrTenantNotConfigured` at the request layer (ERROR), `ErrNotReady` (DEBUG).
  Reuse the `projection.*` log keys from login for consistency.
- **Startup ordering** — consumers register and catch up before the service
  reports ready; the per-process consumer group id pattern from login
  (`"<group> - projection - <uuid>"`) SHALL be reused so each replica replays the
  compacted topic independently.
- **Resilience** — a missing/late topic or a wedged projection surfaces as
  not-ready + request-level errors, never as a crash loop.
- **Performance** — snapshot reads are RW-locked map lookups; re-publish on
  change is O(tenants), negligible at current tenant counts (<10).
- **Backward compatibility** — envelope `SupportedSchemaVersion` MUST match the
  value used by login/channel at port time; bump in lockstep if the shared
  contract advances.

## 9. Open Questions

- **Q1 (world rates on change):** For FR-15, is re-running
  `initializeRatesFromConfig` on every tenant apply sufficient, or do live rate
  changes need to invalidate/replace existing `rate.GetRegistry()` entries
  (vs. only initializing missing ones)? Confirm desired semantics during design.
- **Q2 (world tombstone):** Is any active teardown wanted when a tenant is
  removed from world (rates, in-flight channel status), or is "stop serving new
  requests for it" acceptable for v1? (PRD assumes the latter.)
- **Q3 (bridge shape):** The factory has no listener diff loop; confirm the
  preferred bridge is "publish snapshot on each applied change" (simplest) vs.
  a login-style periodic ticker. Design decision, not a requirement.
- **Q4 (Id population):** Confirm the tenant envelope `config` payload
  round-trips into the factory/world `tenant.RestModel` via plain
  `json.Unmarshal` (the REST path used JSON:API `SetID`). If field tags differ,
  the port must reconcile them.

## 10. Acceptance Criteria

- [ ] `atlas-character-factory` consumes `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`
      and serves `GetTenantConfig` from the live projection snapshot.
- [ ] `atlas-world` does the same and re-initializes world rates on tenant
      apply/change.
- [ ] No `log.Fatalf("tenant not configured")` remains in either service;
      `grep` for it in both services returns nothing.
- [ ] `GetTenantConfig` returns `ErrNotReady` before first snapshot and
      `ErrTenantNotConfigured` for an absent tenant in a ready snapshot; callers
      handle both without crashing.
- [ ] Both services expose `/readyz` gated on projection catch-up, and both
      Deployments declare a `readinessProbe` against it.
- [ ] **Repro test:** with a factory pod already running, provision a new GMS
      tenant in `atlas-configurations`; within seconds the factory's snapshot
      includes it and a seed request for that tenant succeeds (or fails
      gracefully on validation) **without a pod restart**. Confirm `RESTARTS`
      does not increment.
- [ ] **Delete test:** tombstoning a tenant removes it from the snapshot live;
      a subsequent request returns `ErrTenantNotConfigured` (no crash).
- [ ] Per CLAUDE.md "Build & Verification": `go test -race ./...`, `go vet
      ./...`, `tools/redis-key-guard.sh`, and `go build ./...` clean in every
      changed module; `docker buildx bake atlas-character-factory` and
      `docker buildx bake atlas-world` succeed from the worktree root.
- [ ] Existing factory/world behavior (character seed, world rate/channel
      status) is unchanged for already-configured tenants.
