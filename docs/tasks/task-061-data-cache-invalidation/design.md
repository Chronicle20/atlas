# task-061 — atlas-data Cache Invalidation: Architecture & Tradeoffs

Status: Draft
Created: 2026-05-08
Builds on: PRD `prd.md`; depends on task-060 (`libs/atlas-cache` + `atlas-monsters` per-tenant registry).

This document fixes the architectural choices that the PRD left open at §9 and pins down concrete file paths, function signatures, and test surfaces. It is the contract that `plan.md` will decompose into discrete tasks.

---

## 1. Goals of This Document

1. Decide where each piece of new code lives, given the conventions task-060 establishes (per-tenant registry as a service-side wrapper, not in `libs/atlas-cache`).
2. Pick concrete signatures for the new producer (`atlas-data`), the two new consumers (`atlas-monsters`, `atlas-maps`), and the new `libs/atlas-cache` flush API.
3. Resolve PRD §9 open questions with stated rationale.
4. Specify the per-pod consumer-group convention as a reusable helper in `libs/atlas-kafka` (rather than hand-rolled in each service) so future caching follow-ups can adopt it in <10 LOC.
5. Document the cross-cutting concerns — tenant isolation, error handling, observability, kill-switch behavior — that the plan will turn into tests.

---

## 2. Architecture Overview

```
                                 Kafka EVENT_TOPIC_DATA
                                 key = tenantId, partitions = N
                                       ▲                ▲
                                       │ produces       │ subscribes (per-pod group)
                                       │                │
   ┌───────────────────────┐     ┌─────┴─────┐    ┌─────┴────────┐
   │  atlas-data           │     │ atlas-data│    │ atlas-monsters
   │  StartWorker(name)    │────▶│ producer  │    │ atlas-maps    
   │  on success per       │     └───────────┘    │  (consumers)  
   │  (tenant, worker)     │                      └───────┬───────┘
   └───────────────────────┘                              │
                                                          │ filter on
                                                          │ event.body.worker
                                                          ▼
                                          ┌──────────────────────────┐
                                          │ atlas-monsters:          │
                                          │   monster/information.   │
                                          │   FlushTenant(tenantId)  │
                                          │ atlas-maps:              │
                                          │   map/monster.SpawnReg   │
                                          │   istry.FlushTenant(...) │
                                          └──────────────────────────┘
```

**Three layers, each with one responsibility:**

| Layer | Responsibility | Lives in |
|---|---|---|
| Library primitive | In-process TTL cache; offers `Flush()` to clear all entries atomically. Knows nothing about tenants, topics, or Kafka. | `libs/atlas-cache` |
| Service-side wrapper | Per-tenant registry of `Cache[K,V]` instances; exposes `FlushTenant(tenant.Id)`. Each consumer service has its own. | `services/<svc>/atlas.com/<svc>/<domain>/cache.go` |
| Event plumbing | Producer in `atlas-data` emits `DATA_UPDATED`; consumers in `atlas-monsters` and `atlas-maps` subscribe and call `FlushTenant`. | `services/atlas-data/atlas.com/data/data/{kafka.go,producer.go,processor.go}` and `services/<svc>/atlas.com/<svc>/kafka/consumer/data/` |

The library knows nothing about events. The wrappers know nothing about Kafka. The consumers know nothing about cache internals. Each layer can be tested in isolation.

---

## 3. `libs/atlas-cache` — `Flush()` Addition

Task-060 lands the `Cache[K, V]` interface (`Get`, `Put`, `PutNegative`, `IsNegative`, `Delete`, `Len`) and the `Config{TTL, NegativeTTL, Now, OnEviction(kind string)}` shape. This task adds **one method** and **one extension to `OnEviction`'s `kind` enumeration**.

### 3.1 Public API Addition

```go
type Cache[K comparable, V any] interface {
    // existing methods from task-060...

    // Flush atomically discards every entry (positive and negative).
    // After Flush, Len() returns (0, 0) until subsequent Put/PutNegative
    // calls. OnEviction is invoked once per discarded entry with kind
    // "invalidation" so callers can attribute evictions correctly.
    Flush() (positive int, negative int)
}
```

**Why return counts:** the service-side wrapper's metric (`atlas_<svc>_data_cache_evictions_total{reason="invalidation"}`) needs to know how many entries were evicted in aggregate. We could compute it in the wrapper by calling `Len()` then `Flush()`, but that's two passes under separate locks and racy — a `Put` between the two calls would inflate the count. Returning the count from inside the write-locked section is one pass and accurate.

### 3.2 OnEviction `kind` Enumeration

Task-060 defines `kind ∈ {"positive", "negative"}` for lazy-expiration evictions. This task extends:

| `kind` value | Source | Introduced in |
|---|---|---|
| `"positive"` | Lazy expiration of a positive entry on a `Get` past TTL | task-060 |
| `"negative"` | Lazy expiration of a negative entry on a `IsNegative` past NegativeTTL | task-060 |
| `"invalidation"` | Explicit `Flush()` call | **task-061** |

Service-side wrappers route `"invalidation"` to the `reason` label of their existing eviction counter (no new counter needed; new label value).

### 3.3 Implementation Sketch

```go
func (c *cache[K, V]) Flush() (positive, negative int) {
    c.mu.Lock()
    defer c.mu.Unlock()
    for _, e := range c.entries {
        if e.negative {
            negative++
        } else {
            positive++
        }
    }
    // Replace the map rather than iterate-and-delete: O(1) clear, no
    // lingering capacity on a large map after a flush.
    c.entries = make(map[K]entry[V])
    if c.cfg.OnEviction != nil {
        // Emit one event per evicted entry so counter math stays
        // consistent with lazy-expiration semantics.
        for i := 0; i < positive; i++ {
            c.cfg.OnEviction("invalidation")
        }
        for i := 0; i < negative; i++ {
            c.cfg.OnEviction("invalidation")
        }
    }
    return
}
```

**Why one OnEviction call per entry, not one per `Flush`:** the existing counter `atlas_<svc>_data_cache_evictions_total{reason}` is a counter of *entries evicted*, not *flush operations*. Treating Flush as a single eviction event would silently break the existing TTL-based dashboard math. We accept the O(N) callback cost — N is bounded by tenant-local entity count (≤ a few thousand) and Flush is rare.

**Concurrency:** the implementation holds the write lock across the entire scan + replace + callback phase. This is safe because `OnEviction` MUST NOT call back into the cache (documented in task-060's contract). The wrapper-side OnEviction is a `prometheus.CounterVec.Inc()` call — non-blocking, lock-free.

### 3.4 Tests Added in `libs/atlas-cache`

| Test | Asserts |
|---|---|
| `TestCacheFlush_EmptiesPopulatedCache` | After `Put(k, v); Flush()`, `Get(k)` returns `(zero, false)`; `Len() == (0, 0)`. |
| `TestCacheFlush_ReturnsAccurateCounts` | After 3 positive + 2 negative entries, `Flush()` returns `(3, 2)`. |
| `TestCacheFlush_InvokesOnEvictionInvalidation` | A `Cache` configured with a counting `OnEviction` records exactly `positive+negative` "invalidation" events. |
| `TestCacheFlush_RaceCleanWithGetPut` | `go test -race`: 8 goroutines doing mixed Get/Put/PutNegative/Flush for 1 s; no race detector hits, no panics. |
| `TestCacheFlush_NoEntriesIsZero` | Calling Flush on an empty cache returns `(0, 0)` and emits no eviction callbacks. |

These tests live in `libs/atlas-cache/cache_test.go` (extending task-060's table-driven test file) using the same `cache.Config{Now: fakeNow}` injection pattern.

---

## 4. Service-Side Wrappers: Per-Tenant `FlushTenant`

Task-060's design (§4.2) places the per-tenant registry in service code, not in `libs/atlas-cache`, because:

- Different services hold different value types (`Model` for monsters; future `MapModel` for atlas-channel; etc.) — generics force the registry into a per-service location anyway.
- Different services have different metric labels and kill-switch env vars.
- The registry is responsible for OnEviction-to-Prometheus wiring, which is service-specific.

This task **does not change that structure**. It adds one new free function per service that has a per-tenant registry: `FlushTenant(t tenant.Id)`.

### 4.1 `services/atlas-monsters/.../monster/information/cache.go` — `FlushTenant`

```go
// FlushTenant clears the per-tenant monster-information cache for t.
// If the cache is disabled (MONSTER_DATA_CACHE_ENABLED=false), this is a
// no-op. If the tenant has no live cache (no GetById call has touched
// it yet), this is a no-op — we deliberately do NOT allocate a registry
// entry here, because allocating-on-flush would be a memory amplifier
// for a poison-message payload.
func FlushTenant(t tenant.Id) {
    initOnce.Do(loadConfig)
    if !cacheCfg.enabled {
        return
    }

    tenantCachesMu.RLock()
    c, ok := tenantCaches[uuid.UUID(t)]
    tenantCachesMu.RUnlock()
    if !ok {
        return
    }

    pos, neg := c.Flush()
    flushesTotal.WithLabelValues(uuid.UUID(t).String()).Inc()
    // OnEviction has already incremented evictionsTotal{reason="invalidation"}
    // pos+neg times via the cache callback; nothing to do here.
    _ = pos
    _ = neg
}
```

**Why "no-op if not allocated":** the consumer fires `FlushTenant` for **every** `DATA_UPDATED` event with `Worker=MONSTER`. If the cache for tenant T was never read, there's nothing to flush — and we do not want to instantiate a per-tenant `Cache[uint32, Model]` purely to clear it.

**Why kill-switch path returns before lookup:** preserves the task-060 promise that disabling the cache costs zero memory.

**New metric:** `atlas_monsters_data_cache_flushes_total{tenant}` — counter incremented per `FlushTenant` call (regardless of whether anything was evicted). Distinguished from per-entry `evictionsTotal{reason="invalidation"}` because operators want to see "did the consumer fire?" separate from "how many entries did it clear?"

### 4.2 `services/atlas-maps/.../map/monster/registry.go` — `FlushTenant`

Atlas-maps' spawn-point registry is Redis-backed, not in-process. The flush is `SCAN` + `DEL` on `atlas:maps:spawn:{tenantId}:*`. The existing `Reset(ctx)` method at `registry.go:259-264` is the model — it scans the global pattern; we add a tenant-scoped variant:

```go
// FlushTenant deletes every spawn-point hash for tenantId.
// Uses SCAN with COUNT to avoid blocking the broker on large key spaces;
// pipelines DEL per batch. Errors are logged but do not abort the sweep.
func (r *SpawnPointRegistry) FlushTenant(ctx context.Context, l logrus.FieldLogger, tenantId uuid.UUID) (deleted int, err error) {
    pattern := fmt.Sprintf("atlas:maps:spawn:%s:*", tenantId.String())
    iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
    pipe := r.client.Pipeline()
    pipeSize := 0
    for iter.Next(ctx) {
        pipe.Del(ctx, iter.Val())
        deleted++
        pipeSize++
        if pipeSize >= 100 {
            if _, perr := pipe.Exec(ctx); perr != nil {
                l.WithError(perr).Warnf("Partial spawn-registry flush failure for tenant [%s].", tenantId)
                err = perr
            }
            pipe = r.client.Pipeline()
            pipeSize = 0
        }
    }
    if pipeSize > 0 {
        if _, perr := pipe.Exec(ctx); perr != nil {
            l.WithError(perr).Warnf("Final spawn-registry flush batch failure for tenant [%s].", tenantId)
            err = perr
        }
    }
    if ierr := iter.Err(); ierr != nil {
        l.WithError(ierr).Warnf("Spawn-registry SCAN failure for tenant [%s].", tenantId)
        err = ierr
    }
    return deleted, err
}
```

**Why `SCAN` with `COUNT=100` and pipelined `DEL`:** PRD §8.1 caps tenant key counts at "a few thousand at most." A single `KEYS atlas:maps:spawn:{T}:*` would block the Redis server briefly on a large tenant; `SCAN` is the cooperative variant. Pipelining the `DEL` batches keeps round-trips to ≤ N/100.

**Why one `err` accumulator instead of returning immediately on first error:** spawn registry state is not transactional across maps anyway — partial deletion is recoverable on the next event or via TTL fallback. Aborting halfway through leaves a noisy half-flushed state with no benefit. We log per-batch and surface the *last* error to the consumer for metric purposes.

**Tests:** see §7.2.

### 4.3 Why Not Push the Registry Into the Library

This was option B in the design exploration. Rejected:

- The PRD's "registry-level helper FlushTenant on the per-tenant registry pattern" wording implies a library-level helper, but task-060 deliberately did NOT put the registry in the library. Reversing that decision here would force task-060 to land a generic registry it doesn't need, just so this task can call into it.
- Each service's registry differs in metric labels, kill-switch env var, and value type. A library-level `TenantRegistry[K, V]` would have to take metric callbacks via Config — more API surface for one extra line of saved code per service.
- We anticipate ≥ 5 future caching tasks (atlas-channel, atlas-pets, atlas-monster-death, etc.). Each will already need its own service-side wrapper; adding a free `FlushTenant` to that wrapper is one function. A library abstraction would have to be designed for hypothetical needs we don't yet have.

YAGNI: keep the library minimal, repeat the 8-line `FlushTenant` per service.

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

## 6. Per-Pod Consumer-Group Convention

This task introduces a **library-level helper** in `libs/atlas-kafka/consumer` so future consumers can adopt the per-pod-group pattern in one line, and so the `HOSTNAME` fallback logic lives in exactly one place.

### 6.1 New `consumer.PerPodGroup`

```go
// libs/atlas-kafka/consumer/group.go (NEW)

package consumer

import (
    "fmt"
    "os"
    "sync"

    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

var (
    fallbackHostnameOnce sync.Once
    fallbackHostname     string
)

// PerPodGroup returns a consumer-group id that is unique per pod (per
// container, in compose) so that every replica of the service receives
// every message on a fan-out topic. The shape is:
//
//     "<service> <suffix> <hostname>"
//
// where <hostname> is os.Getenv("HOSTNAME") when set, or a process-startup
// UUID otherwise (with a single WARN log).
//
// Use this for topics where every replica must process every message
// (e.g. cache-invalidation events). Do NOT use it for command topics where
// exactly-one-consumer semantics are required.
func PerPodGroup(l logrus.FieldLogger, service, suffix string) string {
    h := os.Getenv("HOSTNAME")
    if h == "" {
        fallbackHostnameOnce.Do(func() {
            fallbackHostname = uuid.NewString()
            l.Warnf("HOSTNAME env var unset; falling back to startup UUID [%s] for per-pod consumer group. Set HOSTNAME in your deploy manifest to suppress this warning.", fallbackHostname)
        })
        h = fallbackHostname
    }
    return fmt.Sprintf("%s %s %s", service, suffix, h)
}
```

**Why a library helper, not a per-service inline:**

1. The `HOSTNAME` fallback (UUID + WARN) is exactly the kind of logic that drifts when copy-pasted. One implementation, one test.
2. Future cache-invalidation consumers (atlas-channel, atlas-pets, atlas-monster-death, ...) all want the same semantics. Saving 6 lines per consumer multiplies.
3. Discovery: `git grep PerPodGroup` immediately surfaces every adopter.

**Why string suffix instead of variadic args:** keeps the call site `consumer.PerPodGroup(l, "atlas-monsters", "data-events")` readable. The suffix lets a single service distinguish multiple per-pod consumers if it ever needs to (today it doesn't).

**Why a global `sync.Once` for the fallback UUID:** within a single process, every call to `PerPodGroup` should resolve to the same id (otherwise the consumer manager would register N distinct groups for N consumer registrations). The `Once` ensures one UUID per process, one warning per process.

### 6.2 Consumer Wiring Pattern

```go
// In atlas-monsters/main.go and atlas-maps/main.go:
const dataEventsGroupSuffix = "data-events"

// existing fixed-group consumer registration:
monster2.InitConsumers(l)(cmf)(consumerGroupId)
_map.InitConsumers(l)(cmf)(consumerGroupId)

// new per-pod-group consumer registration:
dataEventsGroupId := consumer.PerPodGroup(l, serviceName, dataEventsGroupSuffix)
data2.InitConsumers(l)(cmf)(dataEventsGroupId)
```

Each consumer's `InitConsumers` keeps the existing curried signature `func(l)(rf)(consumerGroupId string)` — no library-level coupling needed. The per-pod-ness is invisible to the consumer code; only the `main.go` chooses the group.

### 6.3 Start-Offset = `kafka.LastOffset`

The decorator already exists. Use `consumer.SetStartOffset(kafka.LastOffset)` on the new consumer registration:

```go
rf(
    consumer2.NewConfig(l)("data_events")(EnvEventTopicData)(consumerGroupId),
    consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
    consumer.SetStartOffset(kafka.LastOffset),
)
```

**Why `LastOffset`:** every pod restart spawns a fresh consumer group (because the group id includes a fresh `HOSTNAME`). With `FirstOffset` (the default in `consumer.NewConfig`), each restart would consume the entire topic history — every flush event ever emitted. Wasted work, possibly thousands of flushes on a fresh pod.

`LastOffset` says: start from the tail. Events emitted between pod start and consumer-group-coordinator-readiness may be missed; that's acceptable per PRD §2 ("in-flight events during a deploy may be missed by a starting pod and that is acceptable").

### 6.4 Tests for `PerPodGroup`

| Test | Asserts |
|---|---|
| `TestPerPodGroup_UsesHostnameWhenSet` | With `HOSTNAME=pod-abc123`, returns `"<svc> <suffix> pod-abc123"`. |
| `TestPerPodGroup_FallsBackToStartupUUID` | With `HOSTNAME` unset, returns a string ending in a parseable UUID; emits a single WARN. |
| `TestPerPodGroup_FallbackIsStable` | Two calls within the same process return the same fallback id (same UUID). |
| `TestPerPodGroup_FallbackWarnsOnce` | Multiple calls produce exactly one WARN entry (ring-buffered logger). |

Tests live in `libs/atlas-kafka/consumer/group_test.go` and use a `logrus.New()` with a buffered hook to assert WARN count.

---

## 7. Consumer Wiring

### 7.1 `atlas-monsters` — `kafka/consumer/data/`

New tree mirroring the existing `kafka/consumer/monster/` shape:

```
services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/
├── consumer.go     # InitConsumers / InitHandlers
├── kafka.go        # event[E] + dataUpdatedEventBody + topic env constant
└── handler.go      # handleDataUpdated
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
            // groupId is already a per-pod group from main.go.
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
        t, _ := topic.EnvProvider(l)(EnvEventTopic)()
        if !consumerEnabled() {
            l.Infof("DATA_EVENTS_CONSUMER_ENABLED=false; not registering DATA_UPDATED handler.")
            return nil
        }
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

    tid, err := uuid.Parse(e.Body.TenantId)
    if err != nil {
        l.WithError(err).Errorf("DATA_UPDATED with malformed tenantId [%s]; ignoring.", e.Body.TenantId)
        eventsConsumerErrorsTotal.WithLabelValues("parse").Inc()
        return
    }

    // Cross-check against the tenant header parser (defensive — both should
    // resolve to the same tenant). If they disagree, prefer the body and warn.
    if hdrTenant, ok := tenant.FromContext(ctx); ok && hdrTenant.Id() != tenant.Id(tid) {
        l.Warnf("Tenant header [%s] disagrees with event body tenant [%s]; using body.",
            hdrTenant.Id(), tid)
    }

    information.FlushTenant(tenant.Id(tid))
    eventsProcessedTotal.WithLabelValues(WorkerMonster, EventTypeDataUpdated, "flushed").Inc()
    l.Debugf("Flushed monster information cache for tenant [%s] in response to DATA_UPDATED.", tid)
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

**Note:** `tenant.FromContext` (non-`MustFromContext`) is used because we want the cross-check, not a panic. If `libs/atlas-tenant` does not expose a non-panicking variant, we wrap `MustFromContext` in a recover. Confirm at plan time.

### 7.2 `atlas-maps` — `kafka/consumer/data/`

Mirrors the atlas-monsters structure exactly. The differences are the handler implementation:

```go
// services/atlas-maps/atlas.com/maps/kafka/consumer/data/handler.go
func handleDataUpdated(l logrus.FieldLogger, ctx context.Context, e event[dataUpdatedEventBody]) {
    if e.Type != EventTypeDataUpdated {
        eventsSkippedTotal.WithLabelValues("unknown_type").Inc()
        return
    }
    if e.Body.Worker != WorkerMap {
        // PRD §4.6 conservative default removed by §10 of this design.
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
        // Still record the partial flush as processed so dashboards stay
        // honest about activity. Offset commits on return.
    }
    spawnRegistryEvictionsTotal.WithLabelValues(tid.String(), "invalidation").Add(float64(deleted))
    eventsProcessedTotal.WithLabelValues(WorkerMap, EventTypeDataUpdated, "flushed").Inc()
    l.Debugf("Flushed [%d] spawn-registry keys for tenant [%s] in response to DATA_UPDATED.", deleted, tid)
}
```

`WorkerMap = "MAP"` is the only worker filter (see §10.2 for rationale).

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
)
```

For `atlas-maps`:

```go
spawnRegistryEvictionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
    Name: "atlas_maps_spawn_registry_evictions_total",
    Help: "Spawn-point registry entries removed, by tenant and reason.",
}, []string{"tenant", "reason"})
```

For `atlas-monsters`, the `OnEviction("invalidation")` callback set up in task-060's `cacheFor` already routes through `evictionsTotal{tenant, reason="invalidation"}`. No new metric here; the new label value rides on the existing counter.

### 7.4 Consumer Tests

Both consumer packages get a `handler_test.go` covering:

| Test | Both | atlas-monsters specific | atlas-maps specific |
|---|---|---|---|
| `Type != "DATA_UPDATED"` skipped | ✓ |  |  |
| `Worker` mismatch skipped | ✓ | (Worker=MAP skipped) | (Worker=MONSTER skipped) |
| Malformed `TenantId` errors-and-continues | ✓ |  |  |
| Happy-path triggers FlushTenant | ✓ | `information.FlushTenant` called once with parsed tenant | `SpawnPointRegistry.FlushTenant` called once with parsed tenant |
| `DATA_EVENTS_CONSUMER_ENABLED=false` skips registration | ✓ |  |  |
| Tenant isolation |  | Flushing tenant A leaves tenant B's cache populated | Flushing tenant A leaves `atlas:maps:spawn:B:*` keys intact |
| Flush failure handling |  | (cache flush is in-process; cannot fail) | Redis error logged + counter; offset commits |

Tenant-isolation tests for atlas-maps use `miniredis` (already used elsewhere in `atlas-maps/map/monster/registry_test.go` if it exists; otherwise `redis-mock`). For atlas-monsters, the test pre-populates two tenant caches via the existing read-through `GetById` path with stubbed upstream and asserts `Len()` after flush.

### 7.5 Wiring in main.go

For both `atlas-monsters/main.go` and `atlas-maps/main.go`:

```go
import data2 "atlas-<svc>/kafka/consumer/data"

const dataEventsSuffix = "data-events"

// in main():
dataEventsGroupId := consumer.PerPodGroup(l, serviceName, dataEventsSuffix)
data2.InitConsumers(l)(cmf)(dataEventsGroupId)
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
| `HOSTNAME` | atlas-monsters, atlas-maps | (k8s auto-populates; falls back to startup UUID) | Pod identifier for per-pod group suffix. |

### 8.2 Deploy ConfigMap Edit

`deploy/k8s/env-configmap.yaml` — single-line addition under the `EVENT_TOPIC_*` block:

```yaml
EVENT_TOPIC_DATA: "EVENT_TOPIC_DATA"
```

The shared `atlas-env` ConfigMap is referenced by every service's deployment via `envFrom: configMapRef.name: atlas-env`, so this single edit lights up the variable in `atlas-data`, `atlas-monsters`, `atlas-maps`, and any future consumer simultaneously. No per-service deploy edit needed.

`deploy/compose/.env.example` — single-line addition:

```
EVENT_TOPIC_DATA=EVENT_TOPIC_DATA
```

### 8.3 Topic Bootstrap

The repo does not contain a Kafka cluster manifest (`deploy/k8s/` has no Strimzi/CRD; the broker is provisioned outside this repo). Existing topics rely on broker-side auto-create-topics (the `COMMAND_TOPIC_DATA` topic is created on first produce). `EVENT_TOPIC_DATA` follows the same convention.

**Risk:** if the production broker disables auto-create, this task ships a producer that will fail silently (Kafka returns "unknown topic" → producer's WARN log fires forever, no events flow). Mitigation: an operator-facing check is added to the runbook in §11. This is identical to the operational risk every other dynamic topic in the repo carries today, so we don't gate the task on it.

### 8.4 No Changes To

- `atlas-data` REST API (no surface changes per PRD §5).
- Any service's HTTP routes.
- Any database schema (no migrations).
- The existing manual `DEL atlas:maps:spawn:*` runbook (remains as fallback).

---

## 9. Cross-Cutting Concerns

### 9.1 Multi-Tenant Isolation

Two independent isolation guarantees:

1. **Producer:** `dataUpdatedEventProvider` takes `tenantId` as a string parameter resolved from `tenant.MustFromContext(ctx)` inside `StartWorker`. There is no path that emits an event with a different tenant than the one in the ctx — verified by the `TestStartWorker_EmitsOnSuccess` test asserting `body.tenantId == ctx tenant`.
2. **Consumer:** the handler parses `e.Body.TenantId` and passes the parsed UUID to `FlushTenant`. The handler MUST NOT fall back to "flush all tenants" on parse error — the `parse_error` branch returns early with no flush. Verified by `TestHandleDataUpdated_MalformedTenantId_NoFlush`.

The PRD's §4.8 cross-check ("Kafka header tenant agrees with body tenant") is implemented as a WARN log only, not an enforcement gate. Header tenant vs body tenant disagreement is a code bug, not a runtime hazard — log it loudly, prefer the body, move on.

### 9.2 Idempotency

A duplicate `DATA_UPDATED` event causes:

- atlas-monsters: a second `Flush()` on an already-empty cache → returns `(0, 0)`, increments the flush counter once, evicts nothing. No-op-equivalent.
- atlas-maps: a second `SCAN`/`DEL` on an already-empty key range → 0 keys deleted, 0 round-trips of meaningful work. No-op-equivalent.

No deduplication tokens, no event ids, no idempotency keys. Per PRD §2, this is by design.

### 9.3 Race Cleanliness

- `libs/atlas-cache.Cache.Flush` holds the write lock for the duration of map-clear + callback loop.
- atlas-monsters' `tenantCachesMu.RLock` is held only while *looking up* the cache instance; the actual `Flush` happens after `RUnlock` because `Cache` itself is concurrency-safe. No nested locking.
- atlas-maps' `SpawnPointRegistry.FlushTenant` is goroutine-safe by virtue of the underlying `goredis.Client`'s pool semantics; no shared state in the registry struct beyond the client pointer.
- All new code paths get `go test -race` coverage in CI.

### 9.4 Order of Events

Tenant T's events are key-partitioned by tenant id, so all `DATA_UPDATED` events for T land on the same partition and are delivered to the consumer in emission order. Per-consumer-group, that consumer gets them strictly ordered.

Cross-tenant order is irrelevant.

### 9.5 Backpressure / Rate Limiting

No special handling. The producer emits at most one event per worker-completion, and a full data import emits ≤ 17 events per tenant (one per worker). The consumer's flush is fast (in-process map clear; bounded Redis SCAN). No backlog accumulation expected.

If the topic were ever heavily reused (e.g. for `DATA_IMPORT_STARTED` events at byte rate), we'd revisit. Out of scope here.

---

## 10. Decisions on PRD §9 Open Questions

### 10.1 Topic Name Finalized as `EVENT_TOPIC_DATA`

Already pinned by PRD §9. No further decision; design uses it consistently.

### 10.2 Atlas-Maps Drops `WorkerMonster` Branch

PRD §4.6 conservatively flushed atlas-maps spawn registry on both `Worker=MAP` and `Worker=MONSTER`, with a note to verify at design time.

**Verified against `services/atlas-maps/atlas.com/maps/map/monster/registry.go`:** the `storedSpawnPoint` struct (lines 18-32) holds only spawn-point geometry and timing (`Id`, `Template`, `MobTime`, `Cy`, `F`, `Fh`, `Rx0`, `Rx1`, `X`, `Y`, `NextSpawnAt`). The `Template` field is a monster id reference — but the registry stores nothing **about** the monster (no HP, no exp, no drops). Those attributes live in atlas-data and are looked up at spawn time via the regular HTTP path.

The `MobTime` field (respawn cooldown) is part of the **map's** spawn-point definition (`Map.wz`), populated by `WorkerMap`, not `WorkerMonster`.

**Conclusion:** atlas-maps' spawn registry depends only on map data. **Drop `Worker=MONSTER` from atlas-maps' filter.** Atlas-maps flushes only on `Worker=MAP`.

This eliminates a redundant flush on every monster re-import (which is the more common operational case), reducing Redis load.

The acceptance criterion in PRD §10 should be updated accordingly during plan execution: "On `DATA_UPDATED` with `Worker=MAP`, the spawn-point registry for `TenantId` is flushed." (Drop the parenthetical about `Worker=MONSTER`.)

### 10.3 `HOSTNAME` Fallback = Startup UUID + WARN

Pinned in §6.1 above. Reasoning: local Docker compose without `hostname:` set is a real path; failing fast there breaks dev. Production k8s always populates `HOSTNAME`, so the fallback is dev-only in practice. The WARN log makes any production miss visible.

Alternative considered: **fail-fast at startup**. Rejected because the cost (a broken local dev experience) outweighs the safety win (a WARN-logged unique UUID is functionally equivalent to a stable hostname for fan-out semantics — both deliver to all replicas).

### 10.4 Topic Bootstrap = Auto-Create

Pinned in §8.3. The repo has no Kafka manifest; auto-create is the de facto convention.

### 10.5 `libs/atlas-kafka` Per-Pod-Group Helper = Add It

Pinned in §6.1. Verified by inspection that no such helper exists today; this task adds `consumer.PerPodGroup`. ~25 LOC of library code, ~15 LOC of test, paid back the first time a third consumer adopts.

### 10.6 Consumer Commit Semantics = Library Default (Commit-After-Handler)

Verified by reading `libs/atlas-kafka/message/handler.go` and `libs/atlas-kafka/consumer/manager.go`: handlers are fire-and-forget; the consumer driver commits the offset after the handler returns regardless of internal handler errors (which are logged but not propagated back to the consumer loop).

This matches PRD §4.9's "commit-and-continue" requirement for parse and flush failures. **No special handling needed; the handler simply logs and returns.** Tests verify the offset is committed on flush failure by examining the consumer's committed-offset tracking.

### 10.7 Future Event Types = Out of Scope

PRD §9 already noted this; design just confirms the discriminator field is present and switched on correctly.

---

## 11. Operational Notes

### 11.1 Healthy-State Indicators

| Question | Query | Healthy |
|---|---|---|
| Producer emitting? | `rate(atlas_data_events_emitted_total[5m])` | Non-zero during/after data imports. |
| Producer failing? | `rate(atlas_data_events_emit_failures_total[5m])` | Always 0 in steady state. |
| Consumer keeping up? | Compare emit rate vs. `rate(atlas_<svc>_data_events_processed_total{action="flushed"}[5m])` per replica | Per-replica processed rate ≈ emit rate. |
| Flushes succeeding? | `rate(atlas_<svc>_data_events_consumer_errors_total{kind="flush"}[5m])` | Always 0 in steady state. |
| Cache evictions correctly classified? | Existing `atlas_monsters_data_cache_evictions_total{reason}` panel | New `reason="invalidation"` series appears alongside `expired_positive` and `expired_negative`. |

### 11.2 Deploy Verification Runbook

Post-deploy, run a smoke test:

1. Trigger a `START_WORKER` for `WorkerMonster` against test tenant T.
2. Within 5 s, on **every** replica of `atlas-monsters`:
   - `atlas_monsters_data_cache_size{tenant=T}` drops to 0.
   - `atlas_monsters_data_events_processed_total{worker="MONSTER", action="flushed"}` increments by 1.
3. Trigger a `START_WORKER` for `WorkerMap` against test tenant T.
4. Within 5 s, on **every** replica of `atlas-maps`:
   - `redis-cli SCAN 0 MATCH atlas:maps:spawn:T:*` returns empty.
   - `atlas_maps_data_events_processed_total{worker="MAP", action="flushed"}` increments by 1.
5. Hit any GET endpoint that touches the cache (e.g. `GET /api/data/monsters/100100` for tenant T) and verify `atlas_monsters_data_cache_size{tenant=T}` rises again.

The existing manual `DEL atlas:maps:spawn:*` runbook (per the user's `reference_atlas_maps_spawn_cache.md` memory) is **demoted to fallback**: only run it if the smoke test fails or `atlas_data_events_emit_failures_total` is rising.

### 11.3 Failure Modes

| Failure | Symptom | Action |
|---|---|---|
| Producer cannot reach Kafka | `atlas_data_events_emit_failures_total` rising; data imports still succeed. | Investigate Kafka health. Caches will still expire via TTL. Falls back to manual flush runbook if urgent. |
| Consumer cannot reach Redis (atlas-maps) | `atlas_maps_data_events_consumer_errors_total{kind="flush"}` rising; spawn keys remain stale. | Investigate Redis health. Manual `DEL` runbook still works as bypass. |
| `HOSTNAME` is unset and uses fallback UUID | Single startup WARN log; consumer group id is process-unique. | Functional but log-noisy on every restart. Set `HOSTNAME` in compose or check k8s downward API. |
| Kafka broker disables auto-create-topics | First producer call returns "unknown topic"; no events flow. | Operator creates the topic manually with default partition/replication settings. |
| Two pods of the same service share a `HOSTNAME` (extremely unusual) | One pod's flush "wins" the partition; the other pod's cache stays stale until TTL. | Not a real risk in k8s; flagged in the PerPodGroup unit test as "must be unique". |

---

## 12. Alternatives Considered

### 12.1 Single Centralized Invalidator Service

**Considered:** stand up a new `atlas-cache-invalidator` service that subscribes to `EVENT_TOPIC_DATA` and calls `POST /admin/cache/flush` on each consumer service.

**Rejected because:**

- Adds a second service to the deploy (more YAML, more pods, more failure modes).
- Requires every consumer service to expose an admin HTTP endpoint with cache-flush authorization — new attack surface.
- Doesn't solve the multi-pod fanout problem; the invalidator would still need to call N replicas, knowing them by service-discovery somehow.

The Kafka-fanout-with-per-pod-group solution is simpler, has no new service, and reuses a transport every consumer already speaks.

### 12.2 Selective Per-Id Invalidation

**Considered:** include the affected ids in the event body, e.g. `{"tenantId": "...", "worker": "MONSTER", "ids": [100100, 100101, ...]}` so consumers can `Delete(id)` instead of `Flush()`.

**Rejected because:**

- The trigger is whole-data-set re-import. The producer doesn't know which ids changed.
- Computing the diff is its own complex operation and doesn't pay back: the data sets are bounded (a few thousand monsters); a Flush + re-fetch on next access is fast.
- Selective invalidation would force the event payload to grow with set size — bad for Kafka log size.

PRD §2 explicitly rules this out. Confirmed.

### 12.3 Etag / Version-Based Polling

**Considered:** add a `GET /api/data/version` endpoint that returns a tenant-data version token. Consumers poll periodically and flush when the token changes.

**Rejected because:**

- Polling is wasteful; eventing is push.
- Adds latency (poll interval lower bound, typically 30s+).
- Doesn't get rid of the Kafka topic — we'd still want a push channel for "version changed."

The Kafka event approach has none of these costs.

### 12.4 Compacted Kafka Topic with `tenantId` as Key

**Considered:** make `EVENT_TOPIC_DATA` log-compacted so each tenant's last invalidation event is retained. A starting pod would consume the compacted log and replay the latest invalidation for each tenant, eliminating the "missed events during pod start" risk.

**Rejected because:**

- Compaction needs careful key design (current key `tenantId` would collapse multi-worker events into one). We'd need a composite key `tenantId|worker` or similar.
- The "missed events during pod start" risk is small: pod start is a rare event; a missed event means one extra round of stale reads until TTL or the next import. Acceptable.
- Compacted topics have heavier broker config requirements; aligning that with the existing Kafka ops story is its own subtask.

If the operational pain becomes real, this is a follow-up — same producer, same body, change of topic config. Not gated here.

### 12.5 Per-Service Static Group Id with Manual Replica Counting

**Considered:** keep a fixed-string group id per service but issue replica-specific consumers via the consumer manager.

**Rejected because:**

- The Kafka library would still deliver the partition to one consumer per group.
- "Manual replica counting" requires the service to know its peers — needs service discovery or k8s headless DNS lookups. Coupling to deployment topology defeats the abstraction.

Per-pod groups are the natural Kafka primitive for fan-out; use it.

---

## 13. Test Plan Summary

Five test harnesses, all `go test -race` clean:

| Harness | Lives in | Tests |
|---|---|---|
| `libs/atlas-cache` | `cache_test.go` | §3.4 (5 tests) |
| `libs/atlas-kafka/consumer` | `group_test.go` | §6.4 (4 tests) |
| `atlas-data` producer | `data/producer_test.go` (new), `data/processor_test.go` extension | §5.5 (6 tests) |
| `atlas-monsters` consumer | `kafka/consumer/data/handler_test.go` (new), `monster/information/cache_test.go` extension | §7.4 (multi-tenant + happy path + skip + error) |
| `atlas-maps` consumer | `kafka/consumer/data/handler_test.go` (new), `map/monster/registry_test.go` extension | §7.4 (multi-tenant + happy path + skip + error + Redis-failure) |

Plus the §11.2 manual smoke test against a deployed cluster, gated by the acceptance criteria in PRD §10.

---

## 14. What This Design Deliberately Does Not Do

- Does not add caches to `atlas-channel`, `atlas-pets`, `atlas-monster-death`, etc. Each is a follow-up task that wires its own consumer using the same `consumer.PerPodGroup` helper.
- Does not introduce dedup tokens, event ids, or idempotency keys.
- Does not change `atlas-data`'s import logic, file format, or REST API.
- Does not retire the existing manual `DEL` runbook — only demotes it to fallback.
- Does not add an admin endpoint for manual invalidation. Operators trigger by re-importing.
- Does not gate on cross-cluster propagation; single-cluster Kafka assumption matches the rest of the project.
- Does not introduce a new compacted topic or change the broker's auto-create-topics behavior.
- Does not retrofit the `atlas-monsters`/`atlas-maps` existing fixed-group consumers to per-pod groups. Only the new `data-events` consumer uses per-pod; everything else is unchanged.
- Does not implement future event types (`DATA_IMPORT_STARTED`, `DATA_IMPORT_FAILED`). The discriminator field is present so future tasks can add them without breaking consumers.
