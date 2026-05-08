# atlas-monsters Data TTL Cache — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-08
---

## 1. Overview

`atlas-monsters` resolves static monster reference data (name, HP/MP, EXP, level, weapon/magic attack & defense, animation timings, attacks, skills, resistances, revives, banish, self-destruct, cool-damage) by calling `GET /api/data/monsters/{id}` on `atlas-data` for every code path that needs it: aggro, picker, recovery, status, basic-attack, drop generation, etc. The lookup happens through `monster/information.GetById`, which is currently a thin wrapper that issues an HTTP request on every invocation with no caching.

A short capture from `tmp/atlas-monster-dbg-log` (35 s window, 2026-05-08 07:17:06–07:17:41) shows **850 GET requests** against `atlas-data` (~24 rps), with each unique monster id polled **23–26 times** in that window. The data is WZ-derived and effectively immutable between `atlas-data` deployments. Re-fetching it dozens of times per minute is wasted upstream load on `atlas-data`, wasted network and CPU in `atlas-monsters`, and unnecessary latency on hot per-tick code paths (aggro, picker, recovery).

This task introduces a small **in-process TTL cache** in front of `information.GetById`. The cache is tenant-keyed, has a configurable positive-entry TTL (default **5 minutes**) and negative-entry TTL (default **30 seconds**). Cache invalidation on `atlas-data` redeploy is explicitly out of scope — pod restart is the supported reset mechanism for now, and a future task will add an event-driven flush.

The cache primitive is built as a new shared library at `libs/atlas-cache` so other services can adopt the same pattern in follow-up tasks (`atlas-channel`, `atlas-monster-death`, and other `atlas-data`-consuming services). `libs/atlas-redis` already provides a Redis-backed `TTLRegistry`, but it is the wrong tool for this hot-path lookup: every cache hit becomes a Redis network roundtrip, which defeats most of the point on read-heavy reference data, and cross-pod consistency is explicitly not required for this use case.

## 2. Goals

Primary goals:

- Eliminate the bulk of repeat `GET /api/data/monsters/{id}` calls from `atlas-monsters` to `atlas-data`. Target: **>95% cache hit rate** in steady state, measured via metrics over a 5-minute window once the cache has warmed.
- Cache positive responses with a **configurable TTL (default 5 m)**.
- Cache negative responses (errors / not-found) with a **configurable shorter TTL (default 30 s)** to absorb retry storms on bad ids.
- Make the cache **tenant-scoped** so two tenants with different `atlas-data` configurations never see each other's entries.
- Make the cache **opt-in via a single call site change** — wrap `information.GetById` only; no callers change.
- Provide a **reusable library** (`libs/atlas-cache`) so the same pattern can be applied to other reference-data lookups in follow-up tasks without copy-paste.
- Provide observability: hit/miss/negative-hit counters and one gauge for cache size, scoped by tenant.

Non-goals:

- No cross-pod / cross-process cache coherence. Each pod has its own cache. Restarting the pod is the supported invalidation mechanism.
- No event-driven invalidation (Kafka or otherwise) on `atlas-data` redeploys. Tracked as a future task.
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

### 4.1 New library: `libs/atlas-cache`

- New Go module at `libs/atlas-cache/` (mirrors layout of `libs/atlas-redis/`, `libs/atlas-tenant/`, etc.).
- Module name: `github.com/Chronicle20/atlas/libs/atlas-cache` (or whatever the existing convention dictates — confirm at design time).
- Exposes a generic, type-parameterized in-process TTL cache.
- Public API (final names confirmed at design time):
  - `type Cache[K comparable, V any] interface { Get(K) (V, bool); Put(K, V); PutNegative(K); IsNegative(K) bool; Delete(K); Len() int }` (or similar; the exact shape is a design decision, but the cache must support distinct positive/negative TTLs).
  - `type Config struct { TTL time.Duration; NegativeTTL time.Duration; Now func() time.Time /* optional for tests */ }`
  - `func New[K comparable, V any](cfg Config) Cache[K, V]`
- Concurrency: safe for many readers and writers. No external locking required by callers.
- Expiration: lazy on read (entry checked against `time.Now()` and, if expired, treated as a miss and removed). No background goroutine in v1. A background sweeper is a non-goal but the API should not preclude one being added later.
- Negative caching: a key with a negative entry returns `(_, false)` from `Get` but `IsNegative(key) == true` until the negative TTL expires. Callers use this to avoid re-issuing a known-bad upstream lookup.
- No serialization, no eviction by size in v1 (entry count is bounded by the upstream id space — at most a few thousand monster ids per tenant — so unbounded growth is acceptable; revisit if metrics show outliers).
- `time.Now` is injectable via `Config.Now` for deterministic tests.

### 4.2 Tenant-scoped wrapper in `atlas-monsters`

- Add a tenant-scoped registry inside `services/atlas-monsters/atlas.com/monsters/monster/information/` (or a new sub-package, e.g. `information/cache`) that maps `tenant.Id` → `Cache[uint32, Model]` from `libs/atlas-cache`.
- Lazy-initialize per-tenant cache instances on first use using the standard `sync.Once` + `sync.RWMutex` pattern.
- Keys: `uint32` monster id (positive cache and negative cache share the same key space; one entry per id per tenant).
- Cache values are the existing `information.Model` (the parsed domain model — not the `RestModel`). This avoids re-running the `Extract` step on every cache hit.

### 4.3 Read-through wrapper around `information.GetById`

- The existing `information.GetById(l)(ctx)(monsterId) (Model, error)` call signature is preserved. Call sites do not change.
- Internally, `GetById` performs:
  1. Resolve `tenant.Model` from `ctx` via `tenant.MustFromContext`.
  2. Resolve the per-tenant cache instance.
  3. If the key has a positive entry, return it.
  4. If the key has an unexpired negative entry, return the original-style "not found" error (preserve current error semantics — see §4.6).
  5. Otherwise, issue the existing HTTP request via `requests.Provider[RestModel, Model]`.
  6. On success: `Put(monsterId, model)` and return.
  7. On failure: classify the error (see §4.6) and either `PutNegative(monsterId)` or do not cache (transient errors). Return the error to the caller unchanged.
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

- Cache key includes the tenant identity. Implementation can be either (a) one `Cache` instance per tenant in a `tenant.Id`-keyed registry, or (b) a single composite-keyed cache. Either is acceptable; (a) matches the codebase's existing `sync.Once` registry idiom and is preferred unless design-task surfaces a reason to do (b).
- A tenant entry is created lazily on first lookup for that tenant. There is no cleanup of per-tenant cache instances on tenant disable in v1 (memory cost is small; revisit if metrics show otherwise).

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
- `atlas_monsters_data_cache_errors_total{tenant, classification}` — counter. `classification` is `not_found` (counted as miss + negative-put), `transient` (counted as miss, not cached), or `parse` (counted as miss, not cached).
- `atlas_monsters_data_cache_size{tenant, kind}` — gauge. `kind` is `positive` or `negative`.
- `atlas_monsters_data_cache_evictions_total{tenant, reason}` — counter. `reason` is `expired_positive` or `expired_negative` (incremented when a lazy expiration purges an entry).

Existing `requests` debug logs are unchanged. The cache layer SHOULD NOT log on every hit (would obliterate signal-to-noise); it MAY log at debug level on negative-cache puts and on the first miss after process start per tenant.

### 4.8 Testing

- Library `libs/atlas-cache` ships with unit tests covering: get-miss, put-then-get, expiration (using injected `Now`), negative-put-then-get, negative expiration, concurrent access (race-detector clean), and `Delete`.
- Service-side wrapper ships with unit tests covering: tenant isolation (two tenants don't share entries), positive cache hit avoids HTTP (using a fake `requests.Provider` or HTTP server), negative cache hit avoids HTTP, non-cacheable errors are not cached, kill-switch disables caching, and the error returned on a cache hit matches the error from a non-cached miss for the same input.
- Existing `atlas-monsters` tests must continue to pass without modification. If they currently rely on `information.GetById` issuing real HTTP calls, the wrapper must be transparent enough that they still work.
- `go test -race ./...` in both modules must be clean.

## 5. API Surface

No changes to any HTTP, JSON:API, or Kafka surface.

The internal Go API of `monster/information` is unchanged. `information.GetById(l)(ctx)(monsterId) (Model, error)` keeps the same signature, semantics, and error types.

The new library `libs/atlas-cache` is purely internal Go API. Final exact shape decided at design time; see §4.1 for the proposed shape.

## 6. Data Model

No persistent data changes. No database migrations. No new Kafka topics.

In-memory data:

- `libs/atlas-cache` stores entries as `(value V, expiresAt time.Time, negative bool)` tuples in a `map[K]entry` guarded by a `sync.RWMutex` (or `sync.Map` if benchmarks favor it — design-time decision).
- Per-tenant registry in `atlas-monsters` stores `map[tenant.Id]*Cache[uint32, Model]`.

## 7. Service Impact

### 7.1 `services/atlas-monsters`

- New dependency on `libs/atlas-cache` in `go.mod`.
- `monster/information/processor.go`: `GetById` changes from a one-line passthrough to a read-through cache. Behavior on the success path is identical; behavior on the error path adds short-lived negative caching for not-found.
- `monster/information/`: add cache registry file (e.g. `cache.go` or new sub-package `cache/`).
- New env vars wired up wherever the service currently reads tunables.
- New metrics registered on the existing Prometheus registry.

### 7.2 `libs/atlas-cache` (new)

- New module with its own `go.mod`, `go.sum`, `README.md`.
- Generic in-process TTL cache implementation, tests, and a small README describing positive/negative TTL semantics and intended use case.

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

- A cache hit must be O(1) with no I/O. Target: sub-microsecond on commodity hardware (validated via a Go benchmark in `libs/atlas-cache`).
- A cache miss adds at most one map lookup + one `time.Now()` call before falling through to the existing HTTP path. Overhead must be statistically indistinguishable from the current path under benchmark.
- Memory: at most a few thousand entries per tenant × small struct (few hundred bytes) → well under 10 MB per tenant in the worst case. Acceptable.

### 8.2 Correctness

- No stale data older than `MONSTER_DATA_CACHE_TTL` may be returned. (Bound: caller observes data that was upstream-current within the last TTL window.)
- No data may leak between tenants under any code path, including concurrent first-touch by two tenants.
- Race-detector clean: `go test -race ./...` in both modules.

### 8.3 Observability

- Metrics §4.7 are present and labeled.
- One operator-facing dashboard panel can answer "what's the hit rate?" from `hits_total / (hits_total + misses_total)` per tenant. Dashboard authoring is out of scope; metric availability is in scope.

### 8.4 Security & multi-tenancy

- Tenant isolation enforced at the cache key level. No code path in `atlas-monsters` may read a cached entry for a tenant other than the one in `ctx`.
- No new secrets, no new external network egress.

### 8.5 Operability

- Master kill-switch (`MONSTER_DATA_CACHE_ENABLED=false`) disables the cache without code change or redeploy of the dependency. Restart of the pod after env-var change is acceptable.
- Cache state is per-pod and lost on restart. This is the intended invalidation mechanism for `atlas-data` redeploys.
- No CLI / admin endpoint to flush. Documented in the §1 Overview and the `libs/atlas-cache` README so it isn't surprising.

## 9. Open Questions

- **Negative TTL default value** — proposed 30 s based on heuristic ("short enough that a fixed bad id recovers within a minute, long enough to absorb a retry storm"). No production data to validate. Revisit if metrics show pathological behavior.
- **Cache value representation** — store the parsed `Model` (proposed) vs. the raw `RestModel`. Storing `Model` skips re-`Extract`-ing on every hit; storing `RestModel` makes future caching of arbitrary `requests.Provider[Rest, Model]` calls easier when we generalize. Decide at design time. Default: `Model`.
- **Library placement** — `libs/atlas-cache` (proposed, new module) vs. extending `libs/atlas-redis` with an in-memory variant vs. adding to `libs/atlas-rest/requests` as a "cached provider" wrapper. The "cached requests provider" framing is appealing for follow-ups but coupling the cache to the REST client may be premature. Default: standalone `libs/atlas-cache`.
- **Background sweeper** — v1 is purely lazy expiration. If a tenant churns through monster ids and never reads them again, expired entries linger in memory until process restart. Not expected in practice (id space is bounded), but worth confirming with the design.
- **Metric label cardinality** — labeling by `tenant` is fine for the small number of tenants today; if tenant count grows, the label may need to drop or be sampled. Document in README; don't pre-optimize.

## 10. Acceptance Criteria

A reviewer can mark this task done when ALL of the following are true:

- [ ] New module `libs/atlas-cache` exists with: a generic TTL cache implementation supporting distinct positive/negative TTLs, injectable `Now`, lazy expiration, race-clean concurrency, and unit tests covering all paths in §4.8.
- [ ] `services/atlas-monsters/atlas.com/monsters/monster/information/processor.go` `GetById` is a read-through cache backed by `libs/atlas-cache`, scoped per tenant via `tenant.MustFromContext`.
- [ ] On a cache hit, no HTTP request is issued (verified via test that asserts the underlying transport is not called).
- [ ] On a 404 from `atlas-data`, a negative entry is recorded; subsequent calls within the negative TTL return the same not-found error without issuing an HTTP request.
- [ ] On a 5xx / network / parse error, no negative entry is recorded; the next call still issues an HTTP request.
- [ ] Two tenants requesting the same monster id with different upstream answers each see their own answer (tenant-isolation test).
- [ ] Setting `MONSTER_DATA_CACHE_ENABLED=false` causes `GetById` to behave identically to the pre-task implementation (kill-switch test).
- [ ] All five metrics in §4.7 are emitted and labeled correctly.
- [ ] `go build ./...` succeeds in both `services/atlas-monsters/atlas.com/monsters` and `libs/atlas-cache`.
- [ ] `go test -race ./...` is clean in both modules. Existing `atlas-monsters` tests pass without modification.
- [ ] Manual verification against a running stack (or recorded log capture) shows that for a 30-second window with sustained monster activity, the rate of `GET /api/data/monsters/{id}` calls leaving `atlas-monsters` drops by **at least 95%** compared to the pre-task baseline (e.g. from ~24 rps in the recorded log to <1.2 rps after warm-up).
- [ ] PRD updated to status `Implemented` and `Date implemented` on merge.
