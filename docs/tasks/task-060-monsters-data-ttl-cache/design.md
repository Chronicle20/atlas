# atlas-monsters Data TTL Cache — Design

Version: v1
Status: Draft
Created: 2026-05-08
Companion to: [`prd.md`](./prd.md)
---

## 1. Goals of This Document

The PRD already settles **what** and **why**. This design pins down **how**:

- The exact public Go API of `libs/atlas-cache` (signatures, types, constructor shape).
- The internal data structure and locking model.
- The shape of the wrapper inside `services/atlas-monsters/atlas.com/monsters/monster/information/`.
- The error-classification rules used for negative caching.
- Where Prometheus metrics live and which package owns them.
- The env-var loader and its failure modes.
- Test strategy (unit, race, integration with the existing `monster` processor tests).
- The handful of decisions left open in PRD §9.

Out of scope: implementation steps, file-by-file deltas, commit ordering. Those live in `plan.md` (next phase).

---

## 2. Architecture Overview

```
                ┌────────────────────────────────────────────────┐
                │ services/atlas-monsters/...../monster/         │
                │   processor.go, aggro_task.go, recovery_task.go│
                │   ─ unchanged callers ─                        │
                └───────────────────┬────────────────────────────┘
                                    │ information.GetById(l)(ctx)(id)
                                    ▼
                ┌────────────────────────────────────────────────┐
                │ services/.../monster/information/              │
                │                                                │
                │   processor.go ── read-through wrapper ────┐   │
                │                                            │   │
                │   cache.go ── per-tenant cache registry,   │   │
                │              env-var loader, metrics,      │   │
                │              error classification          │   │
                │                                            │   │
                │   requests.go, rest.go, model.go (unchanged)   │
                └────────────────────┬───────────────────────────┘
                                     │ Cache[uint32, Model] per tenant
                                     ▼
                ┌────────────────────────────────────────────────┐
                │ libs/atlas-cache  (NEW shared module)          │
                │                                                │
                │   cache.go    — generic TTL cache              │
                │   config.go   — Config struct + validation     │
                │   cache_test.go — unit + race tests            │
                │   bench_test.go — Get-hit microbench           │
                │   README.md                                    │
                └────────────────────────────────────────────────┘
```

The split is deliberate: `libs/atlas-cache` knows nothing about tenants, HTTP, or
Prometheus. The service-side `information/cache.go` knows nothing about map
internals. This is the same boundary discipline used in `libs/atlas-redis` vs.
service-side registry wrappers.

---

## 3. `libs/atlas-cache` — Library Design

### 3.1 Public API

```go
package cache

import (
    "sync"
    "time"
)

// Cache is a generic in-process TTL cache supporting distinct positive and
// negative entry TTLs. All methods are safe for concurrent use.
type Cache[K comparable, V any] interface {
    // Get returns the value if a non-expired positive entry exists.
    // Returns the zero value of V and false otherwise (including when
    // a negative entry is present — callers check IsNegative for that).
    Get(key K) (V, bool)

    // Put stores a positive entry. Overwrites any prior entry (positive
    // or negative). The entry's expiry is set to now() + Config.TTL.
    Put(key K, value V)

    // PutNegative records that 'key' was looked up and found to be absent.
    // Subsequent Get(key) calls will return (_, false), and IsNegative(key)
    // will be true, until now() + Config.NegativeTTL.
    // If Config.NegativeTTL is zero, PutNegative is a no-op.
    PutNegative(key K)

    // IsNegative returns true if a non-expired negative entry exists for key.
    IsNegative(key K) bool

    // Delete removes any entry (positive or negative) for key.
    Delete(key K)

    // Len returns the count of entries currently held, by kind.
    // PositiveLen is the count of non-expired positive entries; NegativeLen
    // is the count of non-expired negative entries. Expired entries are
    // counted as 0 (the implementation may purge them lazily on the next
    // touch). Both counters are O(n); they exist for the metrics gauge,
    // which is sampled, not hot-path.
    Len() (positive int, negative int)
}

type Config struct {
    // TTL is the lifetime of a positive entry. Required (must be > 0).
    TTL time.Duration

    // NegativeTTL is the lifetime of a negative entry. Zero disables
    // negative caching entirely (PutNegative becomes a no-op,
    // IsNegative always returns false).
    NegativeTTL time.Duration

    // Now is the clock function. If nil, time.Now is used. Tests inject
    // a fake clock here.
    Now func() time.Time
}

// New constructs a Cache. Panics if cfg.TTL <= 0 (programmer error;
// configuration validation belongs to the caller).
func New[K comparable, V any](cfg Config) Cache[K, V]
```

**Why this shape:**

- **One method per intent** rather than overloading `Get` to also report negative
  state. `Get` returning `(V, bool)` is the idiomatic Go map signature; callers
  that don't care about negative caching never see it.
- **No `error` returns.** The cache cannot fail at the API layer — there is no
  I/O. Removing `error` from the contract keeps call sites tight.
- **No `Set(K, V, ttl)`.** The whole point of distinct positive/negative TTLs is
  that they are *configured*, not per-call. Per-call TTL is a tax on every site.
- **`Delete` over `Invalidate`.** `Delete` is what the standard library calls
  the operation in `sync.Map`/maps; we follow that.
- **`Len() (int, int)` over a `Stats()` struct.** Two counts is small enough not
  to warrant a struct, and it composes naturally with the gauge label `kind`.
- **Generics over an interface tax.** The PRD calls for reuse across follow-up
  tasks (`map/information`, `item/information`, etc.). Generics avoid the
  type-assertion/boxing overhead an `interface{}`-based cache would impose on
  every hot-path read.

### 3.2 Internal Structure

```go
type entry[V any] struct {
    value     V
    expiresAt time.Time
    negative  bool
}

type cache[K comparable, V any] struct {
    mu      sync.RWMutex
    entries map[K]entry[V]
    cfg     Config
    now     func() time.Time
}
```

**Locking model:** a single `sync.RWMutex`. Reads (`Get`, `IsNegative`, `Len`)
take `RLock`. Writes (`Put`, `PutNegative`, `Delete`, lazy expiration on read)
take `Lock`.

**Why not `sync.Map`?**

- `sync.Map` is optimized for the case where keys are written once and read
  many times *across* goroutines, but it pays for that with allocations on
  every store and a more complex value-by-value path.
- Our key space (~few thousand monster ids per tenant) is bounded and small.
- The hot path is `Get`. With `RWMutex`, parallel readers don't contend; the
  rare writer (after a cache miss + upstream fetch) takes the exclusive lock
  for the duration of one map insert.
- The benchmark in §3.5 will validate this empirically. If `sync.Map` wins by
  >10%, switch — but pre-optimizing for `sync.Map` here ignores that the actual
  contention pattern is read-heavy with sparse writes.

**Why not `sync.Map` + atomic expiry?** Same reasoning; also, the lazy
expiration on read has to *delete* the expired key, which is a write under any
locking model.

### 3.3 Lazy Expiration

`Get`:

```
RLock
  e, ok := entries[k]
  if !ok          → RUnlock; return zero, false
  if e.negative   → RUnlock; return zero, false
  if e.expiresAt before now() → goto expired
  RUnlock
  return e.value, true

expired:
  RUnlock
  Lock
    e2, ok := entries[k]
    if ok && e2 == e (same expiresAt)  // re-check under write lock
      delete(entries, k)
      // metric: evictions_total{reason=expired_positive}
  Unlock
  return zero, false
```

`IsNegative` is the symmetric path with `e.negative` flipped. `Put` and
`PutNegative` go straight to write lock; they don't need to inspect prior
state.

**Why re-check under the write lock:** another goroutine may have already
purged or refreshed the entry between RUnlock and Lock. Without the re-check
we'd race on a refreshed entry and double-evict.

**No background sweeper in v1.** PRD §4.1 confirms this. The only downside is
that an expired entry never accessed again sits in the map forever; with a
bounded id space, this is fine.

### 3.4 Metrics: Library Stays Metric-Free

`libs/atlas-cache` does **not** import Prometheus. Reasons:

- Metric naming is the caller's prerogative (atlas-monsters, future
  atlas-channel, etc. all want different names and labels).
- Forcing every consumer to take on a Prometheus dep makes the library
  unusable in test contexts that don't expose `/metrics`.

The library exposes the raw signal `Len() (int, int)` and the eviction event
via `EvictionFunc` on `Config`:

```go
type Config struct {
    TTL         time.Duration
    NegativeTTL time.Duration
    Now         func() time.Time

    // OnEviction is called (under the write lock) when a lazy expiration
    // removes an entry. May be nil. 'kind' is "positive" or "negative".
    // Callers typically increment a Prometheus counter here.
    OnEviction func(kind string)
}
```

This is the only callback. Hits and misses are fully observable from the
caller's wrapper layer (which sees the return value of `Get`/`IsNegative`),
so there's no need for a `OnHit`/`OnMiss` hook.

### 3.5 Benchmark

A `bench_test.go` validates PRD §8.1 ("sub-microsecond on commodity hardware"):

```go
func BenchmarkCacheGet_Hit(b *testing.B)
func BenchmarkCacheGet_Miss(b *testing.B)
func BenchmarkCachePut(b *testing.B)
```

Targets (commodity laptop, single core):

| Op | Target |
|---|---|
| `Get` hit | ≤ 200 ns/op |
| `Get` miss | ≤ 100 ns/op |
| `Put` | ≤ 500 ns/op |

These are the gates for the design choice "RWMutex + map" — if any miss the
target by >2x, revisit (`sync.Map` or `xsync.Map`).

### 3.6 Module Layout

```
libs/atlas-cache/
├── go.mod                # module github.com/Chronicle20/atlas/libs/atlas-cache
├── go.sum
├── cache.go              # public API + impl (~150 LOC)
├── config.go             # Config + validation
├── cache_test.go         # unit tests (table-driven)
├── bench_test.go         # microbenchmarks
└── README.md             # 30-line user-facing doc
```

The module name follows the existing pattern: `github.com/Chronicle20/atlas/libs/atlas-cache`. Confirmed by inspection of `libs/atlas-tenant/go.mod`.

`go.mod` has **no dependencies** other than the Go standard library. Generics
have been GA since Go 1.18; the project is on Go 1.24+.

---

## 4. Service-Side Wrapper (`monster/information/cache.go`)

### 4.1 Files Touched

| File | Change |
|---|---|
| `monster/information/processor.go` | `GetById` becomes read-through |
| `monster/information/cache.go` | **NEW** — registry, env loader, metrics, classification |
| `monster/information/cache_test.go` | **NEW** — wrapper-level unit tests |
| `monsters/go.mod` + `go.sum` | Add `libs/atlas-cache` (and `prometheus/client_golang` if not yet present) |
| `monsters/go.mod` | `replace github.com/Chronicle20/atlas/libs/atlas-cache => ../../../../libs/atlas-cache` |

### 4.2 Per-Tenant Registry

```go
package information

import (
    "sync"

    "github.com/Chronicle20/atlas/libs/atlas-cache"
    "github.com/Chronicle20/atlas/libs/atlas-tenant"
    "github.com/google/uuid"
)

var (
    tenantCachesMu sync.RWMutex
    tenantCaches   = make(map[uuid.UUID]cache.Cache[uint32, Model])
    initOnce       sync.Once
    cacheCfg       cacheConfig // loaded once at first use
)

func cacheFor(t tenant.Model) cache.Cache[uint32, Model] {
    initOnce.Do(loadConfig) // env vars
    if !cacheCfg.enabled {
        return nil
    }
    id := t.Id()

    tenantCachesMu.RLock()
    c, ok := tenantCaches[id]
    tenantCachesMu.RUnlock()
    if ok {
        return c
    }

    tenantCachesMu.Lock()
    defer tenantCachesMu.Unlock()
    if c, ok = tenantCaches[id]; ok {
        return c // lost the race; reuse winner's instance
    }
    c = cache.New[uint32, Model](cache.Config{
        TTL:         cacheCfg.ttl,
        NegativeTTL: cacheCfg.negativeTTL,
        OnEviction: func(kind string) {
            evictionsTotal.WithLabelValues(id.String(), kind).Inc()
        },
    })
    tenantCaches[id] = c
    return c
}
```

**Why double-checked locking instead of `sync.Once` per tenant:** the tenant
set is dynamic (lazy on first lookup). A `map[uuid.UUID]*sync.Once` would have
the same lookup cost and add a layer of indirection. Double-checked locking is
the idiomatic pattern across this codebase (see `libs/atlas-redis/registry.go`
for an analogue).

**Kill-switch path:** when `cacheCfg.enabled == false`, `cacheFor` returns
`nil` and the wrapper falls through to a direct upstream call. No tenant maps,
no Prometheus emission. This is the rollback.

### 4.3 Read-Through `GetById`

```go
func GetById(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) (Model, error) {
    return func(ctx context.Context) func(monsterId uint32) (Model, error) {
        return func(monsterId uint32) (Model, error) {
            // Resolve tenant once. Errors here propagate as today.
            t := tenant.MustFromContext(ctx)
            tid := t.Id().String()

            c := cacheFor(t)
            if c == nil {
                return upstreamFetch(l, ctx, monsterId) // kill-switch / disabled
            }

            if v, ok := c.Get(monsterId); ok {
                hitsTotal.WithLabelValues(tid, "positive").Inc()
                return v, nil
            }
            if c.IsNegative(monsterId) {
                hitsTotal.WithLabelValues(tid, "negative").Inc()
                return Model{}, notFoundError(monsterId)
            }

            missesTotal.WithLabelValues(tid).Inc()
            v, err := upstreamFetch(l, ctx, monsterId)
            if err == nil {
                c.Put(monsterId, v)
                return v, nil
            }
            switch classifyError(err) {
            case errKindNotFound:
                errorsTotal.WithLabelValues(tid, "not_found").Inc()
                c.PutNegative(monsterId)
            case errKindTransient:
                errorsTotal.WithLabelValues(tid, "transient").Inc()
            case errKindParse:
                errorsTotal.WithLabelValues(tid, "parse").Inc()
            }
            return Model{}, err
        }
    }
}

func upstreamFetch(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (Model, error) {
    return requests.Provider[RestModel, Model](l, ctx)(requestById(monsterId), Extract)()
}
```

**Why preserve the curried signature:** every caller in §7.1 of the PRD
(`processor.go`, `aggro_task.go`, `recovery_task.go`) takes `information.GetById`
*verbatim*. Changing the signature is a service-wide refactor we don't need.

### 4.4 Negative Cache: Error Classification

```go
type errKind int
const (
    errKindNotFound errKind = iota
    errKindTransient
    errKindParse
)

func classifyError(err error) errKind {
    var rerr *requests.HTTPError
    if errors.As(err, &rerr) {
        if rerr.StatusCode == http.StatusNotFound {
            return errKindNotFound
        }
        if rerr.StatusCode >= 500 || rerr.StatusCode == 0 /* transport */ {
            return errKindTransient
        }
        // 4xx other than 404: treat as transient. Don't cache 401/403/etc.
        // These are unexpected and likely signal a misconfigured client.
        return errKindTransient
    }
    var jerr *json.SyntaxError
    if errors.As(err, &jerr) {
        return errKindParse
    }
    // Network errors, context cancellation, etc.
    return errKindTransient
}
```

**Why a single `notFoundError(id)` synthesizer for negative-cache hits:** PRD
§4.6 explicitly allows reconstruction over byte-equality. Storing the original
`error` would (a) defeat the point of `PutNegative` taking no value parameter
and (b) leak the underlying `requests.HTTPError` body — which is fine for the
real call but wasteful to keep around for 30 s. We synthesize the same shape:

```go
func notFoundError(id uint32) error {
    return fmt.Errorf("monster %d not found: %w",
        id, &requests.HTTPError{StatusCode: http.StatusNotFound})
}
```

Caller-observable equivalence with the upstream's 404: same wrapped sentinel,
same `errors.Is(err, requests.ErrNotFound)` (or whatever the existing helper
is — verified at implementation time against `libs/atlas-rest/requests`).

**Open verification (resolved at plan time):** confirm `libs/atlas-rest/requests`
exports a typed `HTTPError` with `StatusCode`. If not (it returns `fmt.Errorf`
strings), the classifier falls back to `strings.Contains(err.Error(), "404")`
— ugly but documented in the PRD as the source-of-truth contract for today's
behavior. Plan task: "Inspect `libs/atlas-rest/requests` and pick the right
classifier hook."

### 4.5 Metrics Wiring

Following the pattern at `services/atlas-maps/atlas.com/maps/character/location/metrics.go`,
declare in `monster/information/cache.go` (or a sibling `metrics.go` file):

```go
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

    cacheSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "atlas_monsters_data_cache_size",
        Help: "Current number of cached monster information entries, by tenant and kind.",
    }, []string{"tenant", "kind"})

    evictionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_cache_evictions_total",
        Help: "Lazy expirations of cached monster information entries, by tenant and reason.",
    }, []string{"tenant", "reason"})
)
```

`cacheSize` is sampled by a small goroutine started from `cacheFor` on first
tenant init — interval **30 s** (matches the existing `RegistryAudit` cadence
in `monster/`). The goroutine walks `tenantCaches`, calls `Len()` on each, and
sets the gauge. Lifetime is tied to process exit; no shutdown hook is wired
because Prometheus gauges going stale at shutdown is fine.

**Cardinality risk** (PRD §9): tenant count is currently ≤ 5 in production.
We accept the `tenant` label and document in the README that any future
multi-tenant explosion means revisiting these labels.

### 4.6 Configuration Loader

```go
type cacheConfig struct {
    enabled     bool
    ttl         time.Duration
    negativeTTL time.Duration
}

func loadConfig() {
    cacheCfg.enabled = parseBoolEnv("MONSTER_DATA_CACHE_ENABLED", true)
    cacheCfg.ttl = parseDurationEnv(
        "MONSTER_DATA_CACHE_TTL", 5*time.Minute, 1*time.Second, 24*time.Hour)
    cacheCfg.negativeTTL = parseDurationEnv(
        "MONSTER_DATA_CACHE_NEGATIVE_TTL", 30*time.Second, 0, 5*time.Minute)
}
```

`parseDurationEnv(name, def, min, max)` logs a warning at `WARN` level via the
service logger when the env var is invalid or out-of-range, then returns `def`.
The warning records the variable name, the offending value, and the chosen
fallback. No fatal errors at startup. (PRD §4.4 contract.)

**Why a private `init()`-time `loadConfig` instead of `func() Config`:**
- Service `main.go` already wires env vars implicitly via `os.Getenv` calls
  inside packages (see `kafka/consumer.go:23`). Following the existing pattern.
- Lazy init via `sync.Once` means tests can override env vars in `TestMain`
  before the first call resolves config.

### 4.7 Test Fakes for Existing Callers

`processor_test.go:1466-1475` already documents that tests cannot easily mock
`information.GetById` because it's a free function. The existing pattern
(see `recovery_task.go:29`) is to wrap `information.GetById` behind a
function-typed field on the consumer:

```go
// recovery_task.go (existing)
type recoveryTask struct {
    ...
    lookup func(ctx context.Context, monsterId uint32) (information.Model, error)
}
```

We do **not** retrofit this pattern across all callers in this task. Instead,
the new wrapper preserves `information.GetById`'s signature exactly, so
existing tests continue to call through. Cache-side tests in
`monster/information/cache_test.go` use a swappable transport hook:

```go
// in cache.go (test-only via build tag or unexported var)
var upstreamFn = upstreamFetch // overridable from tests
```

Alternative considered: thread a `*httptest.Server` through a custom
`requests.RootUrl("DATA")` override. Rejected because `RootUrl` reads
`os.Getenv` and is shared across packages — too invasive. The function-
pointer override is local to the wrapper file and matches the
`testInformationLookup` field already present in `monster/processor.go:65`.

---

## 5. Decisions on PRD §9 Open Questions

| Question | Decision | Rationale |
|---|---|---|
| Negative TTL default | **30 s** (per PRD) | Heuristic stands; revisit if metrics surface pathological behavior. |
| Cache value: `Model` vs `RestModel` | **`Model`** | Skip re-`Extract` on every hit; future generic "cached requests provider" can wrap at the `RestModel` layer for callers that don't already extract. Different concerns, different layers. |
| Library placement | **Standalone `libs/atlas-cache`** | Decoupling from `libs/atlas-redis` (Redis-specific) and `libs/atlas-rest/requests` (HTTP-specific) lets follow-ups reuse without dragging in Redis or HTTP deps. |
| Background sweeper | **No.** Lazy only. | Bounded id space; reclaiming on process restart is fine. API leaves room for a sweeper to be added later (`Config.SweepInterval` could be additive). |
| Metric label cardinality | **Keep `tenant` label**; document in README. | Tenant count is ≤ 5; preemptively dropping the label loses the most useful breakdown for SRE. Cost is negligible. |

---

## 6. Concurrency & Correctness Analysis

### 6.1 Hazards Considered

1. **Two goroutines miss for the same key concurrently** — both fan out to
   upstream, both call `Put`. The second `Put` overwrites the first. Result:
   one wasted upstream call; correctness preserved. The PRD explicitly chose
   **not** to add request coalescing in v1 (§Non-goals).

2. **Concurrent first-touch by two tenants** — `tenantCachesMu` serializes
   the registry insert. Worst case: each tenant gets its own cache instance
   (correct).

3. **Lazy eviction during a concurrent `Put`** — guarded by the re-check
   under the write lock (§3.3). Eviction races with itself become no-ops; it
   does not race with a fresh `Put` because the re-check compares
   `expiresAt`.

4. **Reading `cacheCfg` after `loadConfig`** — `sync.Once` provides the
   happens-before edge. `cacheCfg` is written exactly once, inside the
   `Once.Do` call, before any reader observes it.

5. **Tenant ID collision across cache reuse** — `tenant.Model.Id()` returns
   `uuid.UUID`, which is an array (value type, comparable). Map keys behave
   correctly with no aliasing. Two tenant instances with the same UUID share
   a cache, which is correct.

### 6.2 Race Detector Discipline

`go test -race ./...` is required clean in both modules. The library tests
include a fan-out test:

```go
func TestCache_Concurrent_GetPut(t *testing.T) {
    c := New[int, int](Config{TTL: time.Hour})
    var wg sync.WaitGroup
    for g := 0; g < 16; g++ {
        wg.Add(1)
        go func(g int) {
            defer wg.Done()
            for i := 0; i < 10_000; i++ {
                k := i % 100
                if g%2 == 0 {
                    c.Put(k, i)
                } else {
                    c.Get(k)
                }
            }
        }(g)
    }
    wg.Wait()
}
```

### 6.3 Tenant Isolation Test

```go
func TestGetById_TenantIsolation(t *testing.T) {
    // Two tenants with different upstream answers for the same id.
    // Verify each sees their own.
}
```

This test sets up two `httptest.Server`s (or one with branching), seeds
`tenant.MustFromContext` via `tenant.WithContext` for each request, and
asserts both caches stay disjoint. PRD §10 acceptance criterion.

---

## 7. Alternatives Considered

### 7.1 Extend `libs/atlas-redis` with In-Memory Variant — Rejected

We could add `libs/atlas-redis/InMemoryRegistry` mirroring the `Registry`
shape. Problems:

- Naming: a thing in the `redis` package that doesn't use Redis is confusing.
- Module dep: pulls in `goredis` for nothing.
- API mismatch: `redis.Registry` has `Get(ctx, key) (V, error)` because it's
  I/O. The cache should not require `ctx` or return `error`; doing so taxes
  every call site for a non-existent failure mode.

### 7.2 Cached `requests.Provider` Wrapper — Deferred

A natural follow-up is to expose `requests.Cached(provider, cache)` so any
`Provider[Rest, Model]` can become read-through. Appealing for
`atlas-channel`/`atlas-monster-death` follow-ups. **Not in this task** —
coupling cache to REST in v1 is premature; we don't yet know whether all
consumers want `Model` caching or `RestModel` caching, and the abstraction is
cheaper to design once two concrete consumers exist.

### 7.3 Single Composite-Keyed Cache Instead of Per-Tenant — Rejected

`Cache[tenantKey, Model]` where `tenantKey = struct{tenantId uuid.UUID; mid uint32}`.
Two downsides:

- Tenant deletion (when one is configured-off) cannot reclaim its memory in
  bulk. With per-tenant maps we can `delete(tenantCaches, id)`; with composite
  keys we'd have to scan.
- The metrics gauge `cacheSize{tenant=...}` requires per-tenant counts;
  composite key forces a scan to compute. With per-tenant caches it's a
  single `Len()` call.

Per-tenant matches PRD §4.5's preferred shape and the codebase idiom.

### 7.4 Single-Flight / Coalescing — Out of Scope

PRD §Non-goals. If metrics show a thundering-herd pattern at cache cold-start,
add `golang.org/x/sync/singleflight` *behind* the `Cache.Get` interface — no
caller change required. Documented as a future task.

### 7.5 LRU Eviction by Size — Rejected

Bounded id space (~thousands per tenant), small struct values. The acceptance
criterion (>95% hit rate) does not imply size pressure. Adding LRU costs an
ordered list traversal on every read. Skip.

### 7.6 Static `nowFn` Package Variable Instead of `Config.Now` — Rejected

A package-level `var Now = time.Now` (overridable in tests via
`Now = func() time.Time { return fake }`) is shorter. Problems:

- Globally shared state breaks parallel tests (`go test -parallel`).
- The library exposes one `Cache` type to users, but tests would mutate
  package-level state.

`Config.Now` is per-instance, race-free, parallel-safe.

---

## 8. Test Plan Summary

| Layer | File | Coverage |
|---|---|---|
| Library | `libs/atlas-cache/cache_test.go` | `Get` miss/hit, `Put`, `PutNegative`, `IsNegative`, expiration via injected `Now`, `Delete`, `Len`, eviction callback, race-clean fan-out |
| Library | `libs/atlas-cache/bench_test.go` | Microbench `Get` hit/miss, `Put` |
| Wrapper | `monster/information/cache_test.go` | Tenant isolation, hit avoids HTTP, negative-cache hit returns same error & avoids HTTP, 5xx not cached, parse error not cached, kill-switch bypasses cache, env var fallback on invalid duration |
| Wrapper | (existing tests) | All current `monster/processor_test.go` continue to pass without modification (PRD §10 acceptance criterion). |

`go test -race ./...` runs in both modules in CI.

---

## 9. Operational Notes

- **Deploy procedure:** standard. No data migration, no Kafka topic changes,
  no `atlas-tenants` config to seed.
- **Rollback:** set `MONSTER_DATA_CACHE_ENABLED=false` in the deployment env,
  restart the pod. Code path reverts to identical pre-task behavior.
- **Cache flush after `atlas-data` redeploy:** restart `atlas-monsters` pods.
  Out-of-scope to automate. Documented in the new `libs/atlas-cache/README.md`
  and noted in the service's deploy runbook (added in plan phase).
- **First-deploy expectation:** miss rate is ~100% for the first ~5 ms, then
  drops to <5% within a few seconds as the working-set ids populate.

---

## 10. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Stale data on `atlas-data` redeploy | High | Medium | TTL bound (5 m default); pod restart; documented procedure. Future task: event-driven flush. |
| 5xx storms still hammer atlas-data | Low | Low | Negative cache only catches 404. Transient errors retry as today. If 5xx storms become a real pattern, add coalescing. |
| Memory growth from unused tenant caches | Low | Low | Per-tenant entries are small; tenant count is bounded. Monitored by `cache_size` gauge. |
| Wrong error classifier shape | Medium | Medium | Plan task explicitly inspects `libs/atlas-rest/requests` and adapts. Fall-back uses string match if no typed error is exported. |
| Prometheus dep adds first-time wiring to `atlas-monsters` | Low | Low | Pattern lifted verbatim from `atlas-maps`. |
| Hidden caller assumed `GetById` was authoritative real-time | Low | High | None known. The data is WZ-derived and immutable until redeploy — that's the whole premise of the task. Any caller that needs cache bypass can use a future `GetByIdNoCache` hook (not added in v1 — YAGNI). |

---

## 11. What This Design Deliberately Does Not Do

- Does not change Kafka topics, REST surfaces, or JSON:API resources.
- Does not modify `atlas-data`, `atlas-channel`, `atlas-monster-death`.
- Does not add `Sweep`, `Refresh`, `WarmAll`, or admin endpoints.
- Does not cache anything other than monster data in `atlas-monsters`.
- Does not coalesce concurrent misses.
- Does not introduce a generic `requests.Cached` wrapper.
- Does not cap memory by entry count.

Each of these is either a Non-Goal in the PRD or a deferred follow-up.
