# task-061 — atlas-data Cache Invalidation: Architecture & Tradeoffs

Status: Draft
Created: 2026-05-08
Revised: 2026-05-08 — v2 pivot to align with task-060 v2 (Redis-backed cache; no `libs/atlas-cache` module; no per-pod fanout).
Builds on: PRD `prd.md` (v2); depends on task-060 v2 (`libs/atlas-redis.TenantRegistry`-backed `monster/information` cache).

This document fixes the architectural choices that the PRD left open at §9 and pins down concrete file paths, function signatures, and test surfaces. It is the contract that `plan.md` will decompose into discrete tasks.

---

## 1. Goals of This Document

1. Decide where each piece of new code lives, given that task-060 v2 abandoned the in-process cache and put both proof-case caches in **shared Redis**.
2. Pick concrete signatures for the new producer (`atlas-data`), the two new consumers (`atlas-monsters`, `atlas-maps`), and the new `libs/atlas-redis.TenantRegistry.Clear` library affordance.
3. Resolve PRD §9 open questions with stated rationale.
4. Specify the consumer-group convention (shared, single-delivery; **not** per-pod) and document the rationale in code so the next maintainer doesn't re-derive it.
5. Document the cross-cutting concerns — tenant isolation, error handling, observability, kill-switch behavior — that the plan will turn into tests.

---

## 2. Architecture Overview

```
                                 Kafka EVENT_TOPIC_DATA
                                 key = tenantId, partitions = N
                                       ▲                ▲
                                       │ produces       │ subscribes (shared group)
                                       │                │
   ┌───────────────────────┐     ┌─────┴─────┐    ┌─────┴────────────────────┐
   │  atlas-data           │     │ atlas-data│    │ atlas-monsters           │
   │  StartWorker(name)    │────▶│ producer  │    │   group:                 │
   │  on success per       │     └───────────┘    │   "Monster Data          │
   │  (tenant, worker)     │                      │    Cache Invalidator"    │
   └───────────────────────┘                      │                          │
                                                  │ atlas-maps               │
                                                  │   group:                 │
                                                  │   "Map Spawn Registry    │
                                                  │    Invalidator"          │
                                                  └────────────┬─────────────┘
                                                               │ filter on
                                                               │ event.body.worker
                                                               ▼
                                          ┌──────────────────────────────────┐
                                          │ atlas-monsters:                  │
                                          │   information.FlushTenant(ctx,t) │
                                          │   → posReg.Clear + negReg.Clear  │
                                          │                                  │
                                          │ atlas-maps:                      │
                                          │   SpawnPointRegistry.            │
                                          │     FlushTenant(ctx,l,tenantId)  │
                                          │   → SCAN + pipelined DEL         │
                                          │                                  │
                                          │ All hit shared Redis;            │
                                          │ all replicas see the result.     │
                                          └──────────────────────────────────┘
```

**Three layers, each with one responsibility:**

| Layer | Responsibility | Lives in |
|---|---|---|
| Library affordance | Tenant-scoped namespace flush; one method on `TenantRegistry`. Knows nothing about Kafka. | `libs/atlas-redis/tenant_registry.go` |
| Service-side wrapper | Composes one or more `TenantRegistry.Clear` calls (atlas-monsters: positive + negative; future services may be more) plus kill-switch checks and metric emission. | `services/<svc>/atlas.com/<svc>/<domain>/cache.go` |
| Event plumbing | Producer in `atlas-data` emits `DATA_UPDATED`; consumers in `atlas-monsters` and `atlas-maps` subscribe and call the wrapper's `FlushTenant`. | `services/atlas-data/atlas.com/data/data/{kafka.go,producer.go,processor.go}` and `services/<svc>/atlas.com/<svc>/kafka/consumer/data/` |

The library knows nothing about events. The wrappers know nothing about Kafka. The consumers know nothing about Redis namespacing. Each layer can be tested in isolation.

**Why shared Redis changes everything from v1:** in v1 (where the cache was in-process per pod), every replica had to receive the event and flush its own copy. v1 therefore needed per-pod consumer groups. **In v2, one consumer's `Clear` mutates the shared Redis state immediately visible to every replica.** A single delivery per service is sufficient and correct. The whole "per-pod fan-out" infrastructure goes away.

---

## 3. `libs/atlas-redis` — `TenantRegistry.Clear` Addition

Task-060 v2 lands two `TenantRegistry` instances in `monster/information` (positive in namespace `monsters:cache:data`, negative in `monsters:cache:data:not_found`). The library does not currently expose a way to flush every key for a given tenant under a namespace. This task adds **one method**.

### 3.1 Public API Addition

```go
// Clear deletes every entry for tenant t in this registry's namespace.
// Implementation uses SCAN with COUNT=100 to enumerate keys matching
// tenantScanPattern(r.namespace, t), then pipelines DEL in batches of
// 100. Returns the number of keys deleted (0 if the namespace was
// already empty for this tenant). On a partial failure mid-scan,
// returns (deleted_so_far, err) — the partial deletion is not rolled
// back; Redis converges on the next call.
func (r *TenantRegistry[K, V]) Clear(ctx context.Context, t tenant.Model) (deleted int, err error)
```

`tenantScanPattern(namespace, t)` already exists at `libs/atlas-redis/keys.go:32` and yields `atlas:<namespace>:<tenantUUID>:<region>:<major>.<minor>:*`. `Clear` reuses it directly.

### 3.2 Implementation Sketch

```go
func (r *TenantRegistry[K, V]) Clear(ctx context.Context, t tenant.Model) (int, error) {
    pattern := tenantScanPattern(r.namespace, t)
    iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()

    deleted := 0
    pipe := r.client.Pipeline()
    pipeSize := 0
    var firstErr error

    flushPipe := func() {
        if pipeSize == 0 {
            return
        }
        if _, err := pipe.Exec(ctx); err != nil && firstErr == nil {
            firstErr = err
        }
        pipe = r.client.Pipeline()
        pipeSize = 0
    }

    for iter.Next(ctx) {
        pipe.Del(ctx, iter.Val())
        deleted++
        pipeSize++
        if pipeSize >= 100 {
            flushPipe()
        }
    }
    flushPipe()

    if err := iter.Err(); err != nil && firstErr == nil {
        firstErr = err
    }
    return deleted, firstErr
}
```

**Why `SCAN` with `COUNT=100` and pipelined `DEL`:** `KEYS atlas:<ns>:<tenantKey>:*` would briefly block the Redis server on a large tenant. `SCAN` is the cooperative variant; pipelining `DEL` keeps round-trips to ≤ N/100. PRD §8.1 caps tenant key counts at "a few thousand at most," which converges in single-digit milliseconds.

**Why one `firstErr` accumulator instead of returning immediately on first error:** namespace flushes are not transactional anyway — a partial deletion is recoverable on the next event or via TTL fallback. Aborting halfway through leaves a noisy half-flushed state with no benefit. We log per-batch and surface the *first* error to the caller so metrics don't become misleadingly silent on degraded-mode runs.

**Why expose `deleted int` even on failure:** the caller's metric (`atlas_<svc>_data_events_keys_deleted_total`) needs to attribute work that succeeded. Returning `(0, err)` on a partial would understate observability.

**Why no Lua-scripted atomic SCAN+DEL:** Lua scripts that iterate the keyspace block the entire Redis server; for `atlas-monsters`' shared cluster (running cooldown updates, monster registry I/O, drop timers) on a single broker, that's a hard no. Pipelined DEL is cooperative and lets unrelated traffic interleave between batches.

### 3.3 Tests Added in `libs/atlas-redis`

| Test | Asserts |
|---|---|
| `TestTenantRegistry_Clear_EmptyNamespace` | `Clear` on a namespace with no entries returns `(0, nil)`. |
| `TestTenantRegistry_Clear_DeletesAllForTenant` | After `Put` of 5 entries, `Clear` returns `(5, nil)` and a subsequent `Exists` returns false for each key. |
| `TestTenantRegistry_Clear_TenantIsolation` | After populating tenant A and tenant B in the same namespace, `Clear(ctx, tenantA)` deletes all of A and 0 of B. |
| `TestTenantRegistry_Clear_NamespaceIsolation` | After populating two registries with the same tenant but different namespaces, `Clear` on registry 1 deletes only registry 1's keys; registry 2's keys remain. |
| `TestTenantRegistry_Clear_PartialFailureSurfacesError` | A test hook that forces one DEL to fail mid-scan: result is `(partial_count, err)` where `partial_count > 0`; no panic; subsequent `Clear` succeeds and zeros the remainder. |
| `TestTenantRegistry_Clear_RaceCleanWithPut` | `go test -race`: 4 goroutines doing concurrent `PutWithTTL`, 1 goroutine doing `Clear` for 1 s; no race detector hits, no panics. Final state may have new puts (acceptable — `Clear` is "delete everything that was there at scan time"). |

Tests use the existing miniredis pattern from other `libs/atlas-redis` tests.

### 3.4 No `Cache.Flush` (the v1 plan is dead)

The v1 plan added `Cache.Flush()` to a hypothetical `libs/atlas-cache.Cache[K,V]` interface. **That interface was reverted in task-060 (commits `e983a009a` and `29c6f9603`).** This task therefore does not touch any `libs/atlas-cache`. The library affordance is the single `TenantRegistry.Clear` method described above.

---

## 4. Service-Side Wrappers: Per-Tenant `FlushTenant`

Each service that owns Redis-backed cache state for `atlas-data` exposes a `FlushTenant` function in its package. The function composes one or more `TenantRegistry.Clear` calls (or, for `atlas-maps`' hand-rolled key shape, a parallel SCAN/DEL implementation) and applies the service's kill-switch and metric concerns.

### 4.1 `services/atlas-monsters/.../monster/information/cache.go` — `FlushTenant`

```go
// FlushTenant clears both the positive and negative cache namespaces
// for tenant t. If the cache is disabled (MONSTER_DATA_CACHE_ENABLED=false)
// or has not yet been initialized, this is a no-op returning (0, nil).
// On partial Redis failure across the two namespaces, returns the running
// total of keys deleted and the first error observed; the second Clear
// is still attempted so a degraded posReg does not block negReg cleanup.
func FlushTenant(ctx context.Context, t tenant.Model) (deleted int, err error) {
    if cache == nil || !cache.enabled {
        return 0, nil
    }

    posDeleted, posErr := cache.posReg.Clear(ctx, t)
    deleted += posDeleted

    negDeleted, negErr := cache.negReg.Clear(ctx, t)
    deleted += negDeleted

    if posErr != nil {
        err = posErr
    } else if negErr != nil {
        err = negErr
    }
    return deleted, err
}
```

**Why call both Clears even if the first errors:** v2 keeps positive and negative namespaces independent. A failure in one doesn't say anything about the other; flushing the second namespace independently maximizes the steady-state convergence of the system. The first error is still surfaced for the metric / log path.

**Why "no-op if `cache == nil`":** `InitDataCache` is called from `main.go` after Redis connection. If for any reason the consumer fires before init completes (extremely unlikely given the standard wiring order), we don't want to panic — silently no-op and let the next event do the work.

**Why kill-switch path returns before any Redis call:** preserves the task-060 v2 promise that disabling the cache costs zero Redis I/O.

The package gains one new test in `cache_test.go`:

| Test | Asserts |
|---|---|
| `TestFlushTenant_ClearsBothNamespaces` | After populating tenant T with 3 positive + 2 negative entries, `FlushTenant(ctx, T)` returns `(5, nil)` and both namespaces are empty for T. |
| `TestFlushTenant_TenantIsolation` | After populating tenant A and tenant B, `FlushTenant(ctx, A)` clears A's entries in both namespaces and leaves B's entries intact. |
| `TestFlushTenant_KillSwitch` | With cache disabled, `FlushTenant` returns `(0, nil)` and never touches Redis. |
| `TestFlushTenant_PosRegErrorDoesNotBlockNegReg` | A test hook forces `posReg.Clear` to return an error; `negReg.Clear` is still invoked; final `err` is the posReg error; `deleted` reflects negReg's count plus any partial pos count. |

### 4.2 `services/atlas-maps/.../map/monster/registry.go` — `FlushTenant`

`atlas-maps`' `SpawnPointRegistry` is **not** a `TenantRegistry`. It uses a hand-rolled key shape `atlas:maps:spawn:{tenant}:{world}:{channel}:{map}:{instance}` (see `registry.go:60`) that does not match `tenantEntityKey`'s shape `atlas:<namespace>:<tenantKey>:<id>`. Migrating SpawnPointRegistry to TenantRegistry is out of scope (per PRD §9). We add a parallel SCAN/DEL implementation:

```go
// FlushTenant deletes every spawn-point hash for tenantId.
// Uses SCAN with COUNT=100 to avoid blocking the broker on large key
// spaces; pipelines DEL per batch. Errors are logged at WARN per batch
// and surfaced via the returned error; partial deletions are not rolled
// back.
func (r *SpawnPointRegistry) FlushTenant(ctx context.Context, l logrus.FieldLogger, tenantId uuid.UUID) (deleted int, err error) {
    pattern := fmt.Sprintf("atlas:maps:spawn:%s:*", tenantId.String())
    iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()

    pipe := r.client.Pipeline()
    pipeSize := 0
    flushPipe := func() {
        if pipeSize == 0 {
            return
        }
        if _, perr := pipe.Exec(ctx); perr != nil {
            l.WithError(perr).Warnf("Spawn-registry DEL batch failure for tenant [%s].", tenantId)
            if err == nil {
                err = perr
            }
        }
        pipe = r.client.Pipeline()
        pipeSize = 0
    }

    for iter.Next(ctx) {
        pipe.Del(ctx, iter.Val())
        deleted++
        pipeSize++
        if pipeSize >= 100 {
            flushPipe()
        }
    }
    flushPipe()

    if ierr := iter.Err(); ierr != nil {
        l.WithError(ierr).Warnf("Spawn-registry SCAN failure for tenant [%s].", tenantId)
        if err == nil {
            err = ierr
        }
    }
    return deleted, err
}
```

This is a deliberate code duplication with `TenantRegistry.Clear` — the structure is identical because the operation is identical. Unifying them would require either (a) generalizing `TenantRegistry` to support arbitrary key shapes (over-engineering) or (b) migrating `SpawnPointRegistry` to `TenantRegistry` (out of scope, per PRD §9). The duplication cost is one function; the unification cost is multiple. YAGNI.

Tests live in `services/atlas-maps/atlas.com/maps/map/monster/registry_test.go` (or `flush_test.go` if cleaner separation is preferred):

| Test | Asserts |
|---|---|
| `TestSpawnPointRegistry_FlushTenant_DeletesAllForTenant` | After `InitializeForMap` for two maps under tenant T, `FlushTenant(ctx, l, T)` returns `(2, nil)` and both keys are gone. |
| `TestSpawnPointRegistry_FlushTenant_TenantIsolation` | After populating tenants A and B, `FlushTenant(ctx, l, A)` deletes A's keys and leaves B's intact. |
| `TestSpawnPointRegistry_FlushTenant_RedisError` | Forced Redis error returns `(partial, err)` and logs a WARN; consumer's offset still commits. |
| `TestSpawnPointRegistry_FlushTenant_EmptyTenant` | Tenant with no spawn keys returns `(0, nil)` with no Redis ops other than SCAN. |

### 4.3 Why Not Push the Wrapper Into the Library

The PRD §4.4 v1 originally specified a "registry-level helper FlushTenant on the per-tenant registry pattern" inside the library. With v2's pivot to Redis, that pattern doesn't exist as a library construct: the library knows about `TenantRegistry.Clear` (one method, one responsibility); the *composition* of multiple Clears + kill-switch + metric emission is service-specific. Pushing it into the library would force the library to know about service env vars and Prometheus, which it deliberately does not.

YAGNI: keep the library minimal (one new method). Each service composes its own thin `FlushTenant`.

---

## 5. `atlas-data` — Producer Wiring

### 5.1 New Topic Constants

In `services/atlas-data/atlas.com/data/data/kafka.go`:

```go
const (
    EnvCommandTopic       = "COMMAND_TOPIC_DATA" // existing
    CommandStartWorker    = "START_WORKER"       // existing

    EnvEventTopic         = "EVENT_TOPIC_DATA"   // NEW
    EventTypeDataUpdated  = "DATA_UPDATED"       // NEW
)

// existing command[E]...

// NEW
type event[E any] struct {
    Type string `json:"type"`
    Body E      `json:"body"`
}

type dataUpdatedEventBody struct {
    TenantId    string `json:"tenantId"`    // canonical tenant UUID
    Worker      string `json:"worker"`      // one of data.WorkerMap, data.WorkerMonster, ...
    CompletedAt string `json:"completedAt"` // RFC3339 UTC
}
```

**Why duplicate the `event[E]` shape locally** (when `command[E]` already exists with the same shape): keeping `command` and `event` as distinct types makes the producer signatures self-documenting and prevents accidental cross-wiring (a `command` provider on an event topic, or vice versa). The cost is two identical struct definitions — acceptable.

**Why `string` for `TenantId` and `CompletedAt` (not `uuid.UUID` and `time.Time`):** consistency with the on-the-wire shape used elsewhere in the repo. JSON-encoding `uuid.UUID` works, but `string` makes the contract explicit and survives any future change to the UUID library.

### 5.2 Producer Provider

In `services/atlas-data/atlas.com/data/data/producer.go`:

```go
func dataUpdatedEventProvider(tenantId string, worker string, completedAt time.Time) model.Provider[[]kafka.Message] {
    key := []byte(tenantId) // partition by tenant for ordered per-tenant emission
    value := &event[dataUpdatedEventBody]{
        Type: EventTypeDataUpdated,
        Body: dataUpdatedEventBody{
            TenantId:    tenantId,
            Worker:      worker,
            CompletedAt: completedAt.UTC().Format(time.RFC3339),
        },
    }
    return producer.SingleMessageProvider(key, value)
}
```

The Kafka **message key** is the tenant UUID string. This guarantees per-tenant emission ordering even with multiple partitions (Kafka hashes key → partition). Cross-tenant ordering is irrelevant.

**Tenant headers also flow** because `producer.ProviderImpl` (the existing wrapper at `services/atlas-data/atlas.com/data/kafka/producer/producer.go`) attaches `TenantHeaderDecorator(ctx)` on every message. So consumers can recover tenant via either the body's `TenantId` field OR the standard tenant headers via `TenantHeaderParser`. The body is canonical (PRD §4.8 says consumers MUST flush only `event.Body.TenantId`); the headers are belt-and-braces.

### 5.3 Emit Site in `StartWorker`

The current `StartWorker` shape (`processor.go:88-189`) is a long if-else chain culminating in:

```go
if err != nil {
    l.WithError(err).Errorf("Worker [%s] failed with error.", name)
    return err
}
l.Infof("Worker [%s] completed.", name)
return nil
```

Insert producer call between the success log and the return, gated on the kill-switch env var:

```go
l.Infof("Worker [%s] completed.", name)
emitDataUpdated(l, ctx, t, name) // NEW; never returns an error
return nil
```

```go
// emitDataUpdated emits a DATA_UPDATED event for (tenant, worker). Failures
// are logged at WARN and counted in atlas_data_events_emit_failures_total but
// MUST NOT fail the worker — a Kafka outage should never roll back a
// successful data import.
func emitDataUpdated(l logrus.FieldLogger, ctx context.Context, t tenant.Model, worker string) {
    if !producerEnabled() {
        return
    }
    err := producer.ProviderImpl(l)(ctx)(EnvEventTopic)(
        dataUpdatedEventProvider(t.Id().String(), worker, time.Now()),
    )
    if err != nil {
        l.WithError(err).Warnf("Failed to emit DATA_UPDATED for tenant [%s] worker [%s]; cache invalidation will rely on TTL fallback.", t.Id(), worker)
        eventsEmitFailuresTotal.WithLabelValues(worker, EventTypeDataUpdated).Inc()
        return
    }
    eventsEmittedTotal.WithLabelValues(worker, EventTypeDataUpdated).Inc()
}

func producerEnabled() bool {
    v, ok := os.LookupEnv("DATA_EVENTS_PRODUCER_ENABLED")
    if !ok {
        return true // default on
    }
    enabled, err := strconv.ParseBool(v)
    if err != nil {
        return true // unparseable → on (kill-switch is defensive, not strict)
    }
    return enabled
}
```

**Why a wrapper function instead of inlining at the success site:** `StartWorker`'s body is 100+ LOC of if-else. Adding three lines and a 20-line emit helper at the bottom keeps the success path readable and isolates the new failure mode (Kafka unavailable) behind a single `WARN` log line that operators will recognize.

**Why `time.Now()` (not `t.Now()` or a clock injection):** the PRD `CompletedAt` is informational; not used for any correctness decision. Keeping it on real wall-clock time is fine. (Tests stub by reading the resulting Kafka message and asserting the field is within a recent window.)

**Why log-and-continue on producer failure:** if the producer fails because Kafka is unreachable, the data import already succeeded (worker returned nil). Forcing the import to fail on Kafka error would be worse than the current behavior (manual flush) and would cause a partial import — the import has succeeded; we should report it as such.

### 5.4 Atlas-data Metrics

Two new counters in `services/atlas-data/atlas.com/data/data/metrics.go` (NEW file, parallel to existing service metric declarations elsewhere — confirmed pattern at `services/atlas-maps/.../character/location/metrics.go`):

```go
var (
    eventsEmittedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_data_events_emitted_total",
        Help: "Successful Kafka emits of data lifecycle events, by worker and type.",
    }, []string{"worker", "type"})

    eventsEmitFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_data_events_emit_failures_total",
        Help: "Failed Kafka emits of data lifecycle events, by worker and type.",
    }, []string{"worker", "type"})
)
```

Cardinality: `worker` ∈ 17 values from `data.Workers`; `type` ∈ {`DATA_UPDATED`} for now. 17 series total per metric. Trivial.

### 5.5 Atlas-data Tests

| Test | Asserts |
|---|---|
| `TestStartWorker_EmitsOnSuccess` | A successful worker run produces exactly one Kafka message with `Type=DATA_UPDATED`, the right `Worker`, and a tenant id matching the context. |
| `TestStartWorker_DoesNotEmitOnFailure` | When the underlying init/register call returns an error, no Kafka message is produced and the existing error is returned unchanged. |
| `TestStartWorker_KillSwitchSuppressesEmit` | With `DATA_EVENTS_PRODUCER_ENABLED=false`, no message is produced regardless of worker outcome. |
| `TestStartWorker_ProducerFailureDoesNotFailWorker` | A producer that returns an error logs WARN, increments emit-failure counter, and StartWorker still returns nil. |
| `TestDataUpdatedEventProvider_KeyIsTenantId` | The Kafka message key equals the tenant UUID bytes. |
| `TestDataUpdatedEventProvider_CompletedAtIsRFC3339UTC` | `body.completedAt` parses as RFC3339 and the location is UTC. |

Test fixtures use the existing `producer.MessageProducer` mock pattern from `libs/atlas-kafka/producer/producer_test.go`.

---

## 6. Consumer-Group Convention (Shared Groups)

### 6.1 Each Service Gets a New Fixed Group

| Service | Existing fixed group (commands) | New fixed group (data events) |
|---|---|---|
| `atlas-monsters` | `"Monster Registry Service"` | `"Monster Data Cache Invalidator"` |
| `atlas-maps` | `"Map Service"` | `"Map Spawn Registry Invalidator"` |

A single Kafka consumer per service receives each `DATA_UPDATED` event. Because both proof caches are stored in **shared Redis**, that one consumer's `Clear` call mutates state visible to every replica. There is no need to fan out the event to every pod.

**Why a separate group per service rather than reusing the existing command-topic group:**

- Offset state independence. The command-topic consumer commits offsets at its own pace; intermingling cache-invalidation offsets risks an unrelated command-handling problem stalling cache flushes (or vice versa).
- Distinct restart semantics: command consumers use `FirstOffset` (don't lose commands); data-event consumers use `LastOffset` (don't replay history). Different groups let each set its own offset reset policy.
- Discoverability: `consumerGroupId` strings are visible in Kafka tooling; "Monster Data Cache Invalidator" is a clearer ops signal than overloaded reuse.

### 6.2 `auto.offset.reset = latest`

The decorator already exists. Use `consumer.SetStartOffset(kafka.LastOffset)` on the new consumer registration:

```go
rf(
    consumer2.NewConfig(l)("data_events")(EnvEventTopicData)(consumerGroupId),
    consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
    consumer.SetStartOffset(kafka.LastOffset),
)
```

**Why `LastOffset`:** with a fixed group id, a fresh deploy starts a new consumer and Kafka has no committed offset for it yet. Default `FirstOffset` would consume the entire topic history — every flush event ever emitted. Wasted work, possibly thousands of flushes on a fresh pod. `LastOffset` says: start from the tail. Events emitted between consumer-startup and group-coordinator-readiness may be missed; that's acceptable per PRD §2 ("in-flight events during a deploy may be missed by a starting service and that is acceptable").

### 6.3 Why Not Per-Pod Groups (Anymore)

The v1 PRD made multi-pod fan-out a primary motivator. Rationale: in-process caches are per-pod; every replica needs its own flush; therefore every replica needs its own consumer-group id (since Kafka delivers each partition to exactly one consumer per group).

**Task-060 v2 inverted that premise.** Both proof caches now live in shared Redis. One pod's `Clear` is immediately visible to every other pod — no fan-out is required. Per-pod groups would deliver the *same event* to N replicas, all of which would race to do the same SCAN/DEL on the same Redis keys. That's wasted work, more Kafka group-coordinator load, and group-id sprawl.

**Forward compatibility.** If a future task adds an in-process (per-pod) cache that bypasses Redis, that consumer will need per-pod fan-out. We document the rationale in the new consumer code so the next maintainer doesn't re-derive it. We do **not** ship a `consumer.PerPodGroup` helper now — there is no in-process cache planned, and adding 25 LOC of library code for a hypothetical future need violates YAGNI.

If it is ever needed, the helper is a 25-line addition to `libs/atlas-kafka/consumer/group.go` (sketched in v1 of this design and now archived in §11.4 below for reference).

### 6.4 Documenting the Choice In Code

Each new `consumer.go` includes a short comment block at the top explaining the shared-group choice:

```go
// Package data subscribes to EVENT_TOPIC_DATA for tenant-scoped cache
// invalidation events. We use a shared (single-delivery) consumer group
// "Monster Data Cache Invalidator" rather than a per-pod group because
// the cache state is in shared Redis (task-060 v2): one pod's Clear is
// visible to every replica immediately. If a future cache moves
// in-process, this consumer will need per-pod fan-out — see
// docs/tasks/task-061-data-cache-invalidation/design.md §6.3.
```

---

## 7. Consumer Wiring

### 7.1 `atlas-monsters` — `kafka/consumer/data/`

New tree mirroring the existing `kafka/consumer/monster/` shape:

```
services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/
├── consumer.go     # InitConsumers / InitHandlers
├── kafka.go        # event[E] + dataUpdatedEventBody + topic env constant
├── handler.go      # handleDataUpdated
└── handler_test.go # unit tests
```

```go
// kafka.go
package data

const (
    EnvEventTopic        = "EVENT_TOPIC_DATA"
    EventTypeDataUpdated = "DATA_UPDATED"

    WorkerMonster = "MONSTER" // duplicated from atlas-data; stable contract
)

type event[E any] struct {
    Type string `json:"type"`
    Body E      `json:"body"`
}

type dataUpdatedEventBody struct {
    TenantId    string `json:"tenantId"`
    Worker      string `json:"worker"`
    CompletedAt string `json:"completedAt"`
}
```

```go
// consumer.go
package data

import (
    consumer2 "atlas-monsters/kafka/consumer"
    "github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
    "github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
    "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
    "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
    "github.com/Chronicle20/atlas/libs/atlas-model/model"
    "github.com/segmentio/kafka-go"
    "github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
    return func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
        return func(groupId string) {
            rf(
                consumer2.NewConfig(l)("data_events")(EnvEventTopic)(groupId),
                consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
                consumer.SetStartOffset(kafka.LastOffset),
            )
        }
    }
}

func InitHandlers(l logrus.FieldLogger) func(rf func(string, handler.Handler) (string, error)) error {
    return func(rf func(string, handler.Handler) (string, error)) error {
        if !consumerEnabled() {
            l.Infof("DATA_EVENTS_CONSUMER_ENABLED=false; not registering DATA_UPDATED handler.")
            return nil
        }
        t, _ := topic.EnvProvider(l)(EnvEventTopic)()
        if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDataUpdated))); err != nil {
            return err
        }
        return nil
    }
}
```

```go
// handler.go
package data

import (
    "context"
    "os"
    "strconv"

    "atlas-monsters/monster/information"

    tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

func handleDataUpdated(l logrus.FieldLogger, ctx context.Context, e event[dataUpdatedEventBody]) {
    if e.Type != EventTypeDataUpdated {
        eventsSkippedTotal.WithLabelValues("unknown_type").Inc()
        return
    }
    if e.Body.Worker != WorkerMonster {
        eventsSkippedTotal.WithLabelValues("unrelated_worker").Inc()
        return
    }

    bodyTid, err := uuid.Parse(e.Body.TenantId)
    if err != nil {
        l.WithError(err).Errorf("DATA_UPDATED with malformed tenantId [%s]; ignoring.", e.Body.TenantId)
        eventsConsumerErrorsTotal.WithLabelValues("parse").Inc()
        return
    }

    // Resolve the tenant.Model. The TenantHeaderParser populated ctx; in the
    // (extremely rare) case of header/body disagreement, we prefer the body
    // and warn — the body is the canonical contract per PRD §4.8.
    headerTenant, hasHeader := tenant.FromContext(ctx)
    if hasHeader && headerTenant.Id() != tenant.Id(bodyTid) {
        l.Warnf("Tenant header [%s] disagrees with event body tenant [%s]; using body.",
            headerTenant.Id(), bodyTid)
    }
    var t tenant.Model
    if hasHeader && headerTenant.Id() == tenant.Id(bodyTid) {
        t = headerTenant
    } else {
        // Build a minimal tenant.Model from the body alone. region/version are
        // unknown here; this path is for header-disagreement only and the
        // wrapper does not depend on those fields for FlushTenant.
        t, _ = tenant.Create(bodyTid, "", 0, 0)
    }

    deleted, ferr := information.FlushTenant(ctx, t)
    if ferr != nil {
        l.WithError(ferr).Errorf("Monster data cache flush partially failed for tenant [%s] (deleted [%d] keys before error).", bodyTid, deleted)
        eventsConsumerErrorsTotal.WithLabelValues("flush").Inc()
    }
    keysDeletedTotal.WithLabelValues(bodyTid.String()).Add(float64(deleted))
    eventsProcessedTotal.WithLabelValues(WorkerMonster, EventTypeDataUpdated, "flushed").Inc()
    l.Debugf("Flushed [%d] monster data cache keys for tenant [%s] in response to DATA_UPDATED.", deleted, bodyTid)
}

func consumerEnabled() bool {
    v, ok := os.LookupEnv("DATA_EVENTS_CONSUMER_ENABLED")
    if !ok {
        return true
    }
    enabled, err := strconv.ParseBool(v)
    if err != nil {
        return true
    }
    return enabled
}
```

**Note on `tenant.Create` fallback path:** the body-only construction is only reachable if header/body disagree (a code bug; should never happen in practice). `FlushTenant` reads `posReg.Clear(ctx, t)` which uses `t.Id()`, `t.Region()`, `t.MajorVersion()`, `t.MinorVersion()` to build the SCAN pattern. **A region/version mismatch would target a different tenant-scoped key prefix**, missing the actual cache. Mitigation: this branch fires only on the WARN'd disagreement case and we log it loudly; if it ever fires in production, the operator can re-run the import. We optimize for the 99.99% case (header agrees with body) and accept the dev-friendly fallback.

If the plan team wants stricter behavior, an alternative is "fail closed": if the header is missing or disagrees, skip the flush and increment `consumer_errors_total{kind="parse"}`. Decision deferred to plan time; default behavior in this design is the dev-friendly path.

### 7.2 `atlas-maps` — `kafka/consumer/data/`

Mirrors the atlas-monsters structure exactly. Differences:

```go
// kafka.go
const WorkerMap = "MAP"
```

```go
// handler.go
func handleDataUpdated(l logrus.FieldLogger, ctx context.Context, e event[dataUpdatedEventBody]) {
    if e.Type != EventTypeDataUpdated {
        eventsSkippedTotal.WithLabelValues("unknown_type").Inc()
        return
    }
    if e.Body.Worker != WorkerMap {
        eventsSkippedTotal.WithLabelValues("unrelated_worker").Inc()
        return
    }

    tid, err := uuid.Parse(e.Body.TenantId)
    if err != nil {
        l.WithError(err).Errorf("DATA_UPDATED with malformed tenantId [%s]; ignoring.", e.Body.TenantId)
        eventsConsumerErrorsTotal.WithLabelValues("parse").Inc()
        return
    }

    deleted, ferr := monster.GetRegistry().FlushTenant(ctx, l, tid)
    if ferr != nil {
        l.WithError(ferr).Errorf("Spawn-registry flush partially failed for tenant [%s] (deleted [%d] keys before error).", tid, deleted)
        eventsConsumerErrorsTotal.WithLabelValues("flush").Inc()
    }
    keysDeletedTotal.WithLabelValues(tid.String()).Add(float64(deleted))
    eventsProcessedTotal.WithLabelValues(WorkerMap, EventTypeDataUpdated, "flushed").Inc()
    l.Debugf("Flushed [%d] spawn-registry keys for tenant [%s] in response to DATA_UPDATED.", deleted, tid)
}
```

**No header/body cross-check needed for atlas-maps:** `SpawnPointRegistry.FlushTenant` takes only `uuid.UUID` (it doesn't use region/version because the spawn registry's hand-rolled key shape doesn't include them — see `registry.go:60`). So the body's `TenantId` alone is sufficient.

`WorkerMap = "MAP"` is the only worker filter (see §10.2 for the rationale that `Worker == MONSTER` does not require a spawn-registry flush).

### 7.3 Per-Service Metric Wiring

Both consumers need:

```go
// metrics.go in each consumer package
var (
    eventsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_<svc>_data_events_processed_total",
        Help: "DATA_UPDATED events processed by the cache-invalidation consumer.",
    }, []string{"worker", "type", "action"})

    eventsConsumerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_<svc>_data_events_consumer_errors_total",
        Help: "Errors encountered processing DATA_UPDATED events.",
    }, []string{"kind"})

    eventsSkippedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_<svc>_data_events_consumer_skipped_total",
        Help: "DATA_UPDATED events skipped (unknown type or unrelated worker).",
    }, []string{"reason"})

    keysDeletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_<svc>_data_events_keys_deleted_total",
        Help: "Redis keys deleted by cache-invalidation flushes, by tenant.",
    }, []string{"tenant"})
)
```

Cardinality budget per service: `worker` ≤ 17, `type` ≤ ~5, `action` ≤ 2, `kind` ≤ 2, `reason` ≤ 2, `tenant` ≤ 5. Trivial.

### 7.4 Consumer Tests

Both consumer packages get a `handler_test.go` covering:

| Test | Both | atlas-monsters specific | atlas-maps specific |
|---|---|---|---|
| `Type != "DATA_UPDATED"` skipped | ✓ |  |  |
| `Worker` mismatch skipped | ✓ | (Worker=MAP skipped) | (Worker=MONSTER skipped, NPC skipped) |
| Malformed `TenantId` errors-and-continues | ✓ |  |  |
| Happy-path triggers FlushTenant | ✓ | `information.FlushTenant` called once with correct tenant; both namespaces cleared | `SpawnPointRegistry.FlushTenant` called once with parsed tenant |
| `DATA_EVENTS_CONSUMER_ENABLED=false` skips registration | ✓ |  |  |
| Tenant isolation (miniredis) |  | Flushing tenant A leaves tenant B's keys in both namespaces | Flushing tenant A leaves `atlas:maps:spawn:B:*` keys intact |
| Flush failure (Redis error) handling | ✓ | partial deletion logged; counter incremented; offset commits | partial deletion logged; counter incremented; offset commits |

Tenant-isolation tests use `miniredis` (already used elsewhere — verified at `services/atlas-monsters` will gain it via task-060 v2; `atlas-maps` may need to add it as a test dep).

### 7.5 Wiring in `main.go`

For both `atlas-monsters/main.go` and `atlas-maps/main.go`:

```go
import data2 "atlas-<svc>/kafka/consumer/data"

const (
    consumerGroupId           = "Monster Registry Service" // existing
    dataEventsConsumerGroupId = "Monster Data Cache Invalidator" // NEW (atlas-monsters; mutatis mutandis for atlas-maps)
)

// in main():
data2.InitConsumers(l)(cmf)(dataEventsConsumerGroupId)
if err := data2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
    l.WithError(err).Fatal("Unable to register data-events kafka handlers.")
}
```

`InitConsumers` is invoked unconditionally; the kill-switch check happens in `InitHandlers` so the consumer registration goes away cleanly when disabled (no orphan consumer subscribed to a topic with no handler).

---

## 8. Configuration & Deploy

### 8.1 Env Vars Added

| Env var | Service | Default | Purpose |
|---|---|---|---|
| `EVENT_TOPIC_DATA` | atlas-data, atlas-monsters, atlas-maps | (must be set) | Kafka topic name for data lifecycle events. |
| `DATA_EVENTS_PRODUCER_ENABLED` | atlas-data | `true` | Producer kill-switch. |
| `DATA_EVENTS_CONSUMER_ENABLED` | atlas-monsters, atlas-maps | `true` | Per-consumer kill-switch. |

### 8.2 Deploy ConfigMap Edit

`deploy/k8s/env-configmap.yaml` — single-line addition under the `EVENT_TOPIC_*` block:

```yaml
EVENT_TOPIC_DATA: "EVENT_TOPIC_DATA"
```

The shared `atlas-env` ConfigMap is referenced by every service's deployment via `envFrom: configMapRef.name: atlas-env`, so this single edit lights up the variable in `atlas-data`, `atlas-monsters`, `atlas-maps`, and any future consumer simultaneously.

`deploy/compose/.env.example` — single-line addition:

```
EVENT_TOPIC_DATA=EVENT_TOPIC_DATA
```

### 8.3 Topic Bootstrap

The repo does not contain a Kafka cluster manifest; existing topics rely on broker-side auto-create-topics (the `COMMAND_TOPIC_DATA` topic is created on first produce). `EVENT_TOPIC_DATA` follows the same convention.

**Risk:** if the production broker disables auto-create, this task ships a producer that will fail silently (Kafka returns "unknown topic" → producer's WARN log fires forever, no events flow). Mitigation: an operator-facing check is added to the runbook in §11. This is identical to the operational risk every other dynamic topic in the repo carries today, so we don't gate the task on it.

### 8.4 No Changes To

- `atlas-data` REST API (no surface changes per PRD §5).
- Any service's HTTP routes.
- Any database schema (no migrations).
- The existing manual `redis-cli --scan ... | xargs DEL` and `DEL atlas:maps:spawn:*` runbook (remains as fallback).

---

## 9. Cross-Cutting Concerns

### 9.1 Multi-Tenant Isolation

Two independent isolation guarantees:

1. **Producer:** `dataUpdatedEventProvider` takes `tenantId` as a string parameter resolved from `tenant.MustFromContext(ctx)` inside `StartWorker`. There is no path that emits an event with a different tenant than the one in the ctx — verified by the `TestStartWorker_EmitsOnSuccess` test asserting `body.tenantId == ctx tenant`.
2. **Consumer:** the handler parses `e.Body.TenantId` and passes the parsed UUID to `FlushTenant`. The handler MUST NOT fall back to "flush all tenants" on parse error — the `parse_error` branch returns early with no flush. Verified by `TestHandleDataUpdated_MalformedTenantId_NoFlush`.

The PRD's §4.8 cross-check ("Kafka header tenant agrees with body tenant") is implemented as a WARN log only, not an enforcement gate. Header tenant vs body tenant disagreement is a code bug, not a runtime hazard — log it loudly, prefer the body, move on.

### 9.2 Idempotency

A duplicate `DATA_UPDATED` event causes:

- atlas-monsters: a second `Clear` on already-empty Redis namespaces → returns `(0, nil)` per namespace. The SCAN finds nothing; no DEL is pipelined. ~1 round-trip of meaningful work (the SCAN cursor exchange). Functionally a no-op.
- atlas-maps: same — a second `SCAN`/`DEL` on an already-empty key range. No-op-equivalent.

No deduplication tokens, no event ids, no idempotency keys. Per PRD §2, this is by design.

### 9.3 Race Cleanliness

- `TenantRegistry.Clear` holds no in-process lock; concurrency is mediated by the `goredis.Client`'s pool semantics. SCAN+DEL races against concurrent `Put`/`PutWithTTL` are by design "delete everything that was there at scan time"; new puts during the flush survive.
- atlas-maps' `SpawnPointRegistry.FlushTenant` is goroutine-safe by virtue of the underlying `goredis.Client`'s pool semantics; no shared state in the registry struct beyond the client pointer.
- All new code paths get `go test -race` coverage in CI.

### 9.4 Order of Events

Tenant T's events are key-partitioned by tenant id, so all `DATA_UPDATED` events for T land on the same partition and are delivered to the consumer in emission order. With a shared single-delivery group, that consumer gets them strictly ordered.

Cross-tenant order is irrelevant.

### 9.5 Backpressure / Rate Limiting

No special handling. The producer emits at most one event per worker-completion, and a full data import emits ≤ 17 events per tenant (one per worker). The consumer's flush is fast (~ms-scale Redis SCAN). No backlog accumulation expected.

If the topic were ever heavily reused (e.g. for `DATA_IMPORT_STARTED` events at byte rate), we'd revisit. Out of scope here.

### 9.6 Stale-Read Window During Flush

A flush is not atomic across all keys (pipelined DEL in batches of 100). During the ~tens-of-ms window of a large-tenant flush, callers may read a partially-cleared cache: some keys still hit the old value, others miss and re-fetch from `atlas-data`. **This is acceptable** — partial inconsistency for ms-scale durations is no worse than the steady-state stale window between an upstream change and the next event. Consistency converges as the flush completes.

If sub-ms cross-key consistency were ever required, the alternative would be a Lua-scripted SCAN+DEL — which blocks the entire Redis broker. The cure is worse than the disease.

---

## 10. Decisions on PRD §9 Open Questions

### 10.1 Topic Name Finalized as `EVENT_TOPIC_DATA`

Already pinned by PRD §9. No further decision; design uses it consistently.

### 10.2 Atlas-Maps Drops `WorkerMonster` Branch

Originally the v1 PRD conservatively flushed atlas-maps spawn registry on both `Worker=MAP` and `Worker=MONSTER`, with a note to verify at design time.

**Verified against `services/atlas-maps/atlas.com/maps/map/monster/registry.go`:** the `storedSpawnPoint` struct (lines 18-32) holds only spawn-point geometry and timing (`Id`, `Template`, `MobTime`, `Cy`, `F`, `Fh`, `Rx0`, `Rx1`, `X`, `Y`, `NextSpawnAt`). The `Template` field is a monster id reference — but the registry stores nothing **about** the monster (no HP, no exp, no drops). Those attributes live in atlas-data and are looked up at spawn time via the regular HTTP path.

The `MobTime` field (respawn cooldown) is part of the **map's** spawn-point definition (`Map.wz`), populated by `WorkerMap`, not `WorkerMonster`.

**Conclusion:** atlas-maps' spawn registry depends only on map data. **Drop `Worker=MONSTER` from atlas-maps' filter.** Atlas-maps flushes only on `Worker=MAP`.

This eliminates a redundant flush on every monster re-import (which is the more common operational case), reducing Redis load.

The acceptance criterion in PRD §10 reflects this: "On `DATA_UPDATED` with `Worker=MAP`, the spawn-point registry for `TenantId` is flushed." Other workers (including `MONSTER`) are ignored.

### 10.3 SCAN+pipelined-DEL, not Lua-scripted Atomic SCAN+DEL

PRD §9 lists this as open. Pinned in §3.2 above. Lua atomicity would block the entire Redis broker for the duration of the script. Cooperative SCAN+pipelined-DEL is the right tradeoff for the bounded keyspace in play.

If a future tenant ever exceeds ~10 k keys per namespace in a way that makes the cooperative path's tail latency unacceptable, revisit. Not a near-term concern.

### 10.4 `SpawnPointRegistry` Migration to `TenantRegistry`

PRD §9 lists this as open. **Pinned out of scope.** The hand-rolled key shape is functional; migrating it is a refactor with no behavioral payoff. We add a parallel `FlushTenant` to `SpawnPointRegistry` and accept the duplication. A future cleanup task can unify if motivated.

### 10.5 Topic Bootstrap = Auto-Create

Pinned in §8.3. The repo has no Kafka manifest; auto-create is the de facto convention.

### 10.6 Consumer Commit Semantics = Library Default (Commit-After-Handler)

Verified by reading `libs/atlas-kafka/message/handler.go` and `libs/atlas-kafka/consumer/manager.go`: handlers are fire-and-forget; the consumer driver commits the offset after the handler returns regardless of internal handler errors (which are logged but not propagated back to the consumer loop).

This matches PRD §4.9's "commit-and-continue" requirement for parse and flush failures. **No special handling needed; the handler simply logs and returns.**

### 10.7 Future Event Types = Out of Scope

PRD §9 already noted this; design just confirms the discriminator field is present and switched on correctly.

### 10.8 In-Process Cache Forward Compatibility

PRD §9 raises this. Pinned in §6.3: shared groups today; if a future cache cannot live in Redis, that consumer adds a per-pod `consumer.PerPodGroup` helper at the time. Don't ship the helper now.

---

## 11. Operational Notes

### 11.1 Healthy-State Indicators

| Question | Query | Healthy |
|---|---|---|
| Producer emitting? | `rate(atlas_data_events_emitted_total[5m])` | Non-zero during/after data imports. |
| Producer failing? | `rate(atlas_data_events_emit_failures_total[5m])` | Always 0 in steady state. |
| Consumer keeping up? | Compare emit rate vs. `rate(atlas_<svc>_data_events_processed_total{action="flushed"}[5m])` | Processed rate ≈ emit rate. |
| Flushes succeeding? | `rate(atlas_<svc>_data_events_consumer_errors_total{kind="flush"}[5m])` | Always 0 in steady state. |
| Flushes deleting expected key counts? | `rate(atlas_<svc>_data_events_keys_deleted_total[5m])` | Non-zero during/after data imports for any tenant with cached data. |

### 11.2 Deploy Verification Runbook

Post-deploy, run a smoke test:

1. Hit a `GET /api/data/monsters/100100` for tenant T (any monster id known to exist) several times to populate the positive cache. Verify `redis-cli --scan --pattern 'atlas:monsters:cache:data:<tenantKey>:*'` returns at least one key.
2. Trigger a `START_WORKER` for `WorkerMonster` against tenant T.
3. Within 5 s:
   - `atlas_monsters_data_events_processed_total{worker="MONSTER", action="flushed"}` increments by 1.
   - `redis-cli --scan --pattern 'atlas:monsters:cache:data:<tenantKey>:*'` returns empty.
4. Hit `GET /api/data/monsters/100100` again. Verify the key reappears in Redis (cache re-populates from upstream).
5. Trigger a `START_WORKER` for `WorkerMap` against tenant T.
6. Within 5 s:
   - `atlas_maps_data_events_processed_total{worker="MAP", action="flushed"}` increments by 1.
   - `redis-cli --scan --pattern 'atlas:maps:spawn:T:*'` returns empty.
7. Re-activate any map for tenant T; `atlas:maps:spawn:T:*` keys reappear.

The existing manual `redis-cli DEL` runbook (per the user's `reference_atlas_maps_spawn_cache.md` memory) is **demoted to fallback**: only run it if the smoke test fails or `atlas_data_events_emit_failures_total` is rising.

### 11.3 Failure Modes

| Failure | Symptom | Action |
|---|---|---|
| Producer cannot reach Kafka | `atlas_data_events_emit_failures_total` rising; data imports still succeed. | Investigate Kafka health. Caches will still expire via TTL. Falls back to manual flush runbook if urgent. |
| Consumer cannot reach Redis | `atlas_<svc>_data_events_consumer_errors_total{kind="flush"}` rising; cache keys remain stale. | Investigate Redis health. Manual `DEL` runbook still works as bypass. |
| Kafka broker disables auto-create-topics | First producer call returns "unknown topic"; no events flow. | Operator creates the topic manually with default partition/replication settings. |
| Consumer group rebalance during deploy | Brief processing pause; events delivered after rebalance completes. | Normal Kafka behavior; no action. |
| Header/body tenant disagreement | WARN log; flush proceeds against body tenant (with possibly-empty region/version). | Investigate as a producer-side bug. Re-run import to recover. |

### 11.4 Archived: `consumer.PerPodGroup` Sketch (Not Shipped)

For posterity, in case a future task needs per-pod fan-out:

```go
// libs/atlas-kafka/consumer/group.go (NOT created in this task; sketch only)
func PerPodGroup(l logrus.FieldLogger, service, suffix string) string {
    h := os.Getenv("HOSTNAME")
    if h == "" {
        h = startupUUID() // sync.Once-guarded, with WARN
    }
    return fmt.Sprintf("%s %s %s", service, suffix, h)
}
```

If a future in-process cache consumer needs this, lift the sketch from here, drop it into `libs/atlas-kafka/consumer/group.go` with tests, and adopt at the call site.

---

## 12. Alternatives Considered

### 12.1 Per-Pod Consumer Groups (v1 Default)

**Considered:** every replica of every consumer service uses a unique consumer-group id (suffixed with `HOSTNAME`) so every pod receives every event.

**Rejected because:** with task-060 v2's pivot to shared Redis, no replica owns its own cache state. A single pod's `Clear` is visible to all. Per-pod fan-out would have N replicas all racing to do the same SCAN+DEL on the same Redis keys. Wasted work, more consumer-group-coordinator load, and group-id sprawl in Kafka tooling.

The argument for keeping per-pod groups despite v2 is "future-proofing for in-process caches." Rejected as YAGNI: there is no in-process cache planned, and the helper is a small future addition (~25 LOC) when actually needed. See §11.4 for the archived sketch.

### 12.2 Single Centralized Invalidator Service

**Considered:** stand up a new `atlas-cache-invalidator` service that subscribes to `EVENT_TOPIC_DATA` and calls `POST /admin/cache/flush` on each consumer service.

**Rejected because:**

- Adds a second service to the deploy (more YAML, more pods, more failure modes).
- Requires every consumer service to expose an admin HTTP endpoint with cache-flush authorization — new attack surface.
- Doesn't simplify anything: the invalidator would still need to know which services own which workers' caches.

The Kafka-fanout-with-shared-group solution is simpler, has no new service, and reuses a transport every consumer already speaks.

### 12.3 Selective Per-Id Invalidation

**Considered:** include the affected ids in the event body, e.g. `{"tenantId": "...", "worker": "MONSTER", "ids": [100100, 100101, ...]}` so consumers can `Remove(id)` instead of `Clear` the namespace.

**Rejected because:**

- The trigger is whole-data-set re-import. The producer doesn't know which ids changed.
- Computing the diff is its own complex operation and doesn't pay back: the data sets are bounded (a few thousand monsters); a Clear + re-fetch on next access is fast.
- Selective invalidation would force the event payload to grow with set size — bad for Kafka log size.

PRD §2 explicitly rules this out. Confirmed.

### 12.4 Etag / Version-Based Polling

**Considered:** add a `GET /api/data/version` endpoint that returns a tenant-data version token. Consumers poll periodically and flush when the token changes.

**Rejected because:**

- Polling is wasteful; eventing is push.
- Adds latency (poll interval lower bound, typically 30s+).
- Doesn't get rid of the Kafka topic — we'd still want a push channel for "version changed."

The Kafka event approach has none of these costs.

### 12.5 Compacted Kafka Topic with `tenantId|worker` as Key

**Considered:** make `EVENT_TOPIC_DATA` log-compacted so each tenant-worker's last invalidation event is retained. A starting consumer would consume the compacted log and replay the latest invalidation for each tenant-worker, eliminating the "missed events during deploy" risk.

**Rejected because:**

- Compaction needs careful key design (a composite `tenantId|worker` key changes the partition assignment story).
- The "missed events during deploy" risk is small: deploy is a rare event; a missed event means one extra round of stale reads until TTL or the next import. Acceptable.
- Compacted topics have heavier broker config requirements; aligning that with the existing Kafka ops story is its own subtask.

If the operational pain becomes real, this is a follow-up — same producer, same body, change of topic config. Not gated here.

### 12.6 Lua-Scripted Atomic SCAN+DEL in `TenantRegistry.Clear`

**Considered:** use a single Lua script to enumerate and delete all tenant-prefixed keys atomically.

**Rejected because:** Lua scripts on Redis block the entire server for their duration. atlas-monsters' Redis is shared with cooldown updates, monster registry I/O, and drop timers; a 100ms broker stall would visibly disrupt gameplay. Cooperative SCAN+pipelined-DEL keeps the broker available throughout. See §3.2.

### 12.7 `libs/atlas-cache` In-Process Cache (v1 Plan)

**Considered:** the v1 task-060 plan, which would have introduced `libs/atlas-cache.Cache[K,V]` with `Flush()` semantics.

**Rejected because:** the v1 plan was reverted in task-060 (commits `e983a009a`, `29c6f9603`) in favor of the v2 Redis-backed approach. With no in-process cache, there is nothing for an in-process `Flush()` to clear. This task aligns with v2.

---

## 13. Test Plan Summary

Five test harnesses, all `go test -race` clean:

| Harness | Lives in | Tests |
|---|---|---|
| `libs/atlas-redis` | `tenant_registry_test.go` (extension) | §3.3 (6 tests) |
| `atlas-monsters` cache wrapper | `monster/information/cache_test.go` (extension) | §4.1 (4 tests) |
| `atlas-maps` registry | `map/monster/registry_test.go` (or `flush_test.go`) | §4.2 (4 tests) |
| `atlas-data` producer | `data/producer_test.go` (new), `data/processor_test.go` extension | §5.5 (6 tests) |
| `atlas-monsters` consumer | `kafka/consumer/data/handler_test.go` (new) | §7.4 (consumer behavior + isolation + errors) |
| `atlas-maps` consumer | `kafka/consumer/data/handler_test.go` (new) | §7.4 (consumer behavior + isolation + errors) |

Plus the §11.2 manual smoke test against a deployed cluster, gated by the acceptance criteria in PRD §10.

---

## 14. What This Design Deliberately Does Not Do

- Does not introduce `libs/atlas-cache`. The v1 plan's library is reverted; no in-process cache exists.
- Does not introduce `consumer.PerPodGroup`. Shared groups suffice given task-060 v2's Redis-backed cache. Sketch archived in §11.4 for future use.
- Does not add caches to `atlas-channel`, `atlas-pets`, `atlas-monster-death`, etc. Each is a follow-up task that wires its own consumer using the same `TenantRegistry.Clear` library affordance.
- Does not introduce dedup tokens, event ids, or idempotency keys.
- Does not change `atlas-data`'s import logic, file format, or REST API.
- Does not retire the existing manual `DEL` runbook — only demotes it to fallback.
- Does not add an admin endpoint for manual invalidation. Operators trigger by re-importing.
- Does not gate on cross-cluster propagation; single-cluster Kafka assumption matches the rest of the project.
- Does not introduce a new compacted topic or change the broker's auto-create-topics behavior.
- Does not retrofit the `atlas-monsters`/`atlas-maps` existing fixed-group consumers. Only the new `data-events` consumer is added; everything else is unchanged.
- Does not migrate `SpawnPointRegistry` to `TenantRegistry`. Out of scope; the hand-rolled key shape persists with a parallel `FlushTenant` method.
- Does not implement future event types (`DATA_IMPORT_STARTED`, `DATA_IMPORT_FAILED`). The discriminator field is present so future tasks can add them without breaking consumers.
