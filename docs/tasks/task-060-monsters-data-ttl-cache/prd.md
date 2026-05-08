# atlas-monsters Data TTL Cache — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-08
---

## 1. Overview

`atlas-monsters` resolves static monster reference data (name, HP/MP, EXP, level, weapon/magic attack & defense, animation timings, attacks, skills, resistances, revives, banish, self-destruct, cool-damage) by calling `GET /api/data/monsters/{id}` on `atlas-data` for every code path that needs it: aggro, picker, recovery, status, basic-attack, drop generation, etc. The lookup happens through `monster/information.GetById`, which is currently a thin wrapper that issues an HTTP request on every invocation with no caching.

A short capture from `tmp/atlas-monster-dbg-log` (35 s window, 2026-05-08 07:17:06–07:17:41) shows **850 GET requests** against `atlas-data` (~24 rps), with each unique monster id polled **23–26 times** in that window. The data is WZ-derived and effectively immutable between `atlas-data` deployments. Re-fetching it dozens of times per minute is wasted upstream load on `atlas-data`, wasted network and CPU in `atlas-monsters`, and unnecessary latency on hot per-tick code paths (aggro, picker, recovery).

This task introduces a small **Redis-backed TTL cache** in front of `information.GetById`. The cache is tenant-keyed, has a configurable positive-entry TTL (default **5 minutes**) and negative-entry TTL (default **30 seconds**). Cache invalidation on `atlas-data` redeploy is explicitly out of scope for automatic invalidation — TTL expiry is the steady-state mechanism, and a future task will add an event-driven flush.

The cache reuses `libs/atlas-redis.TenantRegistry`, which already provides tenant-scoped CRUD with native Redis EX TTL via `PutWithTTL`. Two registry instances are used: one in the namespace `monsters:cache:data` for positive entries (TTL 5 m default), and one in `monsters:cache:data:not_found` for negative entries (TTL 30 s default). Cache state lives in the Redis instance that `atlas-monsters` already depends on for cooldowns, drops, and the monster registry — no new shared library, no new runtime dependency. The cache is now shared across `atlas-monsters` pods and survives pod restarts; the formal invalidation mechanism is TTL expiry, with `redis-cli DEL` (pattern-based) as the operator-side flush. A future task adds Kafka-driven invalidation on `atlas-data` redeploys.

## 2. Goals

Primary goals:

- Eliminate the bulk of repeat `GET /api/data/monsters/{id}` calls from `atlas-monsters` to `atlas-data`. Target: **>95% cache hit rate** in steady state, measured via metrics over a 5-minute window once the cache has warmed.
- Cache positive responses with a **configurable TTL (default 5 m)**.
- Cache negative responses (errors / not-found) with a **configurable shorter TTL (default 30 s)** to absorb retry storms on bad ids.
- Make the cache **tenant-scoped** so two tenants with different `atlas-data` configurations never see each other's entries.
- Make the cache **opt-in via a single call site change** — wrap `information.GetById` only; no callers change.
- Reuse the existing `libs/atlas-redis.TenantRegistry` primitive so no new shared library is introduced; the same pattern can be applied to other reference-data lookups in follow-up tasks.
- Provide observability: hit/miss/negative-hit counters and Redis-error counters, scoped by tenant.

Non-goals:

- No event-driven invalidation (Kafka or otherwise) on `atlas-data` redeploys. Tracked as a future task. TTL expiry plus operator-side `redis-cli DEL` is the supported flush in v2.
- No admin / HTTP endpoint to manually flush the cache.
- No single-flight / request coalescing in v1. The 5-minute TTL plus the existing call rate is sufficient to drop upstream load by >95%; coalescing would only help during the first ~5 ms of a cold cache window. Revisit only if metrics show meaningful thundering-herd misses.
- No caching of other `atlas-data` resources (maps, items, npcs, drop tables, etc.). Each is a separate task.
- No changes to `atlas-channel` or `atlas-monster-death`, which also call `data/monsters/{id}` via similar `information` packages. Tracked as follow-ups so this task ships small.
- No changes to `atlas-data` itself (no new endpoints, no eviction headers, no ETag/If-None-Match support).

## 3. User Stories

- As an operator of `atlas-data`, I want `atlas-monsters` to stop hitting me ~24 times per second for the same handful of monster ids so that my service has headroom for genuinely dynamic queries.
- As a developer working on per-tick monster behavior (aggro, picker, recovery, basic attack), I want `information.GetById` to return in microseconds on the warm path so that hot loops don't pay HTTP latency.
- As an SRE diagnosing high latency or `atlas-data` saturation, I want hit/miss/negative-hit counters labeled by tenant so I can see at a glance whether the cache is doing its job.
- As a multi-tenant operator, I need confidence that a tenant override of `atlas-data` reference data never leaks into another tenant's cached responses.
- As a developer adding caching to other `atlas-data` lookups in follow-up tasks (`map/information`, `item/information`, etc.), I want a generic, well-tested in-process TTL cache library I can reuse without copy-pasting concurrency logic.

## 4. Functional Requirements

### 4.1 Reuse `libs/atlas-redis.TenantRegistry`

- No new module. Two `redislib.TenantRegistry` instances back the cache:
  - Positive: `redislib.NewTenantRegistry[uint32, RestModel](rc, "monsters:cache:data", uint32KeyFn)`.
  - Negative: `redislib.NewTenantRegistry[uint32, struct{}](rc, "monsters:cache:data:not_found", uint32KeyFn)`.
- The cached payload is `RestModel`, not `Model`: `Model` has unexported fields and is not JSON-serializable, while `RestModel` is the upstream wire format. `GetById` runs `Extract(rm)` after a positive hit to convert to `Model`.
- `TenantRegistry.Get` returns `redislib.ErrNotFound` on a Redis miss; the wrapper interprets that as "fall through" and `errors.Is`-checks for it explicitly. Any other Redis error is treated as a soft failure: log, increment the `redis_errors_total` counter, fall through to the upstream HTTP path.
- TTL is enforced natively by Redis via `PutWithTTL`. No app-side sweeper. No `OnEviction` callback.

### 4.2 `DataCache` wrapper in `atlas-monsters`

- Add a small `*DataCache` struct inside `services/atlas-monsters/atlas.com/monsters/monster/information/` (or a new sub-package, e.g. `information/cache`) that holds two `*redislib.TenantRegistry` pointers (positive + negative) and the loaded TTL configuration.
- Constructed once at process start and passed via the existing logger/context plumbing — no `sync.Once` per-tenant lazy init in v2. Tenant identity is supplied by `tenant.MustFromContext(ctx)` on each `GetById` call.
- Keys: `uint32` monster id (positive cache and negative cache share the same key space; one entry per id per tenant, per Redis namespace).
- Cache values are the upstream `information.RestModel`, not the parsed `Model`: `Model` has unexported fields and is not JSON-serializable. `Extract(rm)` runs on every positive hit; the cost is negligible compared to the avoided HTTP request.

### 4.3 Read-through wrapper around `information.GetById`

- The existing `information.GetById(l)(ctx)(monsterId) (Model, error)` call signature is preserved. Call sites do not change.
- Internally, `GetById` performs:
  1. Resolve `tenant.Model` from `ctx` via `tenant.MustFromContext`.
  2. Call the positive registry's `Get(tenant, monsterId)`. On hit, run `Extract(rm)` and return the resulting `Model`.
  3. On `redislib.ErrNotFound`, call the negative registry's `Get(tenant, monsterId)`. On hit, return the canonical "not found" error (preserve current error semantics — see §4.6).
  4. Otherwise, issue the existing HTTP request via `requests.Provider[RestModel, Model]`.
  5. On success: call the positive registry's `PutWithTTL(tenant, monsterId, restModel, positiveTTL)` and return the extracted `Model`.
  6. On failure: classify the error (see §4.6) and either call the negative registry's `PutWithTTL(tenant, monsterId, struct{}{}, negativeTTL)` or do not cache (transient errors). Return the error to the caller unchanged.
- Any non-`ErrNotFound` Redis error from a `Get` or `PutWithTTL` call is logged, increments `redis_errors_total`, and falls through to the upstream HTTP path. The caller never sees a Redis error.
- **No behavioral changes** for callers on success. On error, the error type and message returned to callers must match what `information.GetById` returns today.

### 4.4 Configuration

Configuration is read at process start via environment variables (matching how `atlas-monsters` already reads other tunables — confirm exact pattern at design time). The values are NOT loaded from `atlas-tenants` configurations in v1; tenant overrides are out of scope.

| Env var | Purpose | Default | Min | Max |
|---|---|---|---|---|
| `MONSTER_DATA_CACHE_TTL` | Positive-entry TTL, Go duration string | `5m` | `1s` | `24h` |
| `MONSTER_DATA_CACHE_NEGATIVE_TTL` | Negative-entry TTL | `30s` | `0s` (disables negative cache) | `5m` |
| `MONSTER_DATA_CACHE_ENABLED` | Master kill-switch | `true` | n/a | n/a |

When `MONSTER_DATA_CACHE_ENABLED=false`, `information.GetById` bypasses the cache entirely and behaves exactly as it does today. This is the rollback mechanism if the cache misbehaves in production.

Invalid duration strings or values outside the min/max range cause the process to log a warning and fall back to the default for that variable. They do NOT cause startup to fail.

### 4.5 Multi-tenancy

- Tenant identity is encoded into the Redis key via the existing `tenantEntityKey` helper (`libs/atlas-redis/keys.go:27`), which yields `atlas:<namespace>:<tenantUUID>:<region>:<major>.<minor>:<id>`. Two tenants writing the same monster id produce distinct Redis keys; cross-tenant reads are not possible without explicitly constructing a different `tenant.Model`.
- A single pair of `TenantRegistry` instances handles all tenants. There is no per-tenant struct lazy-initialization in v2 — the registry is constructed once and `tenant.Model` is passed into each `Get`/`PutWithTTL` call.

### 4.6 Error handling and negative caching

- The current `information.GetById` returns whatever error the underlying `requests.Provider` surfaces — typically a wrapped HTTP error including 404 and network errors. The cached implementation must preserve the same error values for callers.
- The cache layer classifies errors into two buckets:
  - **Cacheable as negative:** HTTP 404 / "not found". These are deterministic given the upstream data, and the caller will get the same answer for the next 30 s anyway.
  - **Not cacheable:** all other errors (HTTP 5xx, timeouts, network errors, JSON-parse errors). These are likely transient; caching them would suppress automatic recovery. Pass through to the caller without caching.
- If `MONSTER_DATA_CACHE_NEGATIVE_TTL` is `0s`, negative caching is disabled entirely; all errors bypass the cache.
- A negative-cache hit returns the same error type/message as the most recent upstream not-found response would have. Implementation MAY store the error or MAY reconstruct it — the contract is observed-from-caller equivalence, not byte equality.

### 4.7 Observability

Add the following metrics (Prometheus-style; final naming aligned with existing `atlas-monsters` conventions at design time):

- `atlas_monsters_data_cache_hits_total{tenant, kind}` — counter. `kind` is `positive` or `negative`.
- `atlas_monsters_data_cache_misses_total{tenant}` — counter, incremented on every upstream HTTP request issued.
- `atlas_monsters_data_cache_errors_total{tenant, classification}` — counter. `classification` is `not_found` (negative entry recorded) or `transient` (not cached).
- `atlas_monsters_data_cache_redis_errors_total{tenant, operation}` — counter. `operation` is `get_positive`, `get_negative`, `put_positive`, or `put_negative`. Each increment indicates a graceful fallthrough.

No `cache_size` gauge in v2: counting Redis keys requires `SCAN` per sample, which is operationally expensive. Operators introspect cache occupancy via `redis-cli DBSIZE` or `INFO keyspace` when needed; the four counters above are sufficient for hit-rate, error-rate, and degraded-mode dashboards.

### 4.8 Testing

- No new library means no new library-level test suite. `libs/atlas-redis.TenantRegistry` already has its own coverage upstream.
- Service-side wrapper ships with unit tests covering: tenant isolation (two tenants don't share entries), positive cache hit avoids HTTP (using `miniredis` or an equivalent fake), negative cache hit avoids HTTP, non-cacheable errors are not cached, kill-switch disables caching, Redis-error fallthrough still produces the right answer, and the error returned on a cache hit matches the error from a non-cached miss for the same input.
- Existing `atlas-monsters` tests must continue to pass without modification. If they currently rely on `information.GetById` issuing real HTTP calls, the wrapper must be transparent enough that they still work.
- `go test -race ./...` in `services/atlas-monsters/atlas.com/monsters` must be clean.

## 5. API Surface

No changes to any HTTP, JSON:API, or Kafka surface.

The internal Go API of `monster/information` is unchanged. `information.GetById(l)(ctx)(monsterId) (Model, error)` keeps the same signature, semantics, and error types.

No new shared library is introduced. The cache is a thin wrapper over `libs/atlas-redis.TenantRegistry` (already a dependency of `atlas-monsters`).

## 6. Data Model

No persistent data changes. No database migrations. No new Kafka topics.

In-memory state in `atlas-monsters`: a single `*DataCache` struct holding two `*redislib.TenantRegistry` pointers and the loaded TTL configuration. Cache payloads live in Redis under the namespaces `atlas:monsters:cache:data:*` (positive, JSON-encoded `RestModel`) and `atlas:monsters:cache:data:not_found:*` (negative, JSON-encoded empty `struct{}`).

## 7. Service Impact

### 7.1 `services/atlas-monsters`

- No new external dependency. `libs/atlas-redis` is already a transitive dependency.
- `monster/information/processor.go`: `GetById` changes from a one-line passthrough to a read-through cache. Behavior on the success path is identical; behavior on the error path adds short-lived negative caching for not-found.
- `monster/information/`: add cache wrapper file (e.g. `cache.go` or new sub-package `cache/`) that constructs the two `redislib.TenantRegistry` instances and exposes the read-through `GetById` helper.
- New env vars wired up wherever the service currently reads tunables.
- New metrics registered on the existing Prometheus registry.

### 7.2 `libs/atlas-redis` (existing)

- No code changes in v2. The pre-existing `TenantRegistry` + `PutWithTTL` API is sufficient.

### 7.3 `services/atlas-data`

- No code changes. The cache reduces inbound `GET /api/data/monsters/{id}` traffic from `atlas-monsters`. Other clients (`atlas-channel`, `atlas-monster-death`) are unchanged for now.

### 7.4 Other services

- No changes in this task.
- Follow-up tasks (out of scope here):
  - Apply the same caching pattern to `services/atlas-channel/atlas.com/channel/monster/information`.
  - Apply the same caching pattern to `services/atlas-monster-death/atlas.com/monster/monster/information`.
  - Investigate whether `atlas-maps`, `atlas-npcs`, `atlas-items`, etc. have analogous polling patterns.
  - Add a Kafka-event-driven cache flush triggered by `atlas-data` deploys.

## 8. Non-Functional Requirements

### 8.1 Performance

- p99 cache-hit latency ≤ 5 ms in-cluster. Typical Redis GET against an in-cluster instance is 0.2–1 ms; the budget allows for tail. Validation is operational (existing OpenTelemetry tracing on `atlas-monsters`); no microbenchmark gates this task.
- A cache miss adds at most two Redis GETs (positive + optional negative) before falling through to the existing HTTP path. The combined miss overhead must be statistically dwarfed by the upstream HTTP latency it replaces.
- Memory: state lives in the existing Redis instance, not in `atlas-monsters` heap. Worst case ≈ a few thousand `RestModel` JSON blobs × ≤5 tenants ≈ low single-digit MB. Negligible against the existing Redis allocation.

### 8.2 Correctness

- No stale data older than `MONSTER_DATA_CACHE_TTL` may be returned. (Bound: caller observes data that was upstream-current within the last TTL window.)
- No data may leak between tenants under any code path, including concurrent first-touch by two tenants.
- Race-detector clean: `go test -race ./...` in `services/atlas-monsters/atlas.com/monsters`.

### 8.3 Observability

- Metrics §4.7 are present and labeled.
- One operator-facing dashboard panel can answer "what's the hit rate?" from `hits_total / (hits_total + misses_total)` per tenant. Dashboard authoring is out of scope; metric availability is in scope.

### 8.4 Security & multi-tenancy

- Tenant isolation enforced at the cache key level. No code path in `atlas-monsters` may read a cached entry for a tenant other than the one in `ctx`.
- No new secrets, no new external network egress.

### 8.5 Operability

- Master kill-switch (`MONSTER_DATA_CACHE_ENABLED=false`) bypasses both Redis registries and behaves identically to the pre-task implementation. Restart of the pod after env-var change is acceptable.
- Cache state is shared across `atlas-monsters` pods and survives pod restarts. Native Redis TTL is the steady-state invalidation mechanism. Operator-side flush is `redis-cli --scan --pattern 'atlas:monsters:cache:data:*' | xargs redis-cli DEL` (and similarly for the `not_found` namespace) — no pod restart required. A future task adds Kafka-event-driven invalidation on `atlas-data` redeploys.
- Redis unavailability degrades the cache to "no-cache" mode: every lookup falls through to the upstream HTTP path. Counters under `atlas_monsters_data_cache_redis_errors_total` make the degradation visible. No requests fail.

## 9. Open Questions

- **Negative TTL default value** — proposed 30 s based on heuristic ("short enough that a fixed bad id recovers within a minute, long enough to absorb a retry storm"). No production data to validate. Revisit if metrics show pathological behavior.
- **Cache value representation** — store the parsed `Model` (proposed) vs. the raw `RestModel`. Storing `Model` skips re-`Extract`-ing on every hit; storing `RestModel` makes future caching of arbitrary `requests.Provider[Rest, Model]` calls easier when we generalize. Decide at design time. Default: `Model`.
- **Library placement** — `libs/atlas-cache` (proposed, new module) vs. extending `libs/atlas-redis` with an in-memory variant vs. adding to `libs/atlas-rest/requests` as a "cached provider" wrapper. The "cached requests provider" framing is appealing for follow-ups but coupling the cache to the REST client may be premature. Default: standalone `libs/atlas-cache`.
- **Background sweeper** — v1 is purely lazy expiration. If a tenant churns through monster ids and never reads them again, expired entries linger in memory until process restart. Not expected in practice (id space is bounded), but worth confirming with the design.
- **Metric label cardinality** — labeling by `tenant` is fine for the small number of tenants today; if tenant count grows, the label may need to drop or be sampled. Document in README; don't pre-optimize.

## 10. Acceptance Criteria

A reviewer can mark this task done when ALL of the following are true:

- [ ] No new shared library is introduced. `libs/atlas-redis.TenantRegistry` is reused for positive and negative caches.
- [ ] `services/atlas-monsters/atlas.com/monsters/monster/information/processor.go` `GetById` is a read-through cache backed by two `libs/atlas-redis.TenantRegistry` instances (positive + negative), scoped per tenant via `tenant.MustFromContext`.
- [ ] On a cache hit, no HTTP request is issued (verified via test that asserts the underlying transport is not called).
- [ ] On a 404 from `atlas-data`, a negative entry is recorded; subsequent calls within the negative TTL return the same not-found error without issuing an HTTP request.
- [ ] On a 5xx / network / parse error, no negative entry is recorded; the next call still issues an HTTP request.
- [ ] Two tenants requesting the same monster id with different upstream answers each see their own answer (tenant-isolation test).
- [ ] Setting `MONSTER_DATA_CACHE_ENABLED=false` causes `GetById` to behave identically to the pre-task implementation (kill-switch test).
- [ ] All four metrics in §4.7 are emitted and labeled correctly.
- [ ] `go build ./...` succeeds in `services/atlas-monsters/atlas.com/monsters`.
- [ ] `go test -race ./...` is clean in `services/atlas-monsters/atlas.com/monsters` (no new library to test). Existing `atlas-monsters` tests pass without modification.
- [ ] Manual verification against a running stack (or recorded log capture) shows that for a 30-second window with sustained monster activity, the rate of `GET /api/data/monsters/{id}` calls leaving `atlas-monsters` drops by **at least 95%** compared to the pre-task baseline (e.g. from ~24 rps in the recorded log to <1.2 rps after warm-up).
- [ ] PRD updated to status `Implemented` and `Date implemented` on merge.
