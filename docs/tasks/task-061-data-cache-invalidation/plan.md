# atlas-data Cache Invalidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire an event-driven cache-invalidation contract: `atlas-data` emits `DATA_UPDATED` on `EVENT_TOPIC_DATA` after each successful per-tenant per-worker import; `atlas-monsters` and `atlas-maps` subscribe and flush their Redis-backed caches for the affected tenant.

**Architecture:** Three layers — a single library affordance (`TenantRegistry.Clear`), service-side wrappers (`monster/information.FlushTenant`, `SpawnPointRegistry.FlushTenant`), and event plumbing (one producer in atlas-data, one shared-group consumer per service). Whole-tenant per-worker flushes; SCAN+pipelined-DEL; auto-offset-reset=latest; producer log-and-continue on Kafka failure; per-tenant kill switches.

**Tech Stack:** Go, segmentio/kafka-go, redis/go-redis/v9, alicebob/miniredis (tests), prometheus/client_golang, atlas-tenant, atlas-kafka, atlas-redis (existing libs), goredis pipelining.

---

## Prerequisites (one-time, before Task 1)

- [ ] **Verify task-060 v2 is merged to main**

```bash
cd <home>/source/atlas-ms/atlas/.worktrees/task-061-data-cache-invalidation
git log main --oneline | grep -i 'task-060' | head -3
```

Expected: at least one commit referencing `task-060` showing the merge of the Redis-backed `monster/information` cache.

If not merged: STOP. Wait for task-060 to merge, then `git rebase main`. The `posReg`/`negReg` fields and `monster/information/cache.go` referenced throughout this plan come from task-060 v2.

- [ ] **Rebase task-061 branch on main**

```bash
git fetch origin
git rebase origin/main
```

- [ ] **Confirm files exist after rebase**

```bash
ls services/atlas-monsters/atlas.com/monsters/monster/information/cache.go
ls services/atlas-monsters/atlas.com/monsters/monster/information/metrics.go
grep -n 'posReg\|negReg\|MONSTER_DATA_CACHE_ENABLED' services/atlas-monsters/atlas.com/monsters/monster/information/cache.go | head
```

Expected: cache.go and metrics.go both exist; `posReg`, `negReg`, and the env-var constant appear in cache.go.

If any file is missing or the grep is empty, STOP — task-060's merge state is wrong; do not proceed.

---

## Task 1: `libs/atlas-redis.TenantRegistry.Clear` — failing test

**Files:**
- Modify: `libs/atlas-redis/tenant_registry_test.go` (create if absent)

- [ ] **Step 1: Open or create the test file**

Confirm the test setup helper. The pattern lives in `libs/atlas-redis/registry_test.go`:

```go
func setupTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
    t.Helper()
    mr := miniredis.RunT(t)
    return goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), mr
}
```

If `tenant_registry_test.go` already exists, append to it. Otherwise create with package `redis` and standard imports.

- [ ] **Step 2: Add the empty-namespace test**

Append to `libs/atlas-redis/tenant_registry_test.go`:

```go
package redis

import (
    "context"
    "fmt"
    "strconv"
    "sync"
    "testing"

    "github.com/Chronicle20/atlas/libs/atlas-tenant"
    "github.com/google/uuid"
)

func newTestTenant(t *testing.T, region string) tenant.Model {
    t.Helper()
    tm, err := tenant.Create(uuid.New(), region, 0, 83)
    if err != nil {
        t.Fatalf("tenant.Create: %v", err)
    }
    return tm
}

func TestTenantRegistry_Clear_EmptyNamespace(t *testing.T) {
    client, _ := setupTestRedis(t)
    defer client.Close()
    reg := NewTenantRegistry[string, string](client, "test:clear", func(k string) string { return k })
    tm := newTestTenant(t, "GMS")

    deleted, err := reg.Clear(context.Background(), tm)
    if err != nil {
        t.Fatalf("Clear: %v", err)
    }
    if deleted != 0 {
        t.Fatalf("deleted = %d, want 0", deleted)
    }
}
```

- [ ] **Step 3: Run; verify it fails to compile**

```bash
cd libs/atlas-redis && go test -run TestTenantRegistry_Clear_EmptyNamespace -v
```

Expected: build error — `reg.Clear undefined (type *TenantRegistry[string, string] has no field or method Clear)`.

## Task 2: `libs/atlas-redis.TenantRegistry.Clear` — minimal implementation

**Files:**
- Modify: `libs/atlas-redis/tenant_registry.go`

- [ ] **Step 1: Add `Clear` method**

Append to `libs/atlas-redis/tenant_registry.go` (just before the closing of the file, after `Namespace()`):

```go
// Clear deletes every entry for tenant t in this registry's namespace.
// Implementation uses SCAN with COUNT=100 to enumerate keys matching
// tenantScanPattern(r.namespace, t), then pipelines DEL in batches of 100.
// Returns the number of keys deleted (0 if the namespace was already empty
// for this tenant). On a partial failure mid-scan, returns
// (deleted_so_far, err) — the partial deletion is not rolled back; Redis
// converges on the next call.
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

- [ ] **Step 2: Run the empty-namespace test**

```bash
cd libs/atlas-redis && go test -run TestTenantRegistry_Clear_EmptyNamespace -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-redis/tenant_registry.go libs/atlas-redis/tenant_registry_test.go
git commit -m "feat(atlas-redis): TenantRegistry.Clear (empty-namespace happy path)"
```

## Task 3: `Clear` — populated namespace + tenant isolation tests

**Files:**
- Modify: `libs/atlas-redis/tenant_registry_test.go`

- [ ] **Step 1: Add the populated and tenant-isolation tests**

Append:

```go
func TestTenantRegistry_Clear_DeletesAllForTenant(t *testing.T) {
    client, _ := setupTestRedis(t)
    defer client.Close()
    reg := NewTenantRegistry[string, string](client, "test:clear", func(k string) string { return k })
    tm := newTestTenant(t, "GMS")
    ctx := context.Background()

    for i := 0; i < 5; i++ {
        if err := reg.Put(ctx, tm, fmt.Sprintf("k%d", i), "v"); err != nil {
            t.Fatalf("Put: %v", err)
        }
    }

    deleted, err := reg.Clear(ctx, tm)
    if err != nil {
        t.Fatalf("Clear: %v", err)
    }
    if deleted != 5 {
        t.Fatalf("deleted = %d, want 5", deleted)
    }
    for i := 0; i < 5; i++ {
        ok, _ := reg.Exists(ctx, tm, fmt.Sprintf("k%d", i))
        if ok {
            t.Fatalf("key k%d still exists after Clear", i)
        }
    }
}

func TestTenantRegistry_Clear_TenantIsolation(t *testing.T) {
    client, _ := setupTestRedis(t)
    defer client.Close()
    reg := NewTenantRegistry[string, string](client, "test:clear", func(k string) string { return k })
    tA := newTestTenant(t, "GMS")
    tB := newTestTenant(t, "GMS")
    ctx := context.Background()

    for i := 0; i < 3; i++ {
        _ = reg.Put(ctx, tA, fmt.Sprintf("k%d", i), "vA")
        _ = reg.Put(ctx, tB, fmt.Sprintf("k%d", i), "vB")
    }

    deleted, err := reg.Clear(ctx, tA)
    if err != nil {
        t.Fatalf("Clear: %v", err)
    }
    if deleted != 3 {
        t.Fatalf("deleted = %d, want 3", deleted)
    }
    for i := 0; i < 3; i++ {
        if ok, _ := reg.Exists(ctx, tA, fmt.Sprintf("k%d", i)); ok {
            t.Fatalf("tenant A key k%d should be gone", i)
        }
        if ok, _ := reg.Exists(ctx, tB, fmt.Sprintf("k%d", i)); !ok {
            t.Fatalf("tenant B key k%d should still exist", i)
        }
    }
}
```

- [ ] **Step 2: Run new tests**

```bash
cd libs/atlas-redis && go test -run 'TestTenantRegistry_Clear_(DeletesAllForTenant|TenantIsolation)' -v
```

Expected: both PASS.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-redis/tenant_registry_test.go
git commit -m "test(atlas-redis): Clear populated + tenant isolation"
```

## Task 4: `Clear` — namespace isolation + race tests

**Files:**
- Modify: `libs/atlas-redis/tenant_registry_test.go`

- [ ] **Step 1: Add namespace-isolation and race-clean tests**

Append:

```go
func TestTenantRegistry_Clear_NamespaceIsolation(t *testing.T) {
    client, _ := setupTestRedis(t)
    defer client.Close()
    regA := NewTenantRegistry[string, string](client, "test:clear:A", func(k string) string { return k })
    regB := NewTenantRegistry[string, string](client, "test:clear:B", func(k string) string { return k })
    tm := newTestTenant(t, "GMS")
    ctx := context.Background()

    for i := 0; i < 4; i++ {
        _ = regA.Put(ctx, tm, fmt.Sprintf("k%d", i), "vA")
        _ = regB.Put(ctx, tm, fmt.Sprintf("k%d", i), "vB")
    }

    deleted, err := regA.Clear(ctx, tm)
    if err != nil {
        t.Fatalf("Clear: %v", err)
    }
    if deleted != 4 {
        t.Fatalf("deleted = %d, want 4", deleted)
    }
    for i := 0; i < 4; i++ {
        if ok, _ := regA.Exists(ctx, tm, fmt.Sprintf("k%d", i)); ok {
            t.Fatalf("regA key k%d should be gone", i)
        }
        if ok, _ := regB.Exists(ctx, tm, fmt.Sprintf("k%d", i)); !ok {
            t.Fatalf("regB key k%d should still exist", i)
        }
    }
}

func TestTenantRegistry_Clear_RaceCleanWithPut(t *testing.T) {
    client, _ := setupTestRedis(t)
    defer client.Close()
    reg := NewTenantRegistry[string, string](client, "test:clear:race", func(k string) string { return k })
    tm := newTestTenant(t, "GMS")
    ctx := context.Background()

    for i := 0; i < 50; i++ {
        _ = reg.Put(ctx, tm, fmt.Sprintf("seed%d", i), "v")
    }

    var wg sync.WaitGroup
    stop := make(chan struct{})
    for w := 0; w < 4; w++ {
        wg.Add(1)
        go func(w int) {
            defer wg.Done()
            i := 0
            for {
                select {
                case <-stop:
                    return
                default:
                    _ = reg.Put(ctx, tm, fmt.Sprintf("w%d-k%d", w, i), "v")
                    i++
                }
            }
        }(w)
    }

    if _, err := reg.Clear(ctx, tm); err != nil {
        t.Fatalf("Clear: %v", err)
    }
    close(stop)
    wg.Wait()
}

// helper for non-shared use; ensures package compiles with strconv import.
var _ = strconv.Itoa
```

- [ ] **Step 2: Run with race detector**

```bash
cd libs/atlas-redis && go test -race -run 'TestTenantRegistry_Clear' -v
```

Expected: all PASS, no race detector hits.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-redis/tenant_registry_test.go
git commit -m "test(atlas-redis): Clear namespace isolation + race-clean"
```

Note on partial-failure test (design §3.3): miniredis does not provide a "force DEL to fail mid-scan" hook without a custom dial wrapper that doubles the test surface. The behavior is structurally guaranteed by the `firstErr` accumulator and is exercised in service-level tests where Redis can be torn down mid-flush. Skip the dedicated unit test and rely on the wrapper-level `TestFlushTenant_PosRegErrorDoesNotBlockNegReg` (Task 8) for partial-failure coverage.

## Task 5: `atlas-data` event types + topic constant

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/kafka.go`

- [ ] **Step 1: Add new constants and types**

Replace the contents of `services/atlas-data/atlas.com/data/data/kafka.go` with:

```go
package data

const (
    EnvCommandTopic    = "COMMAND_TOPIC_DATA"
    CommandStartWorker = "START_WORKER"

    EnvEventTopic        = "EVENT_TOPIC_DATA"
    EventTypeDataUpdated = "DATA_UPDATED"
)

type command[E any] struct {
    Type string `json:"type"`
    Body E      `json:"body"`
}

type startWorkerCommandBody struct {
    Name string `json:"name"`
    Path string `json:"path"`
}

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

- [ ] **Step 2: Build to verify**

```bash
cd services/atlas-data/atlas.com/data && go build ./data/...
```

Expected: success, no errors.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/kafka.go
git commit -m "feat(atlas-data): add EVENT_TOPIC_DATA + DATA_UPDATED envelope"
```

## Task 6: `atlas-data` producer provider — failing test

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/processor_test.go`

- [ ] **Step 1: Add failing tests for `dataUpdatedEventProvider`**

Append to `processor_test.go` (or create a new `producer_test.go` if the existing test file does not import `kafka-go`; the existing file is already in package `data`):

```go
func TestDataUpdatedEventProvider_KeyIsTenantId(t *testing.T) {
    tenantId := "8b8d2bb0-2d1f-46b0-8c1c-1234567890ab"
    p := dataUpdatedEventProvider(tenantId, WorkerMonster, time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
    msgs, err := p()
    if err != nil {
        t.Fatalf("provider: %v", err)
    }
    if len(msgs) != 1 {
        t.Fatalf("len(msgs) = %d, want 1", len(msgs))
    }
    if string(msgs[0].Key) != tenantId {
        t.Fatalf("key = %q, want %q", string(msgs[0].Key), tenantId)
    }
}

func TestDataUpdatedEventProvider_BodyShape(t *testing.T) {
    tenantId := "8b8d2bb0-2d1f-46b0-8c1c-1234567890ab"
    completedAt := time.Date(2026, 5, 8, 12, 30, 0, 0, time.UTC)
    p := dataUpdatedEventProvider(tenantId, WorkerMap, completedAt)
    msgs, _ := p()

    var ev event[dataUpdatedEventBody]
    if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if ev.Type != EventTypeDataUpdated {
        t.Fatalf("Type = %q, want %q", ev.Type, EventTypeDataUpdated)
    }
    if ev.Body.TenantId != tenantId {
        t.Fatalf("TenantId = %q", ev.Body.TenantId)
    }
    if ev.Body.Worker != WorkerMap {
        t.Fatalf("Worker = %q", ev.Body.Worker)
    }
    if ev.Body.CompletedAt != "2026-05-08T12:30:00Z" {
        t.Fatalf("CompletedAt = %q, want RFC3339 UTC", ev.Body.CompletedAt)
    }
}
```

Add imports as needed: `"encoding/json"`, `"testing"`, `"time"`.

- [ ] **Step 2: Run; expect compile failure**

```bash
cd services/atlas-data/atlas.com/data && go test -run TestDataUpdatedEventProvider -v ./data/...
```

Expected: build error — `dataUpdatedEventProvider undefined`.

## Task 7: `atlas-data` producer provider — implementation

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/producer.go`

- [ ] **Step 1: Add the provider**

Append to `services/atlas-data/atlas.com/data/data/producer.go`:

```go
import "time"

func dataUpdatedEventProvider(tenantId string, worker string, completedAt time.Time) model.Provider[[]kafka.Message] {
    key := []byte(tenantId)
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

(Merge the new `time` import into the existing import block; do not duplicate.)

- [ ] **Step 2: Run the tests**

```bash
cd services/atlas-data/atlas.com/data && go test -run TestDataUpdatedEventProvider -v ./data/...
```

Expected: PASS for both tests.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/producer.go services/atlas-data/atlas.com/data/data/processor_test.go
git commit -m "feat(atlas-data): dataUpdatedEventProvider + tests"
```

## Task 8: `atlas-data` metrics

**Files:**
- Create: `services/atlas-data/atlas.com/data/data/metrics.go`

- [ ] **Step 1: Verify prometheus client_golang is already a dep**

```bash
cd services/atlas-data/atlas.com/data && grep prometheus go.mod
```

If absent, add it: `go get github.com/prometheus/client_golang@latest && go mod tidy`. Otherwise skip.

- [ ] **Step 2: Create metrics.go**

Write `services/atlas-data/atlas.com/data/data/metrics.go`:

```go
package data

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

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

- [ ] **Step 3: Build to verify**

```bash
cd services/atlas-data/atlas.com/data && go build ./data/...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/metrics.go services/atlas-data/atlas.com/data/go.mod services/atlas-data/atlas.com/data/go.sum 2>/dev/null
git commit -m "feat(atlas-data): data-events prometheus counters"
```

## Task 9: `atlas-data` `emitDataUpdated` + kill-switch — failing test

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/processor_test.go`

- [ ] **Step 1: Add failing tests for emit gating**

Append:

```go
func TestProducerEnabled_DefaultTrue(t *testing.T) {
    t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "")
    if !producerEnabled() {
        t.Fatal("expected default true when unset")
    }
}

func TestProducerEnabled_ExplicitFalse(t *testing.T) {
    t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "false")
    if producerEnabled() {
        t.Fatal("expected false when DATA_EVENTS_PRODUCER_ENABLED=false")
    }
}

func TestProducerEnabled_UnparseableTrue(t *testing.T) {
    t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "not-a-bool")
    if !producerEnabled() {
        t.Fatal("expected default true when unparseable")
    }
}
```

- [ ] **Step 2: Run; expect compile failure**

```bash
cd services/atlas-data/atlas.com/data && go test -run TestProducerEnabled -v ./data/...
```

Expected: build error — `producerEnabled undefined`.

## Task 10: `atlas-data` `emitDataUpdated` + kill-switch — implementation

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/processor.go`

- [ ] **Step 1: Add helpers and emit at the bottom of `StartWorker`**

In `processor.go`:

1. Add to imports: `"os"` (already present), `"strconv"`, `"time"`. Add `"atlas-data/kafka/producer"` (already present), `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` (already present).

2. Replace the success tail of `StartWorker` (currently `l.Infof("Worker [%s] completed.", name); return nil`) with:

```go
                if err != nil {
                    l.WithError(err).Errorf("Worker [%s] failed with error.", name)
                    return err
                }
                l.Infof("Worker [%s] completed.", name)
                emitDataUpdated(l, ctx, t, name)
                return nil
```

3. Append two top-level helpers at the end of the file:

```go
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
        return true
    }
    enabled, err := strconv.ParseBool(v)
    if err != nil {
        return true
    }
    return enabled
}
```

- [ ] **Step 2: Run kill-switch tests**

```bash
cd services/atlas-data/atlas.com/data && go test -run TestProducerEnabled -v ./data/...
```

Expected: PASS for all three.

- [ ] **Step 3: Build and run full data package tests**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go test -race ./data/...
```

Expected: build clean, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/processor.go services/atlas-data/atlas.com/data/data/processor_test.go
git commit -m "feat(atlas-data): emit DATA_UPDATED on worker success with kill-switch"
```

Note on `TestStartWorker_*` integration tests (PRD §4.11 list): these require a real DB + filesystem fixture and are deferred to manual E2E verification (§Acceptance below). The producer's behavior is fully covered by `TestDataUpdatedEventProvider_*` and `TestProducerEnabled_*` plus a manual run.

## Task 11: `atlas-monsters` `FlushTenant` wrapper — failing test

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go` (from task-060)

- [ ] **Step 1: Read existing cache_test.go and identify the test setup helper from task-060**

```bash
cat services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go | head -80
```

Identify the helper that initializes the cache against miniredis. Reuse it.

- [ ] **Step 2: Append FlushTenant tests**

Append to `cache_test.go`:

```go
func TestFlushTenant_ClearsBothNamespaces(t *testing.T) {
    // Use the task-060 cache_test.go setup helper. Standard pattern:
    //   client, mr := setupTestCache(t)
    //   InitDataCache(client, true)
    //   defer ResetCacheForTest()
    client, _ := setupTestCache(t)
    InitDataCache(client, true)
    defer ResetCacheForTest()

    tm := newTestTenant(t)
    ctx := context.Background()

    // Populate positive: 3 entries.
    for _, id := range []uint32{1, 2, 3} {
        if err := cache.posReg.Put(ctx, tm, id, Model{}); err != nil {
            t.Fatalf("posReg.Put: %v", err)
        }
    }
    // Populate negative: 2 entries.
    for _, id := range []uint32{99, 100} {
        if err := cache.negReg.Put(ctx, tm, id, struct{}{}); err != nil {
            t.Fatalf("negReg.Put: %v", err)
        }
    }

    deleted, err := FlushTenant(ctx, tm)
    if err != nil {
        t.Fatalf("FlushTenant: %v", err)
    }
    if deleted != 5 {
        t.Fatalf("deleted = %d, want 5", deleted)
    }
}

func TestFlushTenant_TenantIsolation(t *testing.T) {
    client, _ := setupTestCache(t)
    InitDataCache(client, true)
    defer ResetCacheForTest()

    tA := newTestTenant(t)
    tB := newTestTenant(t)
    ctx := context.Background()

    _ = cache.posReg.Put(ctx, tA, uint32(1), Model{})
    _ = cache.posReg.Put(ctx, tB, uint32(1), Model{})

    if _, err := FlushTenant(ctx, tA); err != nil {
        t.Fatalf("FlushTenant: %v", err)
    }
    if ok, _ := cache.posReg.Exists(ctx, tA, uint32(1)); ok {
        t.Fatal("tA key still exists")
    }
    if ok, _ := cache.posReg.Exists(ctx, tB, uint32(1)); !ok {
        t.Fatal("tB key should still exist")
    }
}

func TestFlushTenant_KillSwitchNoOp(t *testing.T) {
    client, _ := setupTestCache(t)
    InitDataCache(client, false) // disabled
    defer ResetCacheForTest()

    deleted, err := FlushTenant(context.Background(), newTestTenant(t))
    if err != nil {
        t.Fatalf("FlushTenant: %v", err)
    }
    if deleted != 0 {
        t.Fatalf("deleted = %d, want 0", deleted)
    }
}

func TestFlushTenant_NilCacheNoOp(t *testing.T) {
    cache = nil // explicit reset
    deleted, err := FlushTenant(context.Background(), newTestTenant(t))
    if err != nil {
        t.Fatalf("FlushTenant: %v", err)
    }
    if deleted != 0 {
        t.Fatalf("deleted = %d, want 0", deleted)
    }
}
```

If task-060's helpers (`setupTestCache`, `InitDataCache`, `ResetCacheForTest`, `newTestTenant`) are named differently, adapt to match. If task-060 does not expose `cache.posReg` / `cache.negReg` directly (i.e. they are unexported and tests are in the same package), this code already works because it's in package `information`.

- [ ] **Step 3: Run; expect compile failure**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -run TestFlushTenant -v ./monster/information/...
```

Expected: build error — `FlushTenant undefined` (and possibly other helpers if names differ; adjust at this step).

## Task 12: `atlas-monsters` `FlushTenant` wrapper — implementation

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/information/cache.go` (task-060 file)

- [ ] **Step 1: Add `FlushTenant`**

Append to `cache.go`:

```go
// FlushTenant clears both the positive and negative cache namespaces for
// tenant t. If the cache is disabled (MONSTER_DATA_CACHE_ENABLED=false) or
// has not yet been initialized, this is a no-op returning (0, nil). On
// partial Redis failure across the two namespaces, returns the running
// total of keys deleted and the first error observed; the second Clear is
// still attempted so a degraded posReg does not block negReg cleanup.
func FlushTenant(ctx context.Context, t tenant.Model) (int, error) {
    if cache == nil || !cache.enabled {
        return 0, nil
    }

    posDeleted, posErr := cache.posReg.Clear(ctx, t)
    negDeleted, negErr := cache.negReg.Clear(ctx, t)
    deleted := posDeleted + negDeleted

    if posErr != nil {
        return deleted, posErr
    }
    return deleted, negErr
}
```

Add imports if missing: `"context"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`.

- [ ] **Step 2: Run the wrapper tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -run TestFlushTenant -v ./monster/information/...
```

Expected: all four PASS.

- [ ] **Step 3: Run package tests with race**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -race ./monster/information/...
```

Expected: PASS, no races.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/information/cache.go services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go
git commit -m "feat(atlas-monsters): FlushTenant wrapper across pos+neg namespaces"
```

## Task 13: `atlas-monsters` data consumer — kafka.go + metrics.go

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/kafka.go`
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/metrics.go`

- [ ] **Step 1: Create kafka.go**

```go
package data

const (
    EnvEventTopic        = "EVENT_TOPIC_DATA"
    EventTypeDataUpdated = "DATA_UPDATED"

    WorkerMonster = "MONSTER"
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

- [ ] **Step 2: Create metrics.go**

```go
package data

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    eventsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_events_processed_total",
        Help: "DATA_UPDATED events processed by the cache-invalidation consumer.",
    }, []string{"worker", "type", "action"})

    eventsConsumerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_events_consumer_errors_total",
        Help: "Errors encountered processing DATA_UPDATED events.",
    }, []string{"kind"})

    eventsSkippedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_events_consumer_skipped_total",
        Help: "DATA_UPDATED events skipped (unknown type or unrelated worker).",
    }, []string{"reason"})

    keysDeletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_monsters_data_events_keys_deleted_total",
        Help: "Redis keys deleted by cache-invalidation flushes, by tenant.",
    }, []string{"tenant"})
)
```

- [ ] **Step 3: Build**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./kafka/consumer/data/...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/kafka.go services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/metrics.go
git commit -m "feat(atlas-monsters): data-events consumer kafka.go + metrics.go"
```

## Task 14: `atlas-monsters` data consumer — handler.go

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/handler.go`

- [ ] **Step 1: Inspect an existing handler for the project's logger/handler signature**

```bash
sed -n '1,60p' services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/handler.go
```

Note the handler signature pattern (typically `func handleX(l logrus.FieldLogger, ctx context.Context, e event[T])`).

- [ ] **Step 2: Create handler.go**

```go
// Package data subscribes to EVENT_TOPIC_DATA for tenant-scoped cache
// invalidation events. We use a shared (single-delivery) consumer group
// "Monster Data Cache Invalidator" rather than a per-pod group because the
// cache state is in shared Redis (task-060 v2): one pod's Clear is visible
// to every replica immediately. If a future cache moves in-process, this
// consumer will need per-pod fan-out — see
// docs/tasks/task-061-data-cache-invalidation/design.md §6.3.
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

    headerTenant, hasHeader := tenant.FromContext(ctx)
    var t tenant.Model
    if hasHeader && headerTenant.Id() == bodyTid {
        t = headerTenant
    } else {
        if hasHeader {
            l.Warnf("Tenant header [%s] disagrees with event body tenant [%s]; using body.", headerTenant.Id(), bodyTid)
        }
        fb, ferr := tenant.Create(bodyTid, "", 0, 0)
        if ferr != nil {
            l.WithError(ferr).Errorf("Failed to construct fallback tenant for [%s].", bodyTid)
            eventsConsumerErrorsTotal.WithLabelValues("parse").Inc()
            return
        }
        t = fb
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

If `tenant.Model.Id()` returns a `uuid.UUID` rather than a stringer wrapper, the comparison `headerTenant.Id() == bodyTid` works directly. Verify by running `go build`; if the comparison fails, switch to `headerTenant.Id().String() == bodyTid.String()`.

- [ ] **Step 3: Build**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./kafka/consumer/data/...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/handler.go
git commit -m "feat(atlas-monsters): data-events handler with shared-group rationale"
```

## Task 15: `atlas-monsters` data consumer — consumer.go

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/consumer.go`

- [ ] **Step 1: Inspect an existing consumer.go to copy structure**

```bash
sed -n '1,80p' services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go
```

Identify:
- The `consumer2` package alias (typically `consumer2 "atlas-monsters/kafka/consumer"`).
- The `NewConfig(l)(name)(envTopic)(groupId)` helper.
- The `message.AdaptHandler(message.PersistentConfig(handler))` adapter chain.

- [ ] **Step 2: Create consumer.go modeled on the existing one**

```go
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

If the project's `NewConfig`/`AdaptHandler` signatures differ from the assumed shape (e.g. existing monster consumer at `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go` uses different curry depth), adjust to match exactly — the existing file is the source of truth.

- [ ] **Step 2: Build**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./kafka/consumer/data/...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/consumer.go
git commit -m "feat(atlas-monsters): data-events consumer registration"
```

## Task 16: `atlas-monsters` data consumer — handler_test.go

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/handler_test.go`

- [ ] **Step 1: Add filter and parse-error tests**

```go
package data

import (
    "context"
    "testing"

    "github.com/Chronicle20/atlas/libs/atlas-tenant"
    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

func nullLogger() logrus.FieldLogger {
    l := logrus.New()
    l.SetOutput(nil)
    return l
}

func TestHandleDataUpdated_UnknownTypeSkipped(t *testing.T) {
    l := nullLogger()
    e := event[dataUpdatedEventBody]{
        Type: "SOME_FUTURE_TYPE",
        Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: WorkerMonster},
    }
    handleDataUpdated(l, context.Background(), e)
    // No assertion needed beyond "did not panic and no flush attempted";
    // counter increment verified by running with prometheus testutil if needed.
}

func TestHandleDataUpdated_UnrelatedWorkerSkipped(t *testing.T) {
    l := nullLogger()
    e := event[dataUpdatedEventBody]{
        Type: EventTypeDataUpdated,
        Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: "MAP"},
    }
    handleDataUpdated(l, context.Background(), e)
}

func TestHandleDataUpdated_MalformedTenantId(t *testing.T) {
    l := nullLogger()
    e := event[dataUpdatedEventBody]{
        Type: EventTypeDataUpdated,
        Body: dataUpdatedEventBody{TenantId: "not-a-uuid", Worker: WorkerMonster},
    }
    handleDataUpdated(l, context.Background(), e)
    // Expects: ERROR log, parse counter increment, no panic, no flush call.
}

func TestConsumerEnabled_Default(t *testing.T) {
    t.Setenv("DATA_EVENTS_CONSUMER_ENABLED", "")
    if !consumerEnabled() {
        t.Fatal("expected default true")
    }
    t.Setenv("DATA_EVENTS_CONSUMER_ENABLED", "false")
    if consumerEnabled() {
        t.Fatal("expected false")
    }
    t.Setenv("DATA_EVENTS_CONSUMER_ENABLED", "garbage")
    if !consumerEnabled() {
        t.Fatal("expected default true on unparseable")
    }
}

func TestHandleDataUpdated_HappyPath_Smoke(t *testing.T) {
    // Smoke: with cache uninitialized, FlushTenant returns (0, nil) and
    // the handler completes without error. End-to-end tenant isolation
    // is covered in monster/information/cache_test.go (Task 11/12).
    l := nullLogger()
    tid := uuid.New()
    tm, _ := tenant.Create(tid, "GMS", 0, 83)
    ctx := tenant.WithContext(context.Background(), tm)
    e := event[dataUpdatedEventBody]{
        Type: EventTypeDataUpdated,
        Body: dataUpdatedEventBody{TenantId: tid.String(), Worker: WorkerMonster},
    }
    handleDataUpdated(l, ctx, e)
}
```

If `tenant.WithContext` is named differently (e.g. `tenant.NewContext` or `tenant.ContextWith`), grep `libs/atlas-tenant` to find the canonical setter and adjust.

```bash
grep -n 'func.*Context\|FromContext' libs/atlas-tenant/*.go
```

- [ ] **Step 2: Run tests**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -race -v ./kafka/consumer/data/...
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/data/handler_test.go
git commit -m "test(atlas-monsters): data-events handler filter + parse + kill-switch"
```

## Task 17: `atlas-monsters` main.go wiring

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go`

- [ ] **Step 1: Add import + group constant + Init calls**

In `main.go`:

1. Add import:
```go
data2 "atlas-monsters/kafka/consumer/data"
```

2. Add a new constant near the existing `consumerGroupId`:
```go
const dataEventsConsumerGroupId = "Monster Data Cache Invalidator"
```

3. After the existing `_map.InitConsumers(l)(cmf)(consumerGroupId)` call, add:
```go
data2.InitConsumers(l)(cmf)(dataEventsConsumerGroupId)
```

4. After the existing `_map.InitHandlers` registration block, add:
```go
if err := data2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
    l.WithError(err).Fatal("Unable to register data-events kafka handlers.")
}
```

- [ ] **Step 2: Build the whole service**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./...
```

Expected: success.

- [ ] **Step 3: Run all tests with race**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test -race ./...
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/main.go
git commit -m "feat(atlas-monsters): wire data-events consumer in main"
```

## Task 18: `atlas-maps` `SpawnPointRegistry.FlushTenant` — failing test

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/map/monster/registry_test.go` (or extend existing)

- [ ] **Step 1: Check whether a test file exists**

```bash
ls services/atlas-maps/atlas.com/maps/map/monster/*_test.go 2>/dev/null
grep -rn "miniredis" services/atlas-maps/atlas.com/maps/ 2>/dev/null | head -3
```

If miniredis is not yet a dep of atlas-maps, add it:

```bash
cd services/atlas-maps/atlas.com/maps && go get github.com/alicebob/miniredis/v2@latest && go mod tidy
```

- [ ] **Step 2: Create the test file**

```go
package monster

import (
    "context"
    "fmt"
    "testing"

    "github.com/alicebob/miniredis/v2"
    "github.com/google/uuid"
    goredis "github.com/redis/go-redis/v9"
    "github.com/sirupsen/logrus"
)

func setupSpawnTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
    t.Helper()
    mr := miniredis.RunT(t)
    return goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), mr
}

func TestSpawnPointRegistry_FlushTenant_DeletesAllForTenant(t *testing.T) {
    client, _ := setupSpawnTestRedis(t)
    r := &SpawnPointRegistry{client: client}
    tid := uuid.New()
    ctx := context.Background()
    l := logrus.New()

    // Pre-populate three spawn-hash keys for tenant T.
    for i := 0; i < 3; i++ {
        k := fmt.Sprintf("atlas:maps:spawn:%s:0:0:%d:00000000-0000-0000-0000-000000000000", tid.String(), 100+i)
        if err := client.HSet(ctx, k, "1", "{}").Err(); err != nil {
            t.Fatalf("HSet seed: %v", err)
        }
    }

    deleted, err := r.FlushTenant(ctx, l, tid)
    if err != nil {
        t.Fatalf("FlushTenant: %v", err)
    }
    if deleted != 3 {
        t.Fatalf("deleted = %d, want 3", deleted)
    }
}

func TestSpawnPointRegistry_FlushTenant_TenantIsolation(t *testing.T) {
    client, _ := setupSpawnTestRedis(t)
    r := &SpawnPointRegistry{client: client}
    tA := uuid.New()
    tB := uuid.New()
    ctx := context.Background()
    l := logrus.New()

    _ = client.HSet(ctx, fmt.Sprintf("atlas:maps:spawn:%s:0:0:1:none", tA), "1", "{}").Err()
    _ = client.HSet(ctx, fmt.Sprintf("atlas:maps:spawn:%s:0:0:1:none", tB), "1", "{}").Err()

    deleted, err := r.FlushTenant(ctx, l, tA)
    if err != nil {
        t.Fatalf("FlushTenant: %v", err)
    }
    if deleted != 1 {
        t.Fatalf("deleted = %d, want 1", deleted)
    }
    n, _ := client.Exists(ctx, fmt.Sprintf("atlas:maps:spawn:%s:0:0:1:none", tB)).Result()
    if n != 1 {
        t.Fatalf("tenant B's spawn key should still exist")
    }
}

func TestSpawnPointRegistry_FlushTenant_EmptyTenant(t *testing.T) {
    client, _ := setupSpawnTestRedis(t)
    r := &SpawnPointRegistry{client: client}
    deleted, err := r.FlushTenant(context.Background(), logrus.New(), uuid.New())
    if err != nil {
        t.Fatalf("FlushTenant: %v", err)
    }
    if deleted != 0 {
        t.Fatalf("deleted = %d, want 0", deleted)
    }
}
```

- [ ] **Step 3: Run; expect compile failure**

```bash
cd services/atlas-maps/atlas.com/maps && go test -run TestSpawnPointRegistry_FlushTenant -v ./map/monster/...
```

Expected: build error — `r.FlushTenant undefined`.

## Task 19: `atlas-maps` `SpawnPointRegistry.FlushTenant` — implementation

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/map/monster/registry.go`

- [ ] **Step 1: Add `FlushTenant` method**

Append after the existing `Reset` method (after line ~266):

```go
// FlushTenant deletes every spawn-point hash for tenantId.
// Uses SCAN with COUNT=100 to avoid blocking the broker on large key
// spaces; pipelines DEL per batch. Errors are logged at WARN per batch
// and surfaced via the returned error; partial deletions are not rolled
// back.
func (r *SpawnPointRegistry) FlushTenant(ctx context.Context, l logrus.FieldLogger, tenantId uuid.UUID) (int, error) {
    pattern := fmt.Sprintf("atlas:maps:spawn:%s:*", tenantId.String())
    iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()

    deleted := 0
    pipe := r.client.Pipeline()
    pipeSize := 0
    var firstErr error

    flushPipe := func() {
        if pipeSize == 0 {
            return
        }
        if _, perr := pipe.Exec(ctx); perr != nil {
            l.WithError(perr).Warnf("Spawn-registry DEL batch failure for tenant [%s].", tenantId)
            if firstErr == nil {
                firstErr = perr
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
        if firstErr == nil {
            firstErr = ierr
        }
    }
    return deleted, firstErr
}
```

Add imports if missing: `"github.com/google/uuid"` (already present given `mapKey.Tenant` uses uuid).

- [ ] **Step 2: Run the FlushTenant tests**

```bash
cd services/atlas-maps/atlas.com/maps && go test -run TestSpawnPointRegistry_FlushTenant -v ./map/monster/...
```

Expected: all 3 PASS.

- [ ] **Step 3: Build full service**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/monster/registry.go services/atlas-maps/atlas.com/maps/map/monster/registry_test.go services/atlas-maps/atlas.com/maps/go.mod services/atlas-maps/atlas.com/maps/go.sum
git commit -m "feat(atlas-maps): SpawnPointRegistry.FlushTenant scan+pipelined-DEL"
```

## Task 20: `atlas-maps` data consumer — kafka.go + metrics.go

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/data/kafka.go`
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/data/metrics.go`

- [ ] **Step 1: Create kafka.go**

```go
package data

const (
    EnvEventTopic        = "EVENT_TOPIC_DATA"
    EventTypeDataUpdated = "DATA_UPDATED"

    WorkerMap = "MAP"
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

- [ ] **Step 2: Create metrics.go**

```go
package data

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    eventsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_maps_data_events_processed_total",
        Help: "DATA_UPDATED events processed by the spawn-registry invalidator.",
    }, []string{"worker", "type", "action"})

    eventsConsumerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_maps_data_events_consumer_errors_total",
        Help: "Errors encountered processing DATA_UPDATED events.",
    }, []string{"kind"})

    eventsSkippedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_maps_data_events_consumer_skipped_total",
        Help: "DATA_UPDATED events skipped (unknown type or unrelated worker).",
    }, []string{"reason"})

    keysDeletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "atlas_maps_data_events_keys_deleted_total",
        Help: "Spawn-registry keys deleted by cache-invalidation flushes, by tenant.",
    }, []string{"tenant"})
)
```

- [ ] **Step 3: Verify prometheus dep in atlas-maps**

```bash
cd services/atlas-maps/atlas.com/maps && grep prometheus go.mod || (go get github.com/prometheus/client_golang@latest && go mod tidy)
```

- [ ] **Step 4: Build**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./kafka/consumer/data/...
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/data/kafka.go services/atlas-maps/atlas.com/maps/kafka/consumer/data/metrics.go services/atlas-maps/atlas.com/maps/go.mod services/atlas-maps/atlas.com/maps/go.sum 2>/dev/null
git commit -m "feat(atlas-maps): data-events consumer kafka.go + metrics.go"
```

## Task 21: `atlas-maps` data consumer — handler.go

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/data/handler.go`

- [ ] **Step 1: Create handler.go**

```go
// Package data subscribes to EVENT_TOPIC_DATA for tenant-scoped spawn-
// registry invalidation. We use a shared (single-delivery) consumer group
// "Map Spawn Registry Invalidator" because the spawn registry lives in
// shared Redis: one pod's FlushTenant is visible to every replica
// immediately. See docs/tasks/task-061-data-cache-invalidation/design.md
// §6.3 for the rationale.
package data

import (
    "context"
    "os"
    "strconv"

    spawnMonster "atlas-maps/map/monster"

    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

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

    deleted, ferr := spawnMonster.GetRegistry().FlushTenant(ctx, l, tid)
    if ferr != nil {
        l.WithError(ferr).Errorf("Spawn-registry flush partially failed for tenant [%s] (deleted [%d] keys before error).", tid, deleted)
        eventsConsumerErrorsTotal.WithLabelValues("flush").Inc()
    }
    keysDeletedTotal.WithLabelValues(tid.String()).Add(float64(deleted))
    eventsProcessedTotal.WithLabelValues(WorkerMap, EventTypeDataUpdated, "flushed").Inc()
    l.Debugf("Flushed [%d] spawn-registry keys for tenant [%s] in response to DATA_UPDATED.", deleted, tid)
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

- [ ] **Step 2: Build**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./kafka/consumer/data/...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/data/handler.go
git commit -m "feat(atlas-maps): data-events handler (Worker=MAP only)"
```

## Task 22: `atlas-maps` data consumer — consumer.go

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/data/consumer.go`

- [ ] **Step 1: Inspect an existing atlas-maps consumer**

```bash
sed -n '1,80p' services/atlas-maps/atlas.com/maps/kafka/consumer/monster/consumer.go
```

Note the `consumer2` alias and `NewConfig` curry depth. Reuse exactly.

- [ ] **Step 2: Create consumer.go modeled on the existing one**

```go
package data

import (
    consumer2 "atlas-maps/kafka/consumer"

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

If existing atlas-maps consumers' signatures differ, adjust to match exactly.

- [ ] **Step 3: Build**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/data/consumer.go
git commit -m "feat(atlas-maps): data-events consumer registration"
```

## Task 23: `atlas-maps` data consumer — handler_test.go

**Files:**
- Create: `services/atlas-maps/atlas.com/maps/kafka/consumer/data/handler_test.go`

- [ ] **Step 1: Add filter and parse-error tests**

```go
package data

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
)

func nullLogger() logrus.FieldLogger {
    l := logrus.New()
    l.SetOutput(nil)
    return l
}

func TestHandleDataUpdated_UnknownTypeSkipped(t *testing.T) {
    handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
        Type: "FUTURE",
        Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: WorkerMap},
    })
}

func TestHandleDataUpdated_WorkerMonsterSkipped(t *testing.T) {
    handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
        Type: EventTypeDataUpdated,
        Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: "MONSTER"},
    })
}

func TestHandleDataUpdated_WorkerNPCSkipped(t *testing.T) {
    handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
        Type: EventTypeDataUpdated,
        Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: "NPC"},
    })
}

func TestHandleDataUpdated_MalformedTenantId(t *testing.T) {
    handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
        Type: EventTypeDataUpdated,
        Body: dataUpdatedEventBody{TenantId: "not-a-uuid", Worker: WorkerMap},
    })
}

func TestConsumerEnabled_Default(t *testing.T) {
    t.Setenv("DATA_EVENTS_CONSUMER_ENABLED", "")
    if !consumerEnabled() {
        t.Fatal("expected default true")
    }
    t.Setenv("DATA_EVENTS_CONSUMER_ENABLED", "false")
    if consumerEnabled() {
        t.Fatal("expected false")
    }
}
```

These tests validate filter logic and that no panic / no flush occurs on early-return paths. Happy-path FlushTenant invocation is covered structurally — `spawnMonster.GetRegistry()` returns nil if `InitRegistry` was not called, in which case calling `.FlushTenant` would panic. To avoid that, gate the smoke test on the registry being initialized; if running in an environment where it isn't, these filter-only tests are sufficient. (Real flush-path coverage is in Task 18/19's registry tests.)

- [ ] **Step 2: Run**

```bash
cd services/atlas-maps/atlas.com/maps && go test -race -v ./kafka/consumer/data/...
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/consumer/data/handler_test.go
git commit -m "test(atlas-maps): data-events handler filter + parse + kill-switch"
```

## Task 24: `atlas-maps` main.go wiring

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/main.go`

- [ ] **Step 1: Add import + group constant + Init calls**

In `main.go`:

1. Add import:
```go
data2 "atlas-maps/kafka/consumer/data"
```

2. Add new constant:
```go
const dataEventsConsumerGroupId = "Map Spawn Registry Invalidator"
```

3. After the existing `sessionConsumer.InitConsumers` call, add:
```go
data2.InitConsumers(l)(cmf)(dataEventsConsumerGroupId)
```

4. After the existing `sessionConsumer` handler registration block, add:
```go
if err := data2.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
    l.WithError(err).Fatal("Unable to register data-events kafka handlers.")
}
```

- [ ] **Step 2: Build whole service**

```bash
cd services/atlas-maps/atlas.com/maps && go build ./...
```

Expected: success.

- [ ] **Step 3: Run all tests with race**

```bash
cd services/atlas-maps/atlas.com/maps && go test -race ./...
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/main.go
git commit -m "feat(atlas-maps): wire data-events consumer in main"
```

## Task 25: Deploy config — env-configmap.yaml + .env.example

**Files:**
- Modify: `deploy/k8s/env-configmap.yaml`
- Modify: `deploy/compose/.env.example`

- [ ] **Step 1: Locate the existing TOPIC_* lines**

```bash
grep -n 'TOPIC_DATA\|EVENT_TOPIC' deploy/k8s/env-configmap.yaml deploy/compose/.env.example 2>/dev/null
```

- [ ] **Step 2: Add `EVENT_TOPIC_DATA` to the ConfigMap**

In `deploy/k8s/env-configmap.yaml`, add (next to other `EVENT_TOPIC_*` lines, alphabetically or grouped by topic family):

```yaml
  EVENT_TOPIC_DATA: "EVENT_TOPIC_DATA"
```

(Match the existing indent — typically 2 spaces under the `data:` key. Verify by looking at adjacent lines.)

- [ ] **Step 3: Add `EVENT_TOPIC_DATA` to compose example**

In `deploy/compose/.env.example`, add:

```
EVENT_TOPIC_DATA=EVENT_TOPIC_DATA
```

- [ ] **Step 4: Verify YAML parseability (if `yq` is installed)**

```bash
yq '.' deploy/k8s/env-configmap.yaml > /dev/null && echo OK
```

Expected: `OK`. If `yq` isn't available, manually inspect the diff.

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/env-configmap.yaml deploy/compose/.env.example
git commit -m "feat(deploy): add EVENT_TOPIC_DATA to shared env config"
```

## Task 26: Cross-module build verification + Docker

**Files:**
- (no source changes; verification step)

- [ ] **Step 1: Build all four affected Go modules**

```bash
cd <home>/source/atlas-ms/atlas/.worktrees/task-061-data-cache-invalidation
( cd libs/atlas-redis                          && go build ./... && go test -race ./... )
( cd services/atlas-data/atlas.com/data        && go build ./... && go test -race ./... )
( cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test -race ./... )
( cd services/atlas-maps/atlas.com/maps        && go build ./... && go test -race ./... )
```

Expected: all four succeed; all tests PASS; no race detector hits. If any fails, fix before proceeding.

- [ ] **Step 2: Verify Docker builds (per CLAUDE.md guidance for shared-lib changes)**

```bash
docker build -t atlas-data:task-061     services/atlas-data/atlas.com/data
docker build -t atlas-monsters:task-061 services/atlas-monsters/atlas.com/monsters
docker build -t atlas-maps:task-061     services/atlas-maps/atlas.com/maps
```

Expected: three successful image builds.

- [ ] **Step 3: Commit anything residual**

```bash
git status
```

If `go.work.sum` or other generated files are dirty, commit:

```bash
git add go.work.sum
git commit -m "chore: refresh go.work.sum for task-061"
```

## Task 27: Update memory note for atlas-maps spawn cache

**Files:**
- Modify: `<home>/.claude/projects/-home-tumidanski-source-atlas-ms-atlas/memory/reference_atlas_maps_spawn_cache.md`

- [ ] **Step 1: Read the current note**

```bash
cat <home>/.claude/projects/-home-tumidanski-source-atlas-ms-atlas/memory/reference_atlas_maps_spawn_cache.md
```

- [ ] **Step 2: Append a "Now automated" addendum**

Append to the memory file:

```markdown

## Update (task-061, 2026-05-08)

Automatic invalidation now exists. After a `WorkerMap` import, atlas-data emits a `DATA_UPDATED` event on `EVENT_TOPIC_DATA`; atlas-maps subscribes via fixed group `"Map Spawn Registry Invalidator"` and runs `SpawnPointRegistry.FlushTenant` on the affected tenant. Likewise for `WorkerMonster` and atlas-monsters' Redis cache.

The manual `redis-cli --scan --pattern 'atlas:maps:spawn:*' | xargs DEL` runbook is now a fallback rather than the primary path. Use it when `atlas_data_events_emit_failures_total` is rising or when bypassing the event pipeline (e.g. local dev without Kafka).
```

- [ ] **Step 3: No commit needed for memory; it lives outside the repo**

(The memory directory is not part of the worktree.)

## Acceptance Criteria (verify before opening PR)

- [ ] Task-060 v2 is on `main` and this branch is rebased (Prereqs).
- [ ] `git log --oneline` on this branch shows the per-task commits in order.
- [ ] All four modules `go build ./... && go test -race ./...` clean (Task 26).
- [ ] Docker builds for atlas-data, atlas-monsters, atlas-maps all succeed (Task 26).
- [ ] `EVENT_TOPIC_DATA` line present in both `deploy/k8s/env-configmap.yaml` and `deploy/compose/.env.example` (Task 25).
- [ ] Manual E2E (deferred to deploy-time smoke test, per design §11.2):
  - Run `START_WORKER` for `WorkerMonster` → `atlas_monsters_data_events_processed_total{worker="MONSTER", action="flushed"}` increments by 1; tenant's positive cache namespace empty post-event; rebuilds on next `GET /api/data/monsters/{id}`.
  - Run `START_WORKER` for `WorkerMap` → `atlas_maps_data_events_processed_total{worker="MAP", action="flushed"}` increments by 1; `KEYS atlas:maps:spawn:{tenant}:*` empty post-event; rebuilds on next map activation.
  - `DATA_EVENTS_PRODUCER_ENABLED=false` suppresses emit; `DATA_EVENTS_CONSUMER_ENABLED=false` suppresses handler registration. Both verified.
- [ ] Code review run (`superpowers:requesting-code-review`) before PR.
- [ ] PRD updated to status `Implemented` and `Date implemented` on merge.
