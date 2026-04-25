# Kafka Writer Registry — Design

Version: v1
Status: Draft
Created: 2026-04-25
Companion to: `prd.md`, `migration-plan.md`

---

## 1. Scope of this document

This design fixes the architectural choices left open by the PRD. Goals, non-goals, acceptance criteria, and the list of affected services are not re-litigated here — see `prd.md`. This document settles:

1. The registry type's name, public surface, and concurrency model.
2. The fate of the existing `WriterProvider` helper.
3. The exact replacement shape for the per-service `kafka/producer/producer.go` wrappers.
4. The exact lines added to each service's `main.go`.
5. The test seam that lets `producer_test.go` keep its mock-driven fast path while also covering the new manager logic.
6. A copy-pasteable smoke-test runbook for atlas-data, plus a graceful-shutdown verification step.

## 2. Decisions summary

| # | Question | Decision |
|---|---|---|
| 1 | Registry naming | `Manager` + `GetManager()` — mirrors `consumer.GetManager()` |
| 2 | Lazy-creation concurrency | `sync.RWMutex` + double-checked locking |
| 3 | `WriterProvider` fate | Deleted in this PR (single-PR migration covers all callsites) |
| 4 | `ProviderImpl` resolution | New free function `producer.ManagerWriterProvider(l)(token) model.Provider[Writer]` |
| 5 | `main.go` integration | Lazy — no `Init`. Single line: `tdm.TeardownFunc(func(){ _ = producer.GetManager().Close(l) })` |
| 6 | Test seam | `ResetInstance()` + `ConfigWriterFactory(WriterFactory)` configurator (mirrors consumer) |
| 7 | Debug HTTP handler | Deferred — not in this task |
| 8 | Smoke test | Concrete reproducer (commands + pass criteria + shutdown verification) — see Section 8 |

## 3. Library API surface (`libs/atlas-kafka/producer/`)

### 3.1 New file: `manager.go`

```go
package producer

import (
    "errors"
    "os"
    "sync"
    "time"

    "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
    "github.com/Chronicle20/atlas/libs/atlas-model/model"
    "github.com/segmentio/kafka-go"
    "github.com/sirupsen/logrus"
)

// WriterFactory builds a Writer for a resolved topic name. Tests inject
// a stub via ConfigWriterFactory; production uses defaultWriterFactory.
type WriterFactory func(topicName string) Writer

type ManagerConfig func(m *Manager)

//goland:noinspection GoUnusedExportedFunction
func ConfigWriterFactory(wf WriterFactory) ManagerConfig {
    return func(m *Manager) { m.wf = wf }
}

type Manager struct {
    mu      sync.RWMutex
    writers map[string]Writer
    wf      WriterFactory
    closed  bool
}

var (
    manager     *Manager
    managerOnce sync.Once
)

// ResetInstance clears the singleton. Test-only.
//
//goland:noinspection GoUnusedExportedFunction
func ResetInstance() {
    manager = nil
    managerOnce = sync.Once{}
}

//goland:noinspection GoUnusedExportedFunction
func GetManager(configurators ...ManagerConfig) *Manager {
    managerOnce.Do(func() {
        manager = &Manager{
            writers: make(map[string]Writer),
            wf:      defaultWriterFactory,
        }
        for _, c := range configurators {
            c(manager)
        }
    })
    return manager
}

var ErrManagerClosed = errors.New("producer manager is closed")

// Writer returns the long-lived Writer for the topic resolved from token,
// constructing it on first request. Concurrent first-touches return the
// same instance.
func (m *Manager) Writer(l logrus.FieldLogger, token string) (Writer, error) {
    t, err := topic.EnvProvider(l)(token)()
    if err != nil {
        return nil, err
    }

    m.mu.RLock()
    if m.closed {
        m.mu.RUnlock()
        return nil, ErrManagerClosed
    }
    if w, ok := m.writers[t]; ok {
        m.mu.RUnlock()
        return w, nil
    }
    m.mu.RUnlock()

    m.mu.Lock()
    defer m.mu.Unlock()
    if m.closed {
        return nil, ErrManagerClosed
    }
    if w, ok := m.writers[t]; ok { // double-check
        return w, nil
    }
    w := m.wf(t)
    m.writers[t] = w
    l.Infof("Created kafka writer for topic [%s].", t)
    return w, nil
}

// Close closes every registered Writer and marks the manager closed.
// Idempotent: subsequent calls are no-ops. Errors from individual
// Writer.Close calls are logged but do not short-circuit the loop.
func (m *Manager) Close(l logrus.FieldLogger) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.closed {
        return nil
    }
    m.closed = true

    var errs int
    for t, w := range m.writers {
        if err := w.Close(); err != nil {
            errs++
            l.WithError(err).Warnf("Error closing kafka writer for topic [%s].", t)
        }
    }
    l.Infof("Producer manager shut down %d writers (errors=%d).", len(m.writers), errs)
    return nil
}

func defaultWriterFactory(topicName string) Writer {
    return WriterImpl{w: &kafka.Writer{
        Addr:                   kafka.TCP(os.Getenv("BOOTSTRAP_SERVERS")),
        Topic:                  topicName,
        Balancer:               &kafka.LeastBytes{},
        BatchTimeout:           50 * time.Millisecond,
        AllowAutoTopicCreation: true,
    }}
}

// ManagerWriterProvider returns a model.Provider[Writer] backed by the
// process-wide manager. Replaces the deleted WriterProvider helper.
//
//goland:noinspection GoUnusedExportedFunction
func ManagerWriterProvider(l logrus.FieldLogger) func(token string) model.Provider[Writer] {
    return func(token string) model.Provider[Writer] {
        return func() (Writer, error) {
            return GetManager().Writer(l, token)
        }
    }
}
```

### 3.2 Modified `producer.go`

Two surgical edits, nothing else:

1. **Delete `WriterProvider`** (the entire function, lines ~46–63). Its only use was inside the per-service wrappers, all of which are being rewritten in the same PR. The `topic` import line becomes unused and is removed.
2. **In `Produce`, remove the per-call close.** The `err = w.Close()` block (lines ~90–93) is deleted. The function still returns `nil` at the bottom of the success path; everything else (`tryMessage`, retry config, decorator evaluation) is unchanged.

The `Writer` interface, `WriterImpl`, `CreateKey`, `tryMessage`, `DecorateHeaders`, and `MessageProducer` itself stay byte-identical.

### 3.3 Concurrency contract

- `kafka.Writer` from `segmentio/kafka-go` is documented as safe for concurrent use; `WriteMessages` may be invoked from many goroutines on one Writer. The registry relies on this contract.
- Steady-state lookup is `RLock → map read → RUnlock`. No allocation, no blocking on a write mutex once a Writer exists.
- Cold-path (first lookup) takes the write lock, re-checks the map (avoids the lost-update window between releasing the read lock and acquiring the write lock), and constructs.
- `Close` takes the write lock and sets `closed = true` before iterating, so no new Writers can be created mid-shutdown. In-flight `Writer()` calls that already returned a Writer are unaffected — those goroutines hold a reference to the underlying `kafka.Writer`, and `kafka.Writer.Close()` is documented to drain pending writes.
- Publishers that arrive *after* `Close` get `ErrManagerClosed`. The publish call returns this error to its caller; for code paths invoked from a teardown context this is the correct behavior (no message is silently dropped, no panic).

### 3.4 Test seam

`producer_test.go`'s existing tests (`TestProducer`, `TestProducer2`) hand a `model.FixedProvider[Writer](&MockWriter{})` directly into `Produce`. They never go through the manager and need no changes.

New manager tests use the configurator hook plus `ResetInstance`:

```go
func TestManager_LazyCreate(t *testing.T) {
    producer.ResetInstance()
    var built int32
    factory := func(topicName string) producer.Writer {
        atomic.AddInt32(&built, 1)
        return &fakeWriter{topic: topicName}
    }
    m := producer.GetManager(producer.ConfigWriterFactory(factory))
    l, _ := test.NewNullLogger()

    // First touch builds, second touch reuses.
    w1, _ := m.Writer(l, "MY_TOPIC")
    w2, _ := m.Writer(l, "MY_TOPIC")
    if w1 != w2 || atomic.LoadInt32(&built) != 1 { t.Fatal(...) }
}
```

Each manager test starts with `ResetInstance()`. They run sequentially within the package (the default; no `t.Parallel()` is called).

### 3.5 Required new tests

| Test | Asserts |
|---|---|
| `TestManager_LazyCreate` | first touch builds; second touch returns same instance; factory called exactly once |
| `TestManager_ConcurrentFirstTouch` | N goroutines call `Writer(l, "T")` simultaneously → factory called exactly once → all goroutines return the same `Writer` instance |
| `TestManager_IdempotentClose` | `Close` then `Close` again returns nil; underlying writer's `Close` called exactly once |
| `TestManager_CloseErrorsDoNotShortCircuit` | factory produces 3 writers, one returns error from `Close`; all 3 are still closed |
| `TestManager_WriterAfterClose` | `Writer(l, "T")` after `Close` returns `ErrManagerClosed` |
| `TestManager_TopicResolutionError` | a token that fails resolution propagates the error to the caller; no entry stored in the map |

The existing two tests in `producer_test.go` are kept verbatim (they validate `Produce`'s decorator/retry path, not Writer lifetime).

## 4. Per-service producer wrapper rewrite

Every `services/*/atlas.com/*/kafka/producer/producer.go` becomes a one-line body change. Before:

```go
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) func(token string) producer.MessageProducer {
    return func(ctx context.Context) func(token string) producer.MessageProducer {
        sd := producer.SpanHeaderDecorator(ctx)
        td := producer.TenantHeaderDecorator(ctx)
        return func(token string) producer.MessageProducer {
            return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(token)))(sd, td)
        }
    }
}
```

After:

```go
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) func(token string) producer.MessageProducer {
    return func(ctx context.Context) func(token string) producer.MessageProducer {
        sd := producer.SpanHeaderDecorator(ctx)
        td := producer.TenantHeaderDecorator(ctx)
        return func(token string) producer.MessageProducer {
            return producer.Produce(l)(producer.ManagerWriterProvider(l)(token))(sd, td)
        }
    }
}
```

Diff per file:
- Replace `producer.WriterProvider(topic.EnvProvider(l)(token))` with `producer.ManagerWriterProvider(l)(token)`.
- Remove the `"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"` import (now unused).

The exported function signature, `Provider` type, and decorator wiring all stay identical. The 163 callsites of `producer.ProviderImpl(l)(ctx)(EnvSomethingTopic)` see no change.

A scripted edit (e.g. `gofmt`+`gopls rename` or a tested `sed` script) is acceptable; the diff is byte-identical across all 63 files modulo the package name on line 1.

## 5. Service `main.go` integration

Each `services/*/atlas.com/*/main.go` adds **one** line, placed immediately after the consumer-handler registration block and before `server.New(l)...Run()`:

```go
tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })
```

This closes over `l` and `tdm` from the surrounding scope. The `_ =` discards the return of `Close` (it currently always returns nil; the return is reserved for future use without breaking the signature).

The import block gains:

```go
"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
```

…unless `main.go` already imports it (some services already reference it).

### 5.1 Why no `Init(l)`

`GetManager()`'s `sync.Once` initializes the singleton on first access. The first publish or the teardown registration (whichever fires first) triggers it. The only registry-level log lines are emitted during `Close` and during first-Writer-creation, both of which receive `l` directly. Threading a logger into `Init` purely to store it for later would add boilerplate without behavior.

### 5.2 Teardown ordering — what's actually guaranteed

`service.Manager.TeardownFunc` registers a goroutine that waits on `doneChan` and then calls the function; `Wait()` blocks until all such goroutines return. There is **no FIFO/LIFO ordering** between teardown functions — they fire concurrently when the signal arrives.

For our purposes, this is fine:
- Consumers committing offsets do not depend on producers being alive (offsets go to Kafka, not to the producer side).
- Producers flushing batches do not depend on consumers being alive.

The PRD's mention of "consumers stop first, producers flush last" is based on a misreading of the teardown manager. The design's correct claim is: "consumer and producer teardown run concurrently; both must complete for `Wait()` to return."

If a future requirement needs ordered shutdown (e.g. "drain consumers before stopping producers"), `service.Manager` will need a phased-teardown extension — that is out of scope here.

## 6. Sequence — first publish on a fresh process

```
Caller                  Wrapper                   Manager                 kafka.Writer
  │                       │                          │                        │
  │  Produce(l)(prov)(d)  │                          │                        │
  ├──────────────────────▶│                          │                        │
  │                       │  ManagerWriterProvider() │                        │
  │                       │  invocation pulls thunk  │                        │
  │                       │                          │                        │
  │  WriteMessages(...)   │                          │                        │
  ├──────────────────────▶│                          │                        │
  │                       │  thunk() → mgr.Writer    │                        │
  │                       ├─────────────────────────▶│                        │
  │                       │                          │  RLock/miss            │
  │                       │                          │  Lock/double-check     │
  │                       │                          │  defaultWriterFactory  │
  │                       │                          ├───────────────────────▶│  alloc kafka.Writer
  │                       │                          │  store in map          │
  │                       │                          │  log "created"         │
  │                       │  Writer instance         │                        │
  │                       │◀─────────────────────────┤                        │
  │                       │  WriteMessages(ctx, msg) │                        │
  │                       ├──────────────────────────────────────────────────▶│  TCP+publish
  │                       │  (no Close)              │                        │
  │  return nil           │                          │                        │
  │◀──────────────────────┤                          │                        │
```

Subsequent publishes on the same topic skip the `Lock` path entirely — the `RLock` hit is cache-friendly and lock-free in the steady state.

## 7. Files touched (final inventory)

The implementation phase MUST re-run these greps before merging to confirm the live count, but the snapshot is:

```
# Library: 4 files
libs/atlas-kafka/producer/manager.go         (new)
libs/atlas-kafka/producer/manager_test.go    (new)
libs/atlas-kafka/producer/producer.go        (delete WriterProvider, remove w.Close())
libs/atlas-kafka/producer/producer_test.go   (no change — verify still passes)

# Per service, 2 files × 63 services = 126 files
services/<svc>/atlas.com/<name>/kafka/producer/producer.go
services/<svc>/atlas.com/<name>/main.go
```

Pre-merge greps to run (from migration-plan.md, restated for completeness):

```bash
# Confirm no callsites of the deleted helper remain
grep -rn "producer\.WriterProvider" services/ libs/

# Confirm every wrapper now uses the new provider
grep -rln "producer\.ManagerWriterProvider" services/ | wc -l   # expect 63

# Confirm every main.go registers teardown
grep -rln "producer\.GetManager().Close" services/ | wc -l       # expect 63

# Confirm no service constructs a kafka.Writer directly
grep -rn "kafka\.Writer{" services/                              # expect 0 hits

# Sanity: no test asserts on per-publish Writer.Close()
grep -rn "Close()" services/*/atlas.com/*/kafka/producer/*_test.go   # expect 0
```

## 8. Smoke-test runbook

Run from a fresh checkout against the standard local Kafka stack (`compose.yml` or Tilt-managed). Substitute `<broker>` with the actual broker address (e.g. `kafka:9092` inside the network or `localhost:9092` from the host).

### 8.1 Pre-flight

```bash
# Confirm or create the command topic with ≥4 partitions.
kafka-topics.sh --bootstrap-server <broker> \
  --create --topic command.data --partitions 4 --replication-factor 1 \
  --if-not-exists

# Verify partition count.
kafka-topics.sh --bootstrap-server <broker> \
  --describe --topic command.data
# Expect: PartitionCount: 4
```

### 8.2 Service env (atlas-data)

```
BOOTSTRAP_SERVERS=<broker>
COMMAND_TOPIC_DATA=command.data
ZIP_DIR=<path containing <tenant-uuid>/<region>/<major>.<minor>/ data>
REST_PORT=8080
```

### 8.3 Start atlas-data and capture baseline

```bash
# Capture baseline group offsets (consumer group is "Data Service").
kafka-consumer-groups.sh --bootstrap-server <broker> \
  --describe --group "Data Service"
```

If the group does not yet exist (cold cluster), run `atlas-data` once with a no-op consumer touch first, or accept that the first describe will say "no active members" — the post-test offsets are what matter.

### 8.4 Drive a publish burst

`POST /api/data/process` triggers `ProcessData`, which calls `InstructWorker` for every entry in `Workers` (currently ~10 worker types). Each call publishes one `START_WORKER` command on `COMMAND_TOPIC_DATA`. A single request produces ~10 messages. Drive enough volume to detect partition skew:

```bash
TENANT_ID=<some-uuid>
REGION=GMS
MAJOR=83
MINOR=1

for i in $(seq 1 5); do
  curl -fsS -X POST http://localhost:8080/api/data/process \
    -H "TENANT_ID: ${TENANT_ID}" \
    -H "REGION: ${REGION}" \
    -H "MAJOR_VERSION: ${MAJOR}" \
    -H "MINOR_VERSION: ${MINOR}"
done
```

5 requests × ~10 workers = ~50 messages. Adjust the loop count if `Workers` has changed.

### 8.5 Verify partition fan-out

```bash
kafka-consumer-groups.sh --bootstrap-server <broker> \
  --describe --group "Data Service"
```

#### Pass criteria

- At least **3 of the 4 partitions** show non-zero `CURRENT-OFFSET` advancement vs. baseline. With 50 messages and `LeastBytes`, all-4 partitions advancing is the most likely outcome but a single empty partition is acceptable noise.

#### Fail signature (the bug being fixed)

- All advancement on partition 0; partitions 1, 2, 3 show `CURRENT-OFFSET = 0`. This is the symptom that prompted the task.

#### Inconclusive

- Fewer than 3 partitions advance — increase the loop count to 20 and re-run. If still skewed, the fix is wrong; do not merge.

### 8.6 Graceful-shutdown verification

While `atlas-data` is still running and after at least one `POST /api/data/process` has completed:

```bash
# Find the PID and send SIGTERM.
pkill -SIGTERM -f atlas-data

# Tail the logs.
tail -f <atlas-data-log>
```

#### Pass criteria

- A log line of the form `Producer manager shut down N writers (errors=0).` appears before the process exits.
- `N >= 1` (atlas-data publishes to at least `command.data`, so the registry must have at least one Writer).
- The process exits cleanly (rc 0, not killed by a follow-up SIGKILL).

#### Fail signature

- Process hangs on shutdown (`Writer.Close` deadlocks on a slow broker). If observed, file a follow-up to add a context-with-deadline around the close loop. Out of scope for this task.

### 8.7 Bonus check — log line for first-time Writer creation

While driving the 5-request burst above, verify the log shows:

```
Created kafka writer for topic [command.data].
```

…exactly **once** across the entire burst. If it appears more than once (e.g. once per request), the registry isn't caching — fix before merging.

## 9. Risks and mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Service test relies on per-call `Writer.Close()` | Low | Pre-flight grep listed in §7 catches this. Existing test surveys (`producer_test.go`-style) only mock the Writer interface; none observed asserting on per-publish close. |
| Hidden producer callsite that doesn't go through `ProviderImpl` | Low | Pre-flight `grep -rn "kafka\.Writer{" services/` catches direct construction. |
| First-publish latency spike (Writer construction now happens on the request path instead of being amortized into per-call construct/close) | Negligible | `kafka.Writer` construction is allocation-only; no I/O until first `WriteMessages`. The previous per-call construct already paid this cost on every publish. |
| Race between in-flight publish and `Close` | Bounded | `Close` sets `closed = true` under the write lock before iterating. New `Writer()` calls return `ErrManagerClosed`. Already-returned `Writer` references survive the close path of the underlying `kafka.Writer` (segmentio/kafka-go drains pending batches). |
| `ErrManagerClosed` surfaces to a publisher mid-shutdown | Acceptable | The error is logged at the call site (existing `Produce` path logs and returns). Operators see "publish failed during shutdown" rather than a silent drop. |
| Diff size (~130 files) | High | Mechanical, identical-template edits. Migration plan calls out grep-based verification and per-service build runs. |

## 10. Out of scope (explicit)

- Producer debug HTTP handler (deferred per Q7).
- Phased teardown ordering between consumers and producers (current model: concurrent).
- `WriterStats` exposure or any new metrics.
- Topic provisioning, partition counts, or broker config.
- Consumer-side changes.
- atlas-data's `START_WORKER` skew (the partition-distribution problem is fixed *structurally* by this task; any service-specific keying choices remain owned by their respective services).
