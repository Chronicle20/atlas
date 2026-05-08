# atlas-monsters Data TTL Cache — Design (v2, Redis-backed)

Version: v2
Status: Draft
Created: 2026-05-08
Companion to: [`prd.md`](./prd.md)
Supersedes: v1 in-process design (rejected — see §3)

---

## 1. Goals of This Document

The PRD pins **what** and **why**. This design pins **how**, after the v1 design's premise (a brand-new `libs/atlas-cache` in-process module) was rejected during phase 4. The current Atlas codebase already ships `libs/atlas-redis` with `Registry`, `TenantRegistry`, and `TenantRegistry.PutWithTTL` (using native Redis TTL). This design uses those primitives instead.

What this document settles:

- The exact way `services/atlas-monsters/atlas.com/monsters/monster/information/GetById` becomes read-through against Redis.
- Why two `redis.TenantRegistry` instances (positive + negative) is the cleanest fit, and what other shapes were considered.
- How error classification, native Redis TTL, graceful Redis-failure degradation, metrics, and tenant scoping plug into the existing service idioms.
- Which PRD lines need touch-up before plan phase, since v2 changes the user-visible NFR profile (latency goes from sub-microsecond to a Redis roundtrip; the cache now survives pod restarts).

Out of scope for this document: file-by-file commit ordering and concrete TDD steps. Those live in `plan.md`.

---

## 2. Architecture Overview

```
                ┌────────────────────────────────────────────────────────┐
                │ atlas-monsters call sites                              │
                │   processor.go, aggro_task.go, recovery_task.go        │
                │   ─ unchanged callers; same GetById signature ─        │
                └─────────────────────────┬──────────────────────────────┘
                                          │ information.GetById(l)(ctx)(id)
                                          ▼
                ┌────────────────────────────────────────────────────────┐
                │ services/.../monster/information/                      │
                │                                                        │
                │   processor.go ── thin read-through wrapper            │
                │   cache.go     ── DataCache singleton:                 │
                │                    • posReg *redis.TenantRegistry      │
                │                      (namespace "monsters:cache:data") │
                │                      TTL = MONSTER_DATA_CACHE_TTL      │
                │                    • negReg *redis.TenantRegistry      │
                │                      (namespace                        │
                │                       "monsters:cache:data:not_found") │
                │                      TTL = MONSTER_DATA_CACHE_NEG_TTL  │
                │                    • env loader, error classifier,     │
                │                      metrics emit, kill-switch         │
                │   metrics.go   ── promauto counter/gauge declarations  │
                │                                                        │
                │   model.go, requests.go, rest.go (unchanged)           │
                └─────────────────────────┬──────────────────────────────┘
                                          │ redis-go client (already wired in main.go)
                                          ▼
                ┌────────────────────────────────────────────────────────┐
                │ Redis  (already a runtime dep of atlas-monsters)       │
                │                                                        │
                │   atlas:monsters:cache:data:<tenantKey>:<id>           │
                │     value: JSON(Model)            TTL: 5m  (default)   │
                │   atlas:monsters:cache:data:not_found:<tenantKey>:<id> │
                │     value: "{}"                   TTL: 30s (default)   │
                └────────────────────────────────────────────────────────┘
```

The two Redis namespaces are tenant-scoped via the existing `tenantEntityKey` helper in `libs/atlas-redis/keys.go:27`, which yields `atlas:<namespace>:<tenantId>:<region>:<major>.<minor>:<entity>`. No new library code; no new shared types; no new runtime dependency.

---

## 3. Why Redis, Not a New In-Process Cache (v1 → v2 Pivot)

The v1 design proposed a brand-new `libs/atlas-cache` module: a generic in-process TTL cache (`map[K]entry` + `RWMutex`, lazy expiration, `OnEviction` callback). After bootstrapping that module (commit `957801571`, since reverted), the user pointed out that `libs/atlas-redis` already provides:

- `TenantRegistry.PutWithTTL(ctx, t, key, value, ttl)` using **native Redis EX TTL** — no app-side sweep, no goroutines, no eviction bookkeeping (`libs/atlas-redis/tenant_registry.go:116-123`).
- `TenantRegistry.Get(ctx, t, key)` returning `(V, error)` with `ErrNotFound` as the standard miss shape (`tenant_registry.go:43-55`).
- Tenant-scoped key namespacing for free via `tenantEntityKey` (`keys.go:27`).
- A wired `*goredis.Client` already constructed in `services/atlas-monsters/atlas.com/monsters/main.go:48` (`rc := atlas.Connect(l)`).
- An established `Init*Registry(rc)` idiom for plumbing the client into per-package singletons (`monster/cooldown.go:21`, `monster/registry.go:265`).

**Cost/benefit vs. v1's in-process cache:**

| Dimension | v1 (in-proc) | v2 (Redis) |
|---|---|---|
| Hit latency | ~100 ns | ~0.2–1 ms (Redis roundtrip in-cluster) |
| Cross-pod hit rate | per-pod cold-start every restart | shared across all atlas-monsters pods |
| Restart resilience | cache lost on pod restart | survives pod restarts |
| Memory footprint | grows in atlas-monsters heap | stays in Redis (already provisioned) |
| New module surface | `libs/atlas-cache` + ~300 LOC + tests + bench | zero new module; ~200 LOC service-side |
| Negative-cache mechanism | second entry kind in same map | second `TenantRegistry` instance, distinct namespace |
| Eviction | lazy, app-side, with `OnEviction` callback | native Redis TTL — Redis handles it |
| Failure mode | none (it's a map) | Redis unavailable ⇒ falls through to upstream |

The PRD's primary goal — "≥95% reduction in `GET /api/data/monsters/{id}` calls" — is achieved trivially by either approach. The latency delta (~1 ms vs. ~100 ns per hit) is irrelevant for monster-information lookups, which are not on a tight inner loop and which already pay multi-ms HTTP roundtrips today. The cross-pod and restart-resilience gains, plus avoiding a new module, decisively favor Redis.

**The only thing v1 had that v2 doesn't:** the option to operate without Redis. atlas-monsters already requires Redis at runtime for cooldowns, drops, and the monster registry, so this is not a new dependency.

---

## 4. PRD Divergences (Plan Must Reconcile)

The PRD as written assumes the v1 in-process model. v2 changes a few user-visible properties. The plan phase MUST update the PRD to match before implementation, OR explicitly call out which PRD bullets are now stale. The deltas:

| PRD line | v1 stance | v2 stance |
|---|---|---|
| §1 Overview, last paragraph: "built as a new shared library at `libs/atlas-cache`" — and "[atlas-redis] is the wrong tool for this hot-path lookup: every cache hit becomes a Redis network roundtrip, which defeats most of the point" | rejects Redis | **embraces Redis.** The "every hit is a roundtrip" objection is reframed: a sub-ms roundtrip is fine for this code path, and the cross-pod and restart-resilience benefits are large. Update §1 accordingly. |
| §4.1 New library `libs/atlas-cache` | required | **removed.** No new library. Replace §4.1 with "Reuse `libs/atlas-redis.TenantRegistry` for positive and negative caches." |
| §4.5 Multi-tenancy: "one `Cache` instance per tenant in a `tenant.Id`-keyed registry, or … a single composite-keyed cache" | per-tenant map preferred | **single-instance per registry, tenant scoped at the key level via `TenantRegistry.entityKey`.** Already the redis idiom. Update §4.5. |
| §4.7 Observability: `cache_size` gauge | sampled via `Cache.Len()` | **dropped or repurposed.** Counting Redis keys requires a `SCAN` per sample, which is operationally expensive. Replace with operator-side Redis introspection (`redis-cli DBSIZE`, `INFO keyspace`) or drop. Final call: drop the gauge in v2; keep the four counters (hits, misses, errors, evictions). Update §4.7. Note: `evictions_total` becomes "TTL expirations observed via `Get → ErrNotFound` after a previous `Put`" — i.e., we cannot directly observe Redis-side TTL expirations, so the metric loses some fidelity. Acceptable. |
| §8.1 Performance: "sub-microsecond on commodity hardware" | required | **reframed.** New target: p99 cache-hit latency ≤ 5 ms in-cluster (typical Redis GET ~0.2–1 ms; budget allows for tail). Update §8.1. |
| §8.5 Operability: "Cache state is per-pod and lost on restart. This is the intended invalidation mechanism" | true | **false in v2.** The cache survives pod restarts. The new invalidation mechanisms are: TTL expiry (5 m default); manual `redis-cli` DEL or FLUSHDB for the specific namespace; future event-driven flush still tracked as follow-up. Update §8.5. |
| §10 Acceptance, line "New module `libs/atlas-cache` exists with: a generic TTL cache implementation…" | required | **dropped entirely.** Replace with: "Reuses `libs/atlas-redis.TenantRegistry`; no new library introduced." |
| §10 Acceptance, race-detector clean | required | **still required**, scoped to atlas-monsters. (No new lib to test.) |

I will state these as a discrete edit list in the plan; the implementer's first task is "PRD revision PR" before code.

---

## 5. Service-Side Wrapper Design

### 5.1 Files Touched

| File | Change |
|---|---|
| `monster/information/processor.go` | `GetById` becomes read-through. Signature preserved verbatim. |
| `monster/information/cache.go` | **NEW** — `DataCache` singleton, `Init`, env loader, error classifier, metric emit, kill-switch. |
| `monster/information/metrics.go` | **NEW** — promauto counter/gauge declarations. |
| `monster/information/cache_test.go` | **NEW** — wrapper-level unit tests using a miniredis backend. |
| `services/.../monsters/main.go` | One line: `information.InitDataCache(rc)` after the existing `monster.Init*Registry(rc)` block at `main.go:49`. |
| `services/.../monsters/go.mod` | Add `github.com/prometheus/client_golang v1.23.2` and `github.com/alicebob/miniredis/v2` (test-only). No `atlas-cache` lib. |
| `services/.../monsters/go.sum` | Regenerated by `go mod tidy`. |

**Untouched:** `model.go`, `requests.go`, `rest.go`, `rest_test.go`, `builder.go`, every caller of `GetById`.

### 5.2 The `DataCache` Singleton

```go
package information

import (
    "context"
    "errors"
    "fmt"
    "os"
    "sync"
    "time"

    redislib "github.com/Chronicle20/atlas/libs/atlas-redis"
    "github.com/Chronicle20/atlas/libs/atlas-rest/requests"
    tenantlib "github.com/Chronicle20/atlas/libs/atlas-tenant"
    goredis "github.com/redis/go-redis/v9"
    "github.com/sirupsen/logrus"
)

const (
    posNamespace = "monsters:cache:data"
    negNamespace = "monsters:cache:data:not_found"
)

type DataCache struct {
    enabled     bool
    posTTL      time.Duration
    negTTL      time.Duration
    posReg      *redislib.TenantRegistry[uint32, Model]
    negReg      *redislib.TenantRegistry[uint32, struct{}]
}

var (
    cacheOnce sync.Once
    cache     *DataCache
)

// InitDataCache wires a singleton DataCache. Idempotent; safe to call from
// main.go alongside other Init*Registry hooks.
func InitDataCache(rc *goredis.Client) {
    cacheOnce.Do(func() {
        cfg := loadConfig()
        cache = &DataCache{
            enabled: cfg.enabled,
            posTTL:  cfg.posTTL,
            negTTL:  cfg.negTTL,
            posReg: redislib.NewTenantRegistry[uint32, Model](
                rc, posNamespace, uint32KeyFn),
            negReg: redislib.NewTenantRegistry[uint32, struct{}](
                rc, negNamespace, uint32KeyFn),
        }
    })
}

func uint32KeyFn(id uint32) string {
    return strconv.FormatUint(uint64(id), 10)
}
```

**Why two `TenantRegistry` instances:** `TenantRegistry` already supports tenant-scoped keys, JSON marshal/unmarshal, and `PutWithTTL` with native Redis EX. We only need distinct *namespaces* and distinct *TTLs* to model "positive" vs. "negative" entries. Two instances costs nothing — they share the same `*goredis.Client` — and gives us:

- Independent `Get`/`Put`/`Remove` semantics per kind.
- Independent TTL configuration per kind.
- Trivial `redis-cli` introspection: an operator can `KEYS atlas:monsters:cache:data:*` to count positives, `KEYS atlas:monsters:cache:data:not_found:*` for negatives.
- No need for a typed envelope (`struct { Value V; Negative bool }`) inside one registry, which would require a custom marshal hook.

**Why `struct{}` for negatives:** the negative entry carries no payload — its mere existence is the signal. JSON-marshalling `struct{}` yields `{}` (3 bytes), which is the smallest stable Redis value the registry will accept.

### 5.3 Read-Through `GetById`

```go
func GetById(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) (Model, error) {
    return func(ctx context.Context) func(monsterId uint32) (Model, error) {
        return func(monsterId uint32) (Model, error) {
            t := tenantlib.MustFromContext(ctx)
            tid := t.Id().String()

            if cache == nil || !cache.enabled {
                return upstreamFetch(l, ctx, monsterId) // kill-switch / unwired
            }

            // Positive lookup.
            v, err := cache.posReg.Get(ctx, t, monsterId)
            switch {
            case err == nil:
                hitsTotal.WithLabelValues(tid, "positive").Inc()
                return v, nil
            case errors.Is(err, redislib.ErrNotFound):
                // Fall through to negative lookup.
            default:
                // Redis hard failure — log, count, fall through to upstream.
                redisErrorsTotal.WithLabelValues(tid, "get_positive").Inc()
                l.WithError(err).Debug("data cache positive lookup failed; falling through")
            }

            // Negative lookup (only if NegativeTTL > 0).
            if cache.negTTL > 0 {
                _, nerr := cache.negReg.Get(ctx, t, monsterId)
                switch {
                case nerr == nil:
                    hitsTotal.WithLabelValues(tid, "negative").Inc()
                    return Model{}, notFoundError(monsterId)
                case errors.Is(nerr, redislib.ErrNotFound):
                    // True miss — fall through to upstream.
                default:
                    redisErrorsTotal.WithLabelValues(tid, "get_negative").Inc()
                    l.WithError(nerr).Debug("data cache negative lookup failed; falling through")
                }
            }

            missesTotal.WithLabelValues(tid).Inc()
            v, ferr := upstreamFetch(l, ctx, monsterId)
            if ferr == nil {
                if perr := cache.posReg.PutWithTTL(ctx, t, monsterId, v, cache.posTTL); perr != nil {
                    redisErrorsTotal.WithLabelValues(tid, "put_positive").Inc()
                    l.WithError(perr).Debug("data cache positive put failed; serving fetched value uncached")
                }
                return v, nil
            }
            switch classifyError(ferr) {
            case errKindNotFound:
                errorsTotal.WithLabelValues(tid, "not_found").Inc()
                if cache.negTTL > 0 {
                    if perr := cache.negReg.PutWithTTL(ctx, t, monsterId, struct{}{}, cache.negTTL); perr != nil {
                        redisErrorsTotal.WithLabelValues(tid, "put_negative").Inc()
                        l.WithError(perr).Debug("data cache negative put failed; not caching")
                    }
                }
            default:
                errorsTotal.WithLabelValues(tid, "transient").Inc()
            }
            return Model{}, ferr
        }
    }
}

func upstreamFetch(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (Model, error) {
    return requests.Provider[RestModel, Model](l, ctx)(requestById(monsterId), Extract)()
}

// upstreamFn is a test-overridable indirection. Production points at upstreamFetch.
var upstreamFn = upstreamFetch
```

**Why `upstreamFn` indirection:** wrapper unit tests inject a fake fetcher (`upstreamFn = func(...) {...}`) without standing up a real `httptest.Server`, mirroring the function-typed-field pattern at `monster/processor.go:65` (existing test idiom).

**Why three error buckets and not four (no `parse` kind):** v1 design split parse errors out separately. v2 collapses parse → transient because `requests.Provider` already wraps unmarshal failures in a transport-level error and never surfaces a typed `*json.SyntaxError` to callers — distinguishing them adds metric noise without an action.

### 5.4 Error Classification

```go
type errKind int
const (
    errKindTransient errKind = iota
    errKindNotFound
)

// classifyError decides whether to record a negative cache entry.
// requests.ErrNotFound is the sentinel libs/atlas-rest/requests returns
// on HTTP 404 (libs/atlas-rest/requests/get.go:14-15). Everything else —
// 5xx, 400 (ErrBadRequest), network errors, parse errors, retry exhaustion —
// is transient and never cached.
func classifyError(err error) errKind {
    if errors.Is(err, requests.ErrNotFound) {
        return errKindNotFound
    }
    return errKindTransient
}

// notFoundError synthesizes the same error shape callers see from a live miss
// (errors.Is(err, requests.ErrNotFound) holds) so negative-cache hits are
// indistinguishable from upstream 404s at the call site.
func notFoundError(monsterId uint32) error {
    return fmt.Errorf("monster %d not found: %w", monsterId, requests.ErrNotFound)
}
```

This is the same correction noted in `context.md` §3.2: `requests.ErrNotFound` is a sentinel `var`, not a typed error struct. v1's design.md proposed `requests.HTTPError` (which doesn't exist); v2 keeps the sentinel-based check. No change to that conclusion.

### 5.5 Configuration Loader

```go
type cacheConfig struct {
    enabled bool
    posTTL  time.Duration
    negTTL  time.Duration
}

func loadConfig() cacheConfig {
    return cacheConfig{
        enabled: parseBoolEnv("MONSTER_DATA_CACHE_ENABLED", true),
        posTTL:  parseDurationEnv("MONSTER_DATA_CACHE_TTL", 5*time.Minute, 1*time.Second, 24*time.Hour),
        negTTL:  parseDurationEnv("MONSTER_DATA_CACHE_NEGATIVE_TTL", 30*time.Second, 0, 5*time.Minute),
    }
}
```

`parseBoolEnv` and `parseDurationEnv` log a warning at WARN level and fall back to default on bad input — same shape as v1 §4.6. Lazy-loaded once via `cacheOnce` inside `InitDataCache`. Tests override env vars before invoking `InitDataCache` against a miniredis client.

### 5.6 Metrics Wiring

```go
package information

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    hitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_cache_hits_total",
        Help: "Cache hits for monster information lookups, by tenant and entry kind.",
    }, []string{"tenant", "kind"})

    missesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_cache_misses_total",
        Help: "Cache misses (upstream HTTP issued) for monster information lookups, by tenant.",
    }, []string{"tenant"})

    errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_cache_errors_total",
        Help: "Upstream errors observed during monster information lookups, by tenant and classification.",
    }, []string{"tenant", "classification"})

    redisErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_cache_redis_errors_total",
        Help: "Redis-side errors during monster data cache operations, by tenant and operation. " +
              "Each increment indicates a graceful fallthrough to upstream (or a discarded cache write).",
    }, []string{"tenant", "operation"})
)
```

Four counters; **no gauge**. The v1 `cache_size` gauge required `Cache.Len()`, which is cheap on an in-proc map but means a Redis `SCAN` here. Operator visibility into Redis key count is a Redis-side concern (`redis-cli DBSIZE`, `INFO keyspace`), not an app concern. Drop it. The four counters are sufficient to compute hit rate, error rate, and degraded-mode rate.

`cardinality`: `tenant` label kept (≤5 tenants in production); `operation` and `classification` are bounded enums (≤4 values each). Cardinality budget: ≤ 4 × 5 × 4 = 80 series. Trivial.

---

## 6. Decisions on PRD §9 Open Questions

| Question | v1 decision | v2 decision | Rationale |
|---|---|---|---|
| Negative TTL default | 30 s | **30 s (unchanged)** | Heuristic stands; revisit if metrics surface pathological behavior. |
| Cache value: `Model` vs `RestModel` | `Model` | **`Model` (unchanged)** | `Model` is what callers want; storing `RestModel` would require re-`Extract` on every hit. JSON-marshalling `Model` is cheap. |
| Library placement | new `libs/atlas-cache` | **no new library** | Reuse `libs/atlas-redis.TenantRegistry`. The "library reuse for follow-up tasks" PRD goal is preserved: any future service caching `data/<x>/{id}` gets the same pattern by instantiating its own pair of `TenantRegistry`s — identical idiom, no shared code needed. |
| Background sweeper | none (lazy in v1) | **none — Redis native TTL handles it** | Even better than lazy app-side: Redis evicts on its own; we never carry expired entries. |
| Metric label cardinality | keep `tenant` | **keep `tenant` (unchanged)** | ≤5 tenants × bounded enum dims. |

---

## 7. Concurrency, Correctness, and Failure Modes

### 7.1 Tenant Isolation

All keys go through `redislib.TenantRegistry.entityKey(t, key)` ⇒ `tenantEntityKey(namespace, t, keyFn(key))` ⇒ `atlas:<namespace>:<tenantUUID>:<region>:<major>.<minor>:<id>`. Two tenants with the same monster id occupy distinct Redis keys. No cross-tenant read or write is possible without explicit construction of a different `tenant.Model`. Verified by a wrapper unit test (`TestGetById_TenantIsolation`) that runs two tenants against the same miniredis instance and asserts each sees its own answer.

### 7.2 Concurrent Misses for the Same Key

Two goroutines both miss for `(t, id)`:

1. Both fetch upstream.
2. Both `PutWithTTL` to Redis. Last write wins. Both succeed independently.
3. Both return their own fetched value.

This is one wasted upstream call. PRD non-goal: no single-flight in v1. Same posture as v1 design §6.1.

### 7.3 Concurrent Positive + Negative Put for the Same Key

A monster id transitions from existing to deleted upstream within a TTL window:

1. Goroutine A reads the positive cache entry and returns the old value (within TTL — acceptable per PRD §8.2 staleness bound).
2. Eventually positive TTL expires; next call sees Redis miss.
3. Upstream returns 404; we `PutWithTTL` to negative registry.
4. Subsequent calls hit the negative entry until 30 s elapse.

If the monster is *recreated* upstream during the negative window, callers see 404 for up to 30 s after recreation — bounded staleness, acceptable per PRD §8.2.

If a `PutWithTTL` to negative succeeds while a stale positive entry still exists (TTL-skewed transition), `Get` will hit positive first and return the stale value. The two registries are independent, and we do not perform cross-registry consistency. Acceptable: positive TTL is short (5 m default) and bounds the staleness.

### 7.4 Redis Unavailable

Every Redis-touching operation has a fallthrough path:

| Operation | On Redis error | Caller-observable behavior |
|---|---|---|
| `posReg.Get` | log, increment `redis_errors_total`, fall through to negReg | identical to a miss; one upstream HTTP call |
| `negReg.Get` | log, increment, fall through to upstream | identical to a miss |
| `posReg.PutWithTTL` after a successful upstream fetch | log, increment, return fetched value uncached | caller gets the right answer; cache not populated this round |
| `negReg.PutWithTTL` after a 404 | log, increment, return upstream error | caller gets the right error; negative cache not populated this round |

A full Redis outage degrades atlas-monsters to "no cache" mode — exactly the pre-task behavior, plus log noise and counter increments to make the outage visible. The task's primary goal (≥95% upstream-call reduction) is *not* met during a Redis outage, but no requests fail.

### 7.5 Race-Detector Discipline

`go test -race ./...` for `services/atlas-monsters` must be clean. The only goroutines added are the implicit ones from `goredis.Client`'s connection pool, which is itself race-clean. The `cacheOnce sync.Once` guards `InitDataCache`. No new locks in this design.

### 7.6 Context Propagation

Every `redis.TenantRegistry` call takes `ctx`. The wrapper passes the caller's `ctx` through to both Redis ops and `upstreamFetch`. Cancellation propagates correctly: a cancelled `ctx` aborts the Redis call (which returns a context error treated as a transient Redis error → graceful fallthrough → `upstreamFetch(ctx)` also sees the cancelled ctx and aborts).

---

## 8. Alternatives Considered (v2)

### 8.1 v1 In-Process `libs/atlas-cache` — Rejected

Already discussed in §3. Loses cross-pod sharing, restart resilience, and adds a new module the codebase does not need.

### 8.2 Use `libs/atlas-redis.TTLRegistry` Instead of `TenantRegistry.PutWithTTL` — Rejected

`TTLRegistry` (`libs/atlas-redis/ttl.go`) is built around a sorted-set tracking pattern for `PopExpired`-style consumers (expressions, cashshop). It explicitly does **not** use Redis native TTL — line 56 says "Store data without Redis native TTL — PopExpired needs to read it." That's the wrong tradeoff for a read-through cache: we want Redis to evict for us, not to track expirations in an app-side sorted set. `TenantRegistry.PutWithTTL` (line 116) does use native EX TTL, which is what we want.

### 8.3 Use `libs/atlas-redis.CoalescedRegistry` for Hot-Tier In-Process Cache atop Redis — Deferred

`CoalescedRegistry` (`libs/atlas-redis/coalesced.go`) maintains an in-process `readCache map[string]V` refreshed periodically from Redis. If the latency delta (1 ms vs. 100 ns) ever matters, layering a `CoalescedRegistry`-style read-tier in front of `TenantRegistry` would recover most of it while preserving cross-pod correctness. **Not in v1.** The existing 24 rps profile gives Redis a trivial budget; adding the read-tier now would be premature optimization. If post-deploy metrics show p99 hit-latency is a problem, this is the natural next step — and it does not require any cache wire-format change.

### 8.4 Single Composite Namespace with a Typed Envelope — Rejected

Encode positive/negative in one `TenantRegistry[uint32, envelope]` where `envelope = struct{ Value Model; Negative bool }`. Allows different TTLs only by setting the right TTL at write time (which `PutWithTTL` supports). Downsides:

- One read returns either a positive or a negative entry, but the caller still needs both TTLs honored — meaning we'd write the envelope with whichever TTL applies, requiring the negative-vs-positive distinction at Put time anyway. We get nothing for the complexity.
- Operators lose the per-namespace key visibility (`KEYS atlas:monsters:cache:data:not_found:*` is gone).
- Unmarshalling cost grows (envelope is bigger; `Model` is read even on negative entries).

Two registries, two namespaces is the cleaner shape.

### 8.5 New Primitive `libs/atlas-redis.TenantTTLCache[K, V]` — Deferred

We could add a primitive that wraps positive+negative as a single typed pair: `c.Get(ctx, t, key) (V, found bool, err error)` and `c.PutNegative(ctx, t, key) error`. This removes ~30 lines of glue in `cache.go`. **Not in this task.** Reason: we have exactly one consumer right now. Two consumers (when `atlas-channel` and `atlas-monster-death` adopt this) is the right time to extract the pattern. Premature library factoring is what landed us in v1.

### 8.6 Single-Flight on Cold-Start Misses — Out of Scope

PRD non-goal. Same posture as v1 §7.4. Bounded by upstream rate; not justified by current load profile.

---

## 9. Test Plan Summary

| Layer | File | Coverage |
|---|---|---|
| Wrapper | `monster/information/cache_test.go` (NEW) | 1. Hit avoids upstream (positive). 2. Negative cache hit avoids upstream and synthesizes `requests.ErrNotFound`. 3. Transient error not cached. 4. 404 records negative entry; second call hits it. 5. Tenant isolation across two `tenant.Model` contexts. 6. Kill-switch (`MONSTER_DATA_CACHE_ENABLED=false`) bypasses cache entirely. 7. `MONSTER_DATA_CACHE_NEGATIVE_TTL=0` disables negative caching. 8. Redis-down simulation (close miniredis, expect graceful fallthrough; counter increments). 9. `loadConfig` env-var fallback on invalid input. |
| Wrapper | (existing `processor_test.go` etc.) | Continue to pass without modification. The cache wrapper is invisible to existing tests because they stub `information.GetById` via function-typed fields (`recovery_task.go:29`, `processor.go:65`). |
| Integration | None — `httptest`-based atlas-data integration is out of scope. The miniredis-backed wrapper tests give us deterministic coverage without standing up the full HTTP stack. |
| Race | `go test -race ./...` clean for atlas-monsters. |

**Test backend:** `github.com/alicebob/miniredis/v2` already exists in atlas-monsters' transitive deps via `libs/atlas-redis` test usage; if not, add it. miniredis honors `EX` TTL via its `FastForward` helper, which lets us deterministically test TTL expiry without sleeping.

**No `bench_test.go`:** the v1 design's `Get_Hit ≤ 200 ns/op` target is meaningless against Redis. p99 latency is a runtime/operational concern; we will sample it via the existing OpenTelemetry tracing in atlas-monsters once deployed, rather than gating on a microbenchmark.

---

## 10. Operational Notes

- **Deploy procedure:** standard. No data migration, no Kafka topic changes. The first restart of an atlas-monsters pod after deploy starts populating the cache; cross-pod warmth means the second pod restart sees a substantially-warm cache from day one.
- **Rollback:** set `MONSTER_DATA_CACHE_ENABLED=false` in the deployment env, restart the pod. The wrapper detects the disabled cache and bypasses both registries entirely — code path identical to pre-task.
- **Cache flush after `atlas-data` redeploy:** `redis-cli DEL atlas:monsters:cache:data:*` and `redis-cli DEL atlas:monsters:cache:data:not_found:*` (or `--scan --pattern` for safety on large keyspaces). No pod restart required. This is a meaningful improvement over v1 (which required pod restarts for any flush). Document in the service runbook in plan phase.
- **Future event-driven flush:** Kafka-driven invalidation on atlas-data redeploy still tracked as a follow-up. Implementation will publish a flush event; atlas-monsters subscribes and runs the `DEL` pattern above.
- **Memory:** Redis already provisioned; ~few thousand entries × few hundred bytes JSON × 5 tenants ⇒ low single-digit MB. Negligible.

---

## 11. Risks (v2)

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Stale data on `atlas-data` redeploy | High | Medium | TTL bound (5 m); `redis-cli DEL` for forced flush; future event-driven flush. **No regression vs. v1** — same invalidation story, lower MTTR (no pod restart). |
| Redis outage degrades atlas-monsters | Low (Redis is already a hard dep) | Low (graceful fallthrough; no failed requests) | All cache ops have a fallthrough path; `redis_errors_total` makes the degradation visible. |
| Redis network slow / saturated | Low | Low | `goredis.Client` defaults include a sane `DialTimeout`; if Redis is the slow path, p99 latency rises but no correctness issue. Monitor via existing tracing. |
| JSON marshal/unmarshal cost on every hit | Low | Low | `Model` is small; `encoding/json` handles thousands of ops/s on commodity hardware. Far cheaper than the upstream HTTP it replaces. |
| Cardinality explosion if tenant count grows | Low | Low | ≤80 series with 5 tenants. If tenant count grows beyond ~50, drop the `tenant` label or sample. |
| PRD divergence not reconciled before plan | Medium | Medium | §4 of this design enumerates the deltas; plan phase task 1 is "PRD revision PR before code." |
| Wrong error classifier shape | Low (corrected during v1 phase 4) | Medium | `requests.ErrNotFound` sentinel verified in `libs/atlas-rest/requests/get.go:14-15`. |

---

## 12. What This Design Deliberately Does Not Do

- Does not introduce a new `libs/<name>` module.
- Does not change Kafka topics, REST surfaces, JSON:API resources, or HTTP wire formats.
- Does not modify `atlas-data`, `atlas-channel`, `atlas-monster-death`.
- Does not add admin endpoints, single-flight, in-process hot-tier, LRU eviction, or `WarmAll`.
- Does not cache `data/maps/{id}`, `data/items/{id}`, `data/npcs/{id}`, etc. Each is a separate task that will follow the same two-`TenantRegistry` shape.
- Does not implement event-driven invalidation on atlas-data redeploys (still a follow-up).
- Does not add a `cache_size` gauge (Redis-side `KEYS`/`SCAN` is the wrong mechanism for steady-state metrics; operators introspect via `redis-cli` when they need a count).
- Does not add a microbenchmark suite (operational p99 from tracing replaces it).
- Does not change `information.GetById`'s public signature.
