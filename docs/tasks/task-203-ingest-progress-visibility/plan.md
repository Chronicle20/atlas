# Ingest Progress Visibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make an ingest run's phase and per-worker progress visible in the Atlas UI, durable past Kubernetes Job deletion, scoped to the calling tenant, and blocking on the Baselines publish control.

**Architecture:** One Redis key per `(scope, region, major, minor)` holds a JSON run record listing the run phase plus an entry per registered worker. Three writers touch it — the REST pod at Job creation, the ingest pod at every worker transition, and the Watchdog on a stuck-Job deletion — through a new leaf package `atlas-data/ingestrun` that both runtime packages can import without a cycle. `GET /api/data/process` is rewritten to resolve the caller's scope, read that one record, cross-check a `running` record against the heartbeat and a label-selected Job list, and return a JSON:API `ingestRun` document. Two UI pages poll it on the existing five-second cadence through one shared presentational panel.

**Tech Stack:** Go 1.24 (`services/atlas-data`, `libs/atlas-redis`), `libs/atlas-redis` generic `Registry`, api2go `jsonapi`, `k8s.io/client-go` (+ `fake` clientset and `miniredis` in tests), React 19 + TypeScript + TanStack React Query + vitest/@testing-library (`services/atlas-ui`).

## Global Constraints

- **Redis access only via `libs/atlas-redis` registries.** `tools/redis-key-guard.sh` bans keyed go-redis commands outside `libs/atlas-redis`. No `EXPIRE`/`SET`/`GET` on a raw `*goredis.Client` anywhere in `services/atlas-data`.
- **No new goroutines.** All progress writes are synchronous at the transition point. `tools/goroutine-guard.sh` bans bare `go` statements; do not add one, and do not add a `//goroutine-guard:allow` comment.
- **Progress writes are best-effort.** A Redis failure MUST NOT fail, abort, or slow an ingest run beyond the 5-second per-write ceiling. Log at warn and continue. (PRD FR-2.5.)
- **The run record's key suffix uses the RAW scope** (`tenants/<uuid>` or `shared`); the Kubernetes label uses the SANITIZED scope (`tenants-<uuid>` / `shared`, via `sanitizeLabel`). Every place that crosses between them must convert explicitly. This asymmetry is the F-2 defect; do not reintroduce it.
- **Record TTL is 7 days**, refreshed on every write. Every mutation goes through `UpdateWithTTL` (never `Update`, which clears the TTL) or `PutWithTTL`.
- **`Record` mutations are copy-on-write and pure.** They run inside an optimistic-lock closure that may execute up to 1000 times; they must not mutate the input's backing array and must not read the clock.
- Worker roster comes from `workers.Registered` — never a hardcoded list of eleven names.
- No `// TODO`, stub, or 501 in any landed commit.
- Repo-relative paths only in committed files; never a literal `/home/<user>/…`.

### Design deviation to apply deliberately

The design's §7 error table says "`jc == nil` / k8s list error → serve the stored record; skip the cross-check", while §4.4 says `jc == nil` "degrades a `running` record to `unknown` only when the heartbeat is also absent". These conflict. **§4.4 wins** (it is the FR-4.3 requirement, and `unknown` is the conservative choice for the publish gate). Resolution, implemented in Task 7:

- fresh heartbeat present → `running` (no Kubernetes call at all)
- no fresh heartbeat, `jc == nil` → `unknown`
- no fresh heartbeat, Job list **errors** → `running` (a failed list is not evidence of absence)
- no fresh heartbeat, Job list succeeds with no live Job → `unknown`

---

## File Structure

**`libs/atlas-redis`**
- Modify `registry.go` — add `UpdateWithTTL`.
- Modify `registry_test.go` — TTL retention/clearing tests.

**`services/atlas-data/atlas.com/data`**
- Create `ingestrun/ingestrun.go` — `Record`, `Phase`, `WorkerState`, `WorkerEntry`, key helpers, both registry constructors, pure transition methods. Leaf package: imports only stdlib, `go-redis`, and `libs/atlas-redis`.
- Create `ingestrun/ingestrun_test.go`.
- Modify `data/workers/registry.go` — add `RegisteredNames()`.
- Create `data/progress.go` — `ProgressSink`, `noopSink`, `RunOption`, `WithProgress`, `runWithProgress`, `newRunConfig`.
- Create `data/progress_test.go`.
- Modify `data/runwz.go` — variadic `RunOption`, `runOne` routed through `runWithProgress`.
- Modify `runtime/rest/jobs.go` — drop the duplicated namespace/registry helpers, fix `ingestJobKeySuffixFromLabels`, add `IngestRegistries`, inject `INGEST_RUN_ID`, initialise the run record in `Create`.
- Modify `runtime/rest/jobs_test.go` — round-trip and run-record tests.
- Modify `runtime/rest/watchdog.go` — `DefaultWatchdogTimeoutSecs` const; `deleteStuckJob` writes phase `stuck`.
- Modify `runtime/rest/watchdog_test.go`.
- Create `runtime/rest/ingestrun_model.go` — `IngestRunRestModel` + jsonapi boilerplate.
- Modify `runtime/rest/resource.go` — `InitResource(jc, regs)`, `processCreate` on `wzinput.ResolveScope`, `processStatus` rewritten.
- Modify `runtime/rest/resource_test.go` — the FR-4 matrix.
- Create `runtime/ingest/progress.go` — `redisSink` (the Redis-backed `data.ProgressSink`), `Init`, `Finish`, run-id guard, context discipline.
- Create `runtime/ingest/progress_test.go`.
- Modify `runtime/ingest/heartbeat.go` — delete the duplicated namespace/registry helpers.
- Modify `runtime/ingest/run.go` — build the sink, `Init` before and `Finish` after `RunWorkers`.
- Modify `main.go` — hoist `rdb`, build `IngestRegistries`, new `InitResource` signature, `DefaultWatchdogTimeoutSecs`.
- Modify `services/atlas-data/docs/rest.md`.

**`services/atlas-ui/src`**
- Modify `services/api/seed.service.ts` — `IngestRun` types, `getIngestRun`, `getCanonicalIngestRun`.
- Modify `lib/hooks/api/useSeed.ts` — `useIngestRun`.
- Modify `lib/hooks/api/useCanonicalData.ts` — `useCanonicalIngestRun`.
- Create `components/features/setup/IngestProgressPanel.tsx` — presentational, no fetching.
- Create `components/features/setup/ingest-progress.ts` — `INGEST_BLOCKING_PHASES`, `ingestPublishBlockReason`.
- Create `components/features/setup/__tests__/IngestProgressPanel.test.tsx`.
- Create `components/features/setup/__tests__/ingest-progress.test.ts`.
- Modify `pages/SetupPage.tsx` — mount the panel for tenant scope.
- Modify `pages/BaselinesPage.tsx` — mount the panel for shared scope, extend `publishDisabled`.
- Modify `pages/__tests__/BaselinesPage.test.tsx`.

---

### Task 1: `Registry.UpdateWithTTL` in `libs/atlas-redis`

`Update`'s pipelined `Set` passes `0`, which in Redis **clears** any existing TTL. A key created with `PutWithTTL(7d)` becomes immortal on its first `Update`. This task adds the TTL-preserving variant every later task depends on. `Update` itself is left alone — changing its semantics would touch live monster-store state.

**Files:**
- Modify: `libs/atlas-redis/registry.go` (after `Update`, before `PutWithTTL`)
- Test: `libs/atlas-redis/registry_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `func (r *Registry[K, V]) UpdateWithTTL(ctx context.Context, key K, ttl time.Duration, fn func(V) V) (V, error)` — same contract as `Update` (returns `ErrNotFound` when the key is absent; `fn` may run up to `updateMaxRetries` times) but the write carries `ttl`.

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-redis/registry_test.go`:

```go
func TestRegistry_UpdateWithTTL_RetainsTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })
	if err := r.PutWithTTL(ctx, "key1", "value1", time.Hour); err != nil {
		t.Fatalf("PutWithTTL failed: %v", err)
	}

	got, err := r.UpdateWithTTL(ctx, "key1", time.Hour, func(v string) string { return v + "-updated" })
	if err != nil {
		t.Fatalf("UpdateWithTTL failed: %v", err)
	}
	if got != "value1-updated" {
		t.Fatalf("expected value1-updated, got %s", got)
	}

	stored, err := r.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if stored != "value1-updated" {
		t.Fatalf("stored value is %s, want value1-updated", stored)
	}

	// The whole point: SET without an expiry option clears the TTL, so an
	// UpdateWithTTL that forgot to pass ttl would leave this at 0.
	if ttl := mr.TTL(namespacedKey("test", "key1")); ttl <= 0 {
		t.Fatalf("TTL after UpdateWithTTL is %v, want > 0", ttl)
	}
}

// TestRegistry_Update_ClearsTTL pins the defect UpdateWithTTL exists to work
// around. If a future refactor makes Update preserve TTLs this test fails
// loudly rather than silently changing behaviour for every existing caller.
func TestRegistry_Update_ClearsTTL(t *testing.T) {
	client, mr := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })
	if err := r.PutWithTTL(ctx, "key1", "value1", time.Hour); err != nil {
		t.Fatalf("PutWithTTL failed: %v", err)
	}
	if _, err := r.Update(ctx, "key1", func(v string) string { return v + "-updated" }); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if ttl := mr.TTL(namespacedKey("test", "key1")); ttl != 0 {
		t.Fatalf("TTL after Update is %v, want 0 (Update clears TTLs)", ttl)
	}
}

func TestRegistry_UpdateWithTTL_NotFound(t *testing.T) {
	client, _ := setupTestRedis(t)
	ctx := context.Background()

	r := NewRegistry[string, string](client, "test", func(k string) string { return k })
	_, err := r.UpdateWithTTL(ctx, "missing", time.Hour, func(v string) string { return v })
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
```

Add `"errors"` to that file's import block if it is not already present.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd libs/atlas-redis && go test -run 'TestRegistry_UpdateWithTTL|TestRegistry_Update_ClearsTTL' ./...
```

Expected: FAIL — `r.UpdateWithTTL undefined (type *Registry[string, string] has no field or method UpdateWithTTL)`.

- [ ] **Step 3: Implement `UpdateWithTTL`**

Insert into `libs/atlas-redis/registry.go` immediately after `Update` (before `PutWithTTL`):

```go
// UpdateWithTTL is Update with an explicit expiry on the write.
//
// Update's pipelined Set passes 0, and in Redis a SET without an expiry option
// CLEARS any existing TTL — so a key created with PutWithTTL becomes immortal
// on its first Update. Callers whose key must keep expiring across mutations
// (e.g. a bounded-lifetime progress record) must use this variant and pass the
// same ttl on every write. Update's zero-TTL behaviour is deliberately left
// unchanged: every existing caller relies on it.
//
// fn may run multiple times (optimistic-lock retry) and must be pure.
func (r *Registry[K, V]) UpdateWithTTL(ctx context.Context, key K, ttl time.Duration, fn func(V) V) (V, error) {
	rk := namespacedKey(r.namespace, r.keyFn(key))

	var result V
	txFn := func(tx *goredis.Tx) error {
		data, err := tx.Get(ctx, rk).Bytes()
		if errors.Is(err, goredis.Nil) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		current, err := r.unmarshal(data)
		if err != nil {
			return err
		}

		result = fn(current)
		newData, err := r.marshal(result)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
			pipe.Set(ctx, rk, newData, ttl)
			return nil
		})
		return err
	}

	for i := 0; i < updateMaxRetries; i++ {
		err := r.client.Watch(ctx, txFn, rk)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, goredis.TxFailedErr) {
			continue
		}
		return result, err
	}
	return result, fmt.Errorf("optimistic lock failed after %d retries", updateMaxRetries)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd libs/atlas-redis && go test -race ./... && go vet ./...
```

Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-redis/registry.go libs/atlas-redis/registry_test.go
git commit -m "feat(atlas-redis): add Registry.UpdateWithTTL

Update's pipelined SET passes a zero expiry, which clears any TTL the key
already had. Callers that need a bounded-lifetime value to survive
mutation now have a variant that carries the expiry on every write."
```

---

### Task 2: The `ingestrun` leaf package

The run record is written by `runtime/rest` (create, watchdog), written by `runtime/ingest` (worker transitions), and read by `runtime/rest` (status handler). It cannot live in either without one importing the other, so it gets its own leaf package. The two helpers currently copy-pasted between `runtime/rest/jobs.go` and `runtime/ingest/heartbeat.go` (`ingestJobNamespace`, `newIngestJobRegistry`) move here as the single definition.

**Files:**
- Create: `services/atlas-data/atlas.com/data/ingestrun/ingestrun.go`
- Test: `services/atlas-data/atlas.com/data/ingestrun/ingestrun_test.go`

**Interfaces:**
- Consumes: `libs/atlas-redis` `Registry`, `NewRegistry`.
- Produces (import path `atlas-data/ingestrun`):
  - `const Namespace = "data-ingest"`, `const RunKeySuffix = ":run"`, `const HeartbeatKeySuffix = ":updatedAt"`, `const RecordTTL = 7 * 24 * time.Hour`
  - `type Phase string` with `PhaseNone`, `PhaseRunning`, `PhaseSucceeded`, `PhaseFailed`, `PhaseStuck`, `PhaseUnknown`
  - `type WorkerState string` with `WorkerPending`, `WorkerRunning`, `WorkerSucceeded`, `WorkerFailed`, `WorkerSkipped`
  - `type WorkerEntry struct`, `type Record struct` (fields per code below)
  - `func KeySuffix(scope, region string, major, minor uint16) string`
  - `func NewJobRegistry(rdb *goredis.Client) *redis.Registry[string, string]`
  - `func NewRunRegistry(rdb *goredis.Client) *redis.Registry[string, Record]`
  - `func NewRecord(runId, jobName, scope, region, version, tenantId string, startedAt time.Time, workerNames []string) Record`
  - `func (r Record) WithRoster(names []string) Record`
  - `func (r Record) WithWorkerRunning(name string, at time.Time) Record`
  - `func (r Record) WithWorkerTerminal(name string, state WorkerState, at time.Time, errMsg string) Record`
  - `func (r Record) WithPhase(p Phase, at time.Time, reason string) Record`
  - `func (r Record) CompleteCount() int`
  - `func (r Record) IsTerminal() bool`

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-data/atlas.com/data/ingestrun/ingestrun_test.go`:

```go
package ingestrun

import (
	"testing"
	"time"
)

func names() []string { return []string{"STRING", "MAP", "ITEM"} }

func TestNewRecordSeedsRosterPending(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "tenants/t1", "GMS", "83.1", "t1", start, names())

	if rec.Phase != PhaseRunning {
		t.Fatalf("phase = %s, want %s", rec.Phase, PhaseRunning)
	}
	if len(rec.Workers) != 3 {
		t.Fatalf("roster size = %d, want 3", len(rec.Workers))
	}
	for _, w := range rec.Workers {
		if w.State != WorkerPending {
			t.Fatalf("worker %s state = %s, want %s", w.Name, w.State, WorkerPending)
		}
	}
	if rec.CompleteCount() != 0 {
		t.Fatalf("CompleteCount = %d, want 0", rec.CompleteCount())
	}
}

func TestKeySuffix(t *testing.T) {
	if got := KeySuffix("tenants/t1", "GMS", 83, 1); got != "tenants/t1:GMS:83.1" {
		t.Fatalf("got %q", got)
	}
	if got := KeySuffix("shared", "JMS", 185, 1); got != "shared:JMS:185.1" {
		t.Fatalf("got %q", got)
	}
}

func TestWorkerTransitions(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())

	rec = rec.WithWorkerRunning("STRING", start.Add(time.Second))
	if rec.Workers[0].State != WorkerRunning || rec.Workers[0].StartedAt == nil {
		t.Fatalf("STRING not running with a startedAt: %+v", rec.Workers[0])
	}

	rec = rec.WithWorkerTerminal("STRING", WorkerSucceeded, start.Add(2*time.Second), "")
	if rec.Workers[0].State != WorkerSucceeded || rec.Workers[0].FinishedAt == nil {
		t.Fatalf("STRING not succeeded with a finishedAt: %+v", rec.Workers[0])
	}
	if rec.Workers[0].Error != "" {
		t.Fatalf("succeeded worker carries error %q", rec.Workers[0].Error)
	}

	rec = rec.WithWorkerTerminal("MAP", WorkerFailed, start.Add(3*time.Second), "boom")
	if rec.Workers[1].State != WorkerFailed || rec.Workers[1].Error != "boom" {
		t.Fatalf("MAP not failed with error: %+v", rec.Workers[1])
	}
}

// A worker whose category is genuinely absent from a monolithic Data.wz counts
// as complete and must not stop the run reaching `succeeded` (PRD FR-1.5).
func TestSkippedCountsCompleteAndDoesNotBlockSucceeded(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithWorkerTerminal("STRING", WorkerSucceeded, start, "")
	rec = rec.WithWorkerTerminal("MAP", WorkerSkipped, start, "")
	rec = rec.WithWorkerTerminal("ITEM", WorkerSucceeded, start, "")

	if rec.CompleteCount() != 3 {
		t.Fatalf("CompleteCount = %d, want 3 (skipped counts as complete)", rec.CompleteCount())
	}
	rec = rec.WithPhase(PhaseSucceeded, start.Add(time.Minute), "")
	if rec.Phase != PhaseSucceeded {
		t.Fatalf("phase = %s, want %s", rec.Phase, PhaseSucceeded)
	}
	if rec.FinishedAt == nil {
		t.Fatal("terminal phase left FinishedAt nil")
	}
}

func TestWithPhaseNonTerminalLeavesFinishedAtNil(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithPhase(PhaseRunning, start.Add(time.Minute), "")
	if rec.FinishedAt != nil {
		t.Fatal("running phase set FinishedAt")
	}
	if !rec.StartedAt.Equal(start) {
		t.Fatal("WithPhase overwrote StartedAt")
	}
}

// The mutators run inside an optimistic-lock closure that may execute many
// times against the same input. They must not mutate the caller's slice.
func TestTransitionsDoNotMutateReceiver(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	orig := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())

	_ = orig.WithWorkerRunning("STRING", start)
	_ = orig.WithWorkerTerminal("MAP", WorkerFailed, start, "boom")
	_ = orig.WithPhase(PhaseFailed, start, "boom")

	for _, w := range orig.Workers {
		if w.State != WorkerPending {
			t.Fatalf("receiver mutated: %s is %s", w.Name, w.State)
		}
	}
	if orig.Phase != PhaseRunning || orig.FinishedAt != nil {
		t.Fatalf("receiver phase mutated: %+v", orig.Phase)
	}
}

// A record written by an older REST pod may not know about a worker the ingest
// pod is running. Transitions must record it rather than drop it.
func TestUnknownWorkerIsAppended(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithWorkerRunning("BRAND_NEW", start)
	if len(rec.Workers) != 4 || rec.Workers[3].Name != "BRAND_NEW" {
		t.Fatalf("unknown worker not appended: %+v", rec.Workers)
	}
}

func TestWithRosterAddsOnlyMissingNames(t *testing.T) {
	start := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	rec := NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", start, names())
	rec = rec.WithWorkerTerminal("STRING", WorkerSucceeded, start, "")

	rec = rec.WithRoster([]string{"STRING", "MAP", "ITEM", "COMMODITY"})
	if len(rec.Workers) != 4 {
		t.Fatalf("roster size = %d, want 4", len(rec.Workers))
	}
	if rec.Workers[0].State != WorkerSucceeded {
		t.Fatal("WithRoster reset an already-terminal worker")
	}
	if rec.Workers[3].Name != "COMMODITY" || rec.Workers[3].State != WorkerPending {
		t.Fatalf("new roster entry wrong: %+v", rec.Workers[3])
	}
}

func TestIsTerminal(t *testing.T) {
	cases := map[Phase]bool{
		PhaseNone: false, PhaseRunning: false, PhaseUnknown: false,
		PhaseSucceeded: true, PhaseFailed: true, PhaseStuck: true,
	}
	for p, want := range cases {
		if got := (Record{Phase: p}).IsTerminal(); got != want {
			t.Fatalf("IsTerminal(%s) = %v, want %v", p, got, want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./ingestrun/...
```

Expected: FAIL — build error, no non-test Go files in `ingestrun`.

- [ ] **Step 3: Implement the package**

Create `services/atlas-data/atlas.com/data/ingestrun/ingestrun.go`:

```go
// Package ingestrun holds the shared per-run progress record for atlas-data's
// WZ ingest, plus the Redis key/namespace helpers both runtime modes use.
//
// The record is written by runtime/rest (Job creation, Watchdog) and by
// runtime/ingest (per-worker transitions), and read by runtime/rest (the
// GET /api/data/process handler). Keeping it in a leaf package that imports
// only stdlib and libs/atlas-redis is what lets both runtime packages depend
// on it without an import cycle.
package ingestrun

import (
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// Namespace is the Redis namespace for every ingest/job-lifecycle key. The
// full key shape is <keyPrefix>:data-ingest:<suffix>, where <keyPrefix> comes
// from libs/atlas-redis KeyPrefix() (env-aware, so PR overlays are isolated).
const Namespace = "data-ingest"

// RunKeySuffix is appended to the per-run key suffix to address the run
// record, keeping it distinct from the job-name and heartbeat keys, which are
// typed Registry[string, string] and read by the Watchdog.
const RunKeySuffix = ":run"

// HeartbeatKeySuffix addresses the ingest pod's liveness timestamp.
const HeartbeatKeySuffix = ":updatedAt"

// RecordTTL bounds how long a run record survives. Refreshed on every write,
// so an in-flight run cannot expire mid-run. Redis is not a durable store: an
// eviction loses the record and the endpoint then reports PhaseNone, which is
// deliberately non-blocking for the baseline publish control.
const RecordTTL = 7 * 24 * time.Hour

// Phase is the overall state of an ingest run.
type Phase string

const (
	// PhaseNone means no run record exists for the triple.
	PhaseNone Phase = "none"
	// PhaseRunning means a run is in flight.
	PhaseRunning Phase = "running"
	// PhaseSucceeded means every worker finished without error.
	PhaseSucceeded Phase = "succeeded"
	// PhaseFailed means the ingest process returned an error.
	PhaseFailed Phase = "failed"
	// PhaseStuck means the Watchdog deleted the Job for heartbeat staleness.
	PhaseStuck Phase = "stuck"
	// PhaseUnknown is computed at read time: the record says running but
	// neither a fresh heartbeat nor a live Job corroborates it. Never stored.
	PhaseUnknown Phase = "unknown"
)

// WorkerState is the state of one registered ingest worker within a run.
type WorkerState string

const (
	WorkerPending   WorkerState = "pending"
	WorkerRunning   WorkerState = "running"
	WorkerSucceeded WorkerState = "succeeded"
	WorkerFailed    WorkerState = "failed"
	// WorkerSkipped is the ErrCategoryAbsent case: a category genuinely
	// missing from a monolithic Data.wz (v12 has no Quest). A skipped worker
	// does not make a run non-succeeded.
	WorkerSkipped WorkerState = "skipped"
)

// WorkerEntry is one worker's slot in a run record.
type WorkerEntry struct {
	Name       string      `json:"name"`
	State      WorkerState `json:"state"`
	StartedAt  *time.Time  `json:"startedAt,omitempty"`
	FinishedAt *time.Time  `json:"finishedAt,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// Record is the whole run: one Redis key, read with a single Get.
//
// It is a plain serialisable DTO rather than an immutable model with a
// Builder: it is mutated inside an optimistic-lock closure that may re-run
// many times, and it guards no domain invariant. Mutation is still confined —
// every transition is a With* method returning a modified copy, so the
// closure stays a pure function of its input.
type Record struct {
	RunId      string        `json:"runId"`
	JobName    string        `json:"jobName"`
	Scope      string        `json:"scope"`
	Region     string        `json:"region"`
	Version    string        `json:"version"`
	Tenant     string        `json:"tenant,omitempty"`
	Phase      Phase         `json:"phase"`
	StartedAt  time.Time     `json:"startedAt"`
	FinishedAt *time.Time    `json:"finishedAt,omitempty"`
	Reason     string        `json:"reason,omitempty"`
	Workers    []WorkerEntry `json:"workers"`
}

// KeySuffix returns the per-run key suffix. The full Redis key is
// <prefix>:data-ingest:<suffix>[:run|:updatedAt].
//
// scope here is the RAW scope ("shared" or "tenants/<uuid>"), never the
// sanitized Kubernetes label form — see runtime/rest/jobs.go.
func KeySuffix(scope, region string, major, minor uint16) string {
	return fmt.Sprintf("%s:%s:%d.%d", scope, region, major, minor)
}

// NewJobRegistry returns the env-global Registry for the job-name and
// heartbeat keys. The keyFn is the identity so callers supply the full suffix.
func NewJobRegistry(rdb *goredis.Client) *redis.Registry[string, string] {
	return redis.NewRegistry[string, string](rdb, Namespace, func(s string) string { return s })
}

// NewRunRegistry returns the env-global Registry for run records. Values are
// JSON-marshalled by the Registry itself.
func NewRunRegistry(rdb *goredis.Client) *redis.Registry[string, Record] {
	return redis.NewRegistry[string, Record](rdb, Namespace, func(s string) string { return s })
}

// NewRecord seeds a fresh running record with every roster name pending.
func NewRecord(runId, jobName, scope, region, version, tenantId string, startedAt time.Time, workerNames []string) Record {
	ws := make([]WorkerEntry, 0, len(workerNames))
	for _, n := range workerNames {
		ws = append(ws, WorkerEntry{Name: n, State: WorkerPending})
	}
	return Record{
		RunId:     runId,
		JobName:   jobName,
		Scope:     scope,
		Region:    region,
		Version:   version,
		Tenant:    tenantId,
		Phase:     PhaseRunning,
		StartedAt: startedAt.UTC(),
		Workers:   ws,
	}
}

// copyWorkers returns a shallow copy of r with its own Workers backing array.
// Every With* method starts here so a retried optimistic-lock closure cannot
// corrupt the value it was handed.
func (r Record) copyWorkers() Record {
	out := r
	out.Workers = append([]WorkerEntry(nil), r.Workers...)
	return out
}

func (r Record) indexOf(name string) int {
	for i := range r.Workers {
		if r.Workers[i].Name == name {
			return i
		}
	}
	return -1
}

// WithRoster appends any name not already present, as pending. Existing
// entries — including already-terminal ones — are left untouched.
func (r Record) WithRoster(names []string) Record {
	out := r.copyWorkers()
	for _, n := range names {
		if out.indexOf(n) < 0 {
			out.Workers = append(out.Workers, WorkerEntry{Name: n, State: WorkerPending})
		}
	}
	return out
}

// WithWorkerRunning marks name running at `at`, appending it if the record's
// roster predates it (an older REST pod wrote the record).
func (r Record) WithWorkerRunning(name string, at time.Time) Record {
	out := r.copyWorkers()
	t := at.UTC()
	if i := out.indexOf(name); i >= 0 {
		out.Workers[i].State = WorkerRunning
		out.Workers[i].StartedAt = &t
		out.Workers[i].FinishedAt = nil
		out.Workers[i].Error = ""
		return out
	}
	out.Workers = append(out.Workers, WorkerEntry{Name: name, State: WorkerRunning, StartedAt: &t})
	return out
}

// WithWorkerTerminal marks name with a terminal state at `at`. errMsg is
// stored only for WorkerFailed callers; pass "" otherwise.
func (r Record) WithWorkerTerminal(name string, state WorkerState, at time.Time, errMsg string) Record {
	out := r.copyWorkers()
	t := at.UTC()
	if i := out.indexOf(name); i >= 0 {
		out.Workers[i].State = state
		out.Workers[i].FinishedAt = &t
		out.Workers[i].Error = errMsg
		return out
	}
	out.Workers = append(out.Workers, WorkerEntry{Name: name, State: state, FinishedAt: &t, Error: errMsg})
	return out
}

// WithPhase sets the run phase, stamping FinishedAt for terminal phases and
// leaving StartedAt alone (the REST pod owns it — see design Q2). A non-empty
// reason overwrites any previous one; "" preserves it.
func (r Record) WithPhase(p Phase, at time.Time, reason string) Record {
	out := r.copyWorkers()
	out.Phase = p
	if reason != "" {
		out.Reason = reason
	}
	if out.IsTerminal() {
		t := at.UTC()
		out.FinishedAt = &t
	}
	return out
}

// IsTerminal reports whether the phase will not change again without a new run.
func (r Record) IsTerminal() bool {
	switch r.Phase {
	case PhaseSucceeded, PhaseFailed, PhaseStuck:
		return true
	default:
		return false
	}
}

// CompleteCount is the number of workers that will not change again.
func (r Record) CompleteCount() int {
	n := 0
	for _, w := range r.Workers {
		switch w.State {
		case WorkerSucceeded, WorkerSkipped, WorkerFailed:
			n++
		}
	}
	return n
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./ingestrun/... && go vet ./ingestrun/...
```

Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/ingestrun
git commit -m "feat(atlas-data): add ingestrun package for shared run records

Leaf package holding the per-run progress record, its phase/worker-state
enums, the Redis key helpers, and both registry constructors. Imports only
stdlib and libs/atlas-redis so runtime/rest and runtime/ingest can both
depend on it without a cycle."
```

---

### Task 3: `workers.RegisteredNames()`

The run record's worker roster must be derived from `workers.Registered`, so adding a twelfth worker changes the denominator without a second edit (PRD FR-1.6). `ingestrun` must not import `data/workers` (that would drag the WZ/gorm/minio dependency graph into a leaf package), so the roster is passed in as a `[]string` by each caller.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/data/workers/registry.go`
- Test: `services/atlas-data/atlas.com/data/data/workers/registry_test.go` (create)

**Interfaces:**
- Consumes: `workers.Registered`.
- Produces: `func RegisteredNames() []string` — the `Name()` of every registered worker, in `Registered` order.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-data/atlas.com/data/data/workers/registry_test.go`:

```go
package workers

import "testing"

// The run-record roster (task-203) is derived from this list, so a twelfth
// worker must widen the denominator without any second edit.
func TestRegisteredNamesMatchesRegistered(t *testing.T) {
	got := RegisteredNames()
	if len(got) != len(Registered) {
		t.Fatalf("RegisteredNames returned %d names, want %d", len(got), len(Registered))
	}
	for i, w := range Registered {
		if got[i] != w.Name() {
			t.Fatalf("index %d: got %q, want %q", i, got[i], w.Name())
		}
		if got[i] == "" {
			t.Fatalf("index %d: worker has an empty name", i)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd services/atlas-data/atlas.com/data && go test -run TestRegisteredNames ./data/workers/...
```

Expected: FAIL — `undefined: RegisteredNames`.

- [ ] **Step 3: Implement**

Append to `services/atlas-data/atlas.com/data/data/workers/registry.go`:

```go
// RegisteredNames returns the Name() of every registered worker, in Registered
// order. This is the denominator for ingest run-progress records: consumers
// take a plain []string so they need no dependency on this package's WZ/gorm/
// minio graph, and adding a worker widens the denominator with no second edit.
func RegisteredNames() []string {
	out := make([]string, 0, len(Registered))
	for _, w := range Registered {
		out = append(out, w.Name())
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./data/workers/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/workers/registry.go services/atlas-data/atlas.com/data/data/workers/registry_test.go
git commit -m "feat(atlas-data): add workers.RegisteredNames accessor"
```

---

### Task 4: Fix `ingestJobKeySuffixFromLabels`, inject `INGEST_RUN_ID`, initialise the run record

Three things in one task because they all live in `renderJob`/`Create` and share one test fixture.

**F-2 (blocking defect):** `Create` writes the heartbeat under the RAW scope (`tenants/<uuid>:GMS:83.1`); the Watchdog reconstructs the suffix from the Job's *labels*, where `scope` was written through `sanitizeLabel` (`tenants-<uuid>:GMS:83.1`). The two differ for every tenant-scope run, so `jobIsStuck` never finds a heartbeat for a tenant ingest and silently falls back to the creation timestamp. Fixing this is a prerequisite for Task 5 (the Watchdog becomes a *writer* of the run record; unfixed it would write to a key nothing reads). It also repairs the tenant-scope heartbeat check as a side effect — the intended behaviour of the existing timeout.

**§3.1 superseded-pod hazard:** nothing today tells an ingest pod which run it is. `Create` now generates a run id and injects it as `INGEST_RUN_ID`; the ingest pod (Task 7) drops any write whose stored `runId` differs.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/runtime/rest/jobs.go`
- Test: `services/atlas-data/atlas.com/data/runtime/rest/jobs_test.go`

**Interfaces:**
- Consumes: `ingestrun.KeySuffix`, `ingestrun.NewJobRegistry`, `ingestrun.NewRunRegistry`, `ingestrun.NewRecord`, `ingestrun.RecordTTL`, `ingestrun.RunKeySuffix`, `ingestrun.HeartbeatKeySuffix`, `workers.RegisteredNames`.
- Produces:
  - `type IngestRegistries struct { Job *redis.Registry[string, string]; Run *redis.Registry[string, ingestrun.Record] }`
  - `func NewIngestRegistries(rdb *goredis.Client) *IngestRegistries`
  - `JobCreator` gains field `RunRegistry *redis.Registry[string, ingestrun.Record]`
  - `func ingestJobKeySuffixFromLabels(j *batchv1.Job) string` — now returns the RAW-scope suffix
  - `renderJob(template, namespace, scope, region string, major, minor uint16, tenantId, traceparent, controllerImage, runId string) *batchv1.Job`
  - Package-local `ingestJobNamespace` and `newIngestJobRegistry` are **deleted** (moved to `ingestrun`).

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-data/atlas.com/data/runtime/rest/jobs_test.go` (add imports `"strings"`, `"time"`, `"github.com/alicebob/miniredis/v2"`, `goredis "github.com/redis/go-redis/v9"`, `"atlas-data/ingestrun"`, `"atlas-data/data/workers"` as needed):

```go
// The Redis key suffix uses the RAW scope; the Kubernetes label uses the
// sanitized form. Reconstructing the suffix from labels must undo that
// sanitisation or the Watchdog reads a key nobody writes (design F-2).
func TestIngestJobKeySuffixFromLabelsRoundTrips(t *testing.T) {
	cases := []struct {
		name     string
		scope    string
		tenantId string
	}{
		{"shared", "shared", ""},
		{"tenant", "tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := renderJob(testTemplate(), "ns", tc.scope, "GMS", 83, 1, tc.tenantId, "", "", "run-1")
			want := ingestrun.KeySuffix(tc.scope, "GMS", 83, 1)
			if got := ingestJobKeySuffixFromLabels(job); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestIngestJobKeySuffixFromLabelsRejectsIncompleteLabels(t *testing.T) {
	// Non-shared scope with no tenant label cannot be reconstructed; the
	// contract is to skip (return "") rather than guess.
	j := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
		"scope": "tenants-aaaaaaaa", "region": "GMS", "version": "83.1",
	}}}
	if got := ingestJobKeySuffixFromLabels(j); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestRenderJobInjectsRunId(t *testing.T) {
	job := renderJob(testTemplate(), "ns", "shared", "GMS", 83, 1, "", "", "", "run-abc")
	var found string
	for _, e := range job.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "INGEST_RUN_ID" {
			found = e.Value
		}
	}
	if found != "run-abc" {
		t.Fatalf("INGEST_RUN_ID = %q, want run-abc", found)
	}
}

func TestJobCreatorCreateInitialisesRunRecord(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	regs := NewIngestRegistries(rdb)

	cs := fake.NewSimpleClientset()
	jc := &JobCreator{
		K8s: cs, Namespace: "ns", Template: testTemplate(),
		Registry: regs.Job, RunRegistry: regs.Run,
	}
	scope := "tenants/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	name, err := jc.Create(context.Background(), scope, "GMS", 83, 1, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "")
	if err != nil {
		t.Fatal(err)
	}

	suffix := ingestrun.KeySuffix(scope, "GMS", 83, 1)
	rec, err := regs.Run.Get(context.Background(), suffix+ingestrun.RunKeySuffix)
	if err != nil {
		t.Fatalf("run record not written: %v", err)
	}
	if rec.Phase != ingestrun.PhaseRunning {
		t.Fatalf("phase = %s, want running", rec.Phase)
	}
	if rec.JobName != name {
		t.Fatalf("jobName = %q, want %q", rec.JobName, name)
	}
	if rec.RunId == "" {
		t.Fatal("run record has no runId")
	}
	if rec.Scope != scope || rec.Region != "GMS" || rec.Version != "83.1" {
		t.Fatalf("record identity wrong: %+v", rec)
	}
	if len(rec.Workers) != len(workers.RegisteredNames()) {
		t.Fatalf("roster size = %d, want %d", len(rec.Workers), len(workers.RegisteredNames()))
	}
	for _, w := range rec.Workers {
		if w.State != ingestrun.WorkerPending {
			t.Fatalf("worker %s = %s, want pending", w.Name, w.State)
		}
	}
	if rec.StartedAt.IsZero() {
		t.Fatal("record has no startedAt")
	}

	// The record's runId must be the one the pod will read from its env.
	created, err := cs.BatchV1().Jobs("ns").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var envRunId string
	for _, e := range created.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "INGEST_RUN_ID" {
			envRunId = e.Value
		}
	}
	if envRunId != rec.RunId {
		t.Fatalf("job env runId %q != record runId %q", envRunId, rec.RunId)
	}

	// The run record must expire; PutWithTTL(RecordTTL) is what sets that up.
	if ttl := mr.TTL(strings.Join([]string{"atlas", ingestrun.Namespace, suffix + ingestrun.RunKeySuffix}, ":")); ttl <= 0 {
		t.Fatalf("run record TTL = %v, want > 0", ttl)
	}
}
```

Note on the last assertion: the key prefix is `libs/atlas-redis` `KeyPrefix()`, which is `"atlas"` when `ATLAS_ENV` is unset (the test environment). If the executor prefers not to hardcode it, import `redis "github.com/Chronicle20/atlas/libs/atlas-redis"` and build the key as `redis.KeyPrefix() + ":" + ingestrun.Namespace + ":" + suffix + ingestrun.RunKeySuffix`. Prefer that form.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test -run 'TestIngestJobKeySuffixFromLabels|TestRenderJobInjectsRunId|TestJobCreatorCreateInitialisesRunRecord' ./runtime/rest/...
```

Expected: FAIL — `NewIngestRegistries` undefined; `renderJob` called with 11 args, want 10; `JobCreator` has no field `RunRegistry`.

- [ ] **Step 3: Implement**

In `services/atlas-data/atlas.com/data/runtime/rest/jobs.go`:

**3a.** Replace the `ingestJobNamespace` const and `newIngestJobRegistry` func (lines 34–45) with the `IngestRegistries` bundle:

```go
// IngestRegistries bundles the two Redis registries the /data/process handlers
// read and write. They are constructed from the Redis client directly rather
// than hanging off JobCreator, because the status handler must still serve the
// stored run record when the Kubernetes client is unavailable and JobCreator
// is therefore nil (PRD FR-4.5).
type IngestRegistries struct {
	// Job holds the job-name and :updatedAt heartbeat keys.
	Job *redis.Registry[string, string]
	// Run holds the per-run progress record.
	Run *redis.Registry[string, ingestrun.Record]
}

// NewIngestRegistries builds both ingest registries over one Redis client.
// Returns nil for a nil client so callers can pass the result straight through.
func NewIngestRegistries(rdb *goredis.Client) *IngestRegistries {
	if rdb == nil {
		return nil
	}
	return &IngestRegistries{Job: ingestrun.NewJobRegistry(rdb), Run: ingestrun.NewRunRegistry(rdb)}
}
```

**3b.** Delete `ingestJobKeySuffix` (lines 47–51) and replace every call site with `ingestrun.KeySuffix`.

**3c.** Replace `ingestJobKeySuffixFromLabels`:

```go
// ingestJobKeySuffixFromLabels reconstructs the per-Job Redis key suffix from
// a Job's labels. Returns "" if the suffix cannot be reconstructed.
//
// The Redis keys are built from the RAW scope ("tenants/<uuid>"), but the
// Job's `scope` label went through sanitizeLabel, which maps '/' to '-'. The
// raw tenant id survives unsanitized in the `tenant` label, so that is what we
// rebuild from. Reading the `scope` label directly — as this did before
// task-203 — produced "tenants-<uuid>:…" and silently missed the heartbeat for
// every tenant-scope run.
func ingestJobKeySuffixFromLabels(j *batchv1.Job) string {
	scopeLabel, region, version := j.Labels["scope"], j.Labels["region"], j.Labels["version"]
	if scopeLabel == "" || region == "" || version == "" {
		return ""
	}
	scope := scopeLabel
	if scopeLabel != "shared" {
		tenantId := j.Labels["tenant"]
		if tenantId == "" {
			return ""
		}
		scope = "tenants/" + tenantId
	}
	return fmt.Sprintf("%s:%s:%s", scope, region, version)
}
```

**3d.** Add the `RunRegistry` field to `JobCreator`, immediately after `Registry`:

```go
	// RunRegistry is the env-prefixed store for per-run progress records.
	// Nil means run-record publishing is disabled (compose / test paths).
	RunRegistry *redis.Registry[string, ingestrun.Record]
```

**3e.** In `NewJobCreatorInClusterWithRedis`, replace the `var reg *redis.Registry[string, string]` block with:

```go
	regs := NewIngestRegistries(rdb)
	var jobReg *redis.Registry[string, string]
	var runReg *redis.Registry[string, ingestrun.Record]
	if regs != nil {
		jobReg, runReg = regs.Job, regs.Run
	}
```

and set `Registry: jobReg, RunRegistry: runReg` in the returned struct.

**3f.** Rewrite `Create`'s body after `if j.Template == nil { … }`:

```go
	// The ingest pod has no supported way to learn its own Job name, so the
	// run's identity is a value we mint here and inject as env. Every write
	// from the pod is guarded on it, so an operator re-triggering an ingest
	// while the previous pod is still alive cannot have the old pod stamp a
	// terminal phase over the new run (design §3.1).
	runId := uuid.NewString()
	job := renderJob(j.Template, j.Namespace, scope, region, major, minor, tenantId, traceparent, j.ControllerImage, runId)
	created, err := j.K8s.BatchV1().Jobs(j.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create job: %w", err)
	}
	suffix := ingestrun.KeySuffix(scope, region, major, minor)
	if j.Registry != nil {
		_ = j.Registry.PutWithTTL(ctx, suffix, created.Name, time.Hour)
		_ = j.Registry.PutWithTTL(ctx, suffix+ingestrun.HeartbeatKeySuffix, time.Now().UTC().Format(time.RFC3339), time.Hour)
	}
	if j.RunRegistry != nil {
		// Initialise (or reset) the record here so a run that dies before the
		// ingest pod's first write is still represented (PRD FR-3.2), and so
		// startedAt is stamped by one clock — this pod's — for both the
		// in-flight and terminal cases (design Q2).
		rec := ingestrun.NewRecord(
			runId, created.Name, scope, region,
			fmt.Sprintf("%d.%d", major, minor), tenantId,
			time.Now().UTC(), workers.RegisteredNames(),
		)
		_ = j.RunRegistry.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL)
	}
	return created.Name, nil
```

**3g.** Add the `runId string` parameter to `renderJob` (last position) and append the env var alongside `TENANT_ID`:

```go
	if runId != "" {
		envs = append(envs, corev1.EnvVar{Name: "INGEST_RUN_ID", Value: runId})
	}
```

**3h.** Update the import block: add `"atlas-data/data/workers"`, `"atlas-data/ingestrun"`, `"github.com/google/uuid"`.

**3i.** Update `runtime/rest/watchdog_test.go`'s `newTestRedis` helper — `newIngestJobRegistry` no longer exists:

```go
func newTestRedis(t *testing.T) (*goredis.Client, *redis.Registry[string, string]) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return rdb, ingestrun.NewJobRegistry(rdb)
}
```

Add `"atlas-data/ingestrun"` to that file's imports.

**3j.** In `runtime/ingest/heartbeat.go`, delete the `ingestJobNamespace` const and `newIngestJobRegistry` func (lines 25–34) and their now-unused `goredis`/`redis` imports if nothing else in the file needs them (`runHeartbeat`'s parameter still needs `redis`). Task 8 updates `run.go`'s call site; if the build breaks between commits here, change `run.go`'s `newIngestJobRegistry(rdb)` to `ingestrun.NewJobRegistry(rdb)` now (one-line change, add the import) so this task's commit compiles.

**3k.** Existing `watchdog_test.go` fixtures set the `scope` label to `"tenants-t1"` with no `tenant` label. Any such case must gain `"tenant": "t1"` or it will now (correctly) skip. Run the suite and fix the fixtures — do not weaken the assertion.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go test -race ./runtime/... ./ingestrun/... && go vet ./...
```

Expected: PASS, build and vet clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/runtime services/atlas-data/atlas.com/data/data/workers
git commit -m "fix(atlas-data): reconstruct the raw scope in ingest key suffixes

The Redis key suffix is built from the raw scope while the Job's label went
through sanitizeLabel, so the Watchdog's label-derived suffix never matched
a tenant-scope run's heartbeat and silently fell back to the creation
timestamp. Rebuild from the unsanitized tenant label instead, with a
round-trip regression test.

Also mints a run id per ingest, injects it as INGEST_RUN_ID, and
initialises the run record at Job creation."
```

---

### Task 5: Watchdog writes phase `stuck`

A Watchdog-deleted Job must leave a readable terminal outcome, not vanish. Per design Q1 the per-worker states are preserved exactly as the ingest pod left them — the worker still `running` when the Watchdog fired is the diagnostic signal, and marking it `failed` would assert something we do not know.

**Files:**
- Modify: `services/atlas-data/atlas.com/data/runtime/rest/watchdog.go`
- Modify: `services/atlas-data/atlas.com/data/main.go` (use the new const)
- Test: `services/atlas-data/atlas.com/data/runtime/rest/watchdog_test.go`

**Interfaces:**
- Consumes: `ingestJobKeySuffixFromLabels` (Task 4), `ingestrun.PhaseStuck`, `Record.WithPhase`, `Registry.UpdateWithTTL` (Task 1).
- Produces: `const DefaultWatchdogTimeoutSecs = 7200` in package `rest`; `Watchdog.runRegistry()`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/runtime/rest/watchdog_test.go`:

```go
func TestDeleteStuckJobWritesStuckRecord(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	regs := NewIngestRegistries(rdb)

	scope := "tenants/t1"
	suffix := ingestrun.KeySuffix(scope, "GMS", 83, 1)

	// A record left mid-run: STRING done, MAP still going.
	rec := ingestrun.NewRecord("run-1", "j1", scope, "GMS", "83.1", "t1",
		time.Now().UTC().Add(-time.Hour), []string{"STRING", "MAP"})
	rec = rec.WithWorkerTerminal("STRING", ingestrun.WorkerSucceeded, time.Now().UTC(), "")
	rec = rec.WithWorkerRunning("MAP", time.Now().UTC())
	if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}
	if err := regs.Job.PutWithTTL(ctx, suffix, "j1", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := regs.Job.PutWithTTL(ctx, suffix+ingestrun.HeartbeatKeySuffix, time.Now().UTC().Format(time.RFC3339), time.Hour); err != nil {
		t.Fatal(err)
	}

	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j1", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true", "scope": "tenants-t1",
			"region": "GMS", "version": "83.1", "tenant": "t1",
		},
	}}
	cs := fake.NewSimpleClientset(job)
	jc := &JobCreator{K8s: cs, Namespace: "ns", Registry: regs.Job, RunRegistry: regs.Run}
	w := Watchdog{L: logrus.New(), JobCreator: jc, TimeoutSecs: 1800}

	w.deleteStuckJob(ctx, job)

	got, err := regs.Run.Get(ctx, suffix+ingestrun.RunKeySuffix)
	if err != nil {
		t.Fatalf("run record gone: %v", err)
	}
	if got.Phase != ingestrun.PhaseStuck {
		t.Fatalf("phase = %s, want stuck", got.Phase)
	}
	if got.Reason == "" {
		t.Fatal("stuck record has no reason")
	}
	if got.FinishedAt == nil {
		t.Fatal("stuck record has no finishedAt")
	}
	// Q1: the per-worker states are preserved exactly.
	if got.Workers[0].State != ingestrun.WorkerSucceeded {
		t.Fatalf("STRING = %s, want succeeded", got.Workers[0].State)
	}
	if got.Workers[1].State != ingestrun.WorkerRunning {
		t.Fatalf("MAP = %s, want running (preserved, not rewritten)", got.Workers[1].State)
	}
	// The two heartbeat keys are still removed, as before.
	if _, err := regs.Job.Get(ctx, suffix); err == nil {
		t.Fatal("job-name key not removed")
	}
	if _, err := regs.Job.Get(ctx, suffix+ingestrun.HeartbeatKeySuffix); err == nil {
		t.Fatal("heartbeat key not removed")
	}
}

func TestDeleteStuckJobWithNoRecordIsQuiet(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	regs := NewIngestRegistries(rdb)

	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j1", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true", "scope": "shared", "region": "GMS", "version": "83.1",
		},
	}}
	cs := fake.NewSimpleClientset(job)
	jc := &JobCreator{K8s: cs, Namespace: "ns", Registry: regs.Job, RunRegistry: regs.Run}
	w := Watchdog{L: logrus.New(), JobCreator: jc, TimeoutSecs: 1800}

	// Must not panic and must not resurrect a record that was never written.
	w.deleteStuckJob(ctx, job)

	suffix := ingestrun.KeySuffix("shared", "GMS", 83, 1)
	if _, err := regs.Run.Get(ctx, suffix+ingestrun.RunKeySuffix); err == nil {
		t.Fatal("deleteStuckJob created a record where none existed")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test -run TestDeleteStuckJob ./runtime/rest/...
```

Expected: FAIL — the record's phase is still `running` (nothing writes `stuck`).

- [ ] **Step 3: Implement**

In `services/atlas-data/atlas.com/data/runtime/rest/watchdog.go`:

Add near the top, after the imports:

```go
// DefaultWatchdogTimeoutSecs is the maximum heartbeat staleness the Watchdog
// tolerates before deleting an ingest Job, and — because the status handler
// uses the same window to decide whether a `running` record is corroborated —
// the single definition of "how stale is too stale". The ingest pod refreshes
// its heartbeat every 30s (runtime/ingest/heartbeat.go), so anything over ~60s
// suffices in the happy path; 2 h is a generous margin for a wedged heartbeat
// goroutine or a transient Redis blip, and absorbs archive growth without a
// code change.
const DefaultWatchdogTimeoutSecs = 7200
```

Append the run-record write to `deleteStuckJob`, after the existing heartbeat-key removal block:

```go
	if rr := w.runRegistry(); rr != nil {
		if suffix := ingestJobKeySuffixFromLabels(j); suffix != "" {
			reason := fmt.Sprintf("watchdog deleted the ingest Job after %ds without a heartbeat", w.TimeoutSecs)
			now := time.Now().UTC()
			// Per-worker states are left exactly as the ingest pod wrote them:
			// the worker still `running` when the watchdog fired is the whole
			// diagnostic value, and marking it failed would assert something we
			// do not know.
			_, err := rr.UpdateWithTTL(ctx, suffix+ingestrun.RunKeySuffix, ingestrun.RecordTTL,
				func(rec ingestrun.Record) ingestrun.Record {
					return rec.WithPhase(ingestrun.PhaseStuck, now, reason)
				})
			if err != nil && !errors.Is(err, redis.ErrNotFound) && w.L != nil {
				w.L.WithError(err).Warnf("watchdog: stuck-record write failed for %s", suffix)
			}
		}
	}
```

Add the accessor next to `jobRegistry`:

```go
// runRegistry returns the JobCreator's run-record Registry, or nil if either
// the JobCreator or its registry is absent.
func (w Watchdog) runRegistry() *redis.Registry[string, ingestrun.Record] {
	if w.JobCreator == nil {
		return nil
	}
	return w.JobCreator.RunRegistry
}
```

Add `"errors"`, `"fmt"`, and `"atlas-data/ingestrun"` to the imports.

In `main.go`, replace the literal `TimeoutSecs: 7200` with `TimeoutSecs: restruntime.DefaultWatchdogTimeoutSecs` and trim the now-duplicated portion of the long comment above it down to a one-line pointer at the const.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go test -race ./runtime/... && go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/runtime/rest/watchdog.go services/atlas-data/atlas.com/data/runtime/rest/watchdog_test.go services/atlas-data/atlas.com/data/main.go
git commit -m "feat(atlas-data): watchdog records phase=stuck on job deletion

A watchdog-deleted ingest Job now leaves a durable terminal outcome with a
reason naming the timeout, so an operator sees an abnormal termination
instead of a run that silently disappears. Per-worker states are preserved
so the record still shows which worker was mid-flight."
```

---

### Task 6: `ProgressSink` and per-worker instrumentation in `data`

`runtime/ingest` imports `atlas-data/data`, so the `data` package must declare the interface and `runtime/ingest` supply the implementation (design F-4). Wiring is a variadic `RunOption` so existing callers — including `data/runwz_test.go` and the compose path — compile unchanged and default to a no-op sink; that is also what makes PRD FR-2.6 fall out for free.

The `ErrCategoryAbsent` classification moves into a small free function so the skip contract is testable without MinIO, a database, or Redis.

**Files:**
- Create: `services/atlas-data/atlas.com/data/data/progress.go`
- Modify: `services/atlas-data/atlas.com/data/data/runwz.go`
- Test: `services/atlas-data/atlas.com/data/data/progress_test.go`

**Interfaces:**
- Consumes: `workers.ErrCategoryAbsent`.
- Produces (package `data`):
  - `type ProgressSink interface { WorkerStarted(ctx context.Context, name string); WorkerFinished(ctx context.Context, name string, err error, skipped bool) }`
  - `type RunOption func(*runConfig)`
  - `func WithProgress(s ProgressSink) RunOption`
  - `func RunWorkers(l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, opts ...RunOption) func(ctx context.Context, p workers.Params) error`
  - unexported: `noopSink`, `runConfig`, `newRunConfig(opts []RunOption) runConfig`, `runWithProgress(ctx, sink, name, fn) error`

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-data/atlas.com/data/data/progress_test.go`:

```go
package data

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"atlas-data/data/workers"
)

type event struct {
	name    string
	state   string
	errText string
}

type recordingSink struct{ events []event }

func (s *recordingSink) WorkerStarted(_ context.Context, name string) {
	s.events = append(s.events, event{name: name, state: "running"})
}

func (s *recordingSink) WorkerFinished(_ context.Context, name string, err error, skipped bool) {
	e := event{name: name, state: "succeeded"}
	switch {
	case skipped:
		e.state = "skipped"
	case err != nil:
		e.state = "failed"
		e.errText = err.Error()
	}
	s.events = append(s.events, e)
}

func TestRunWithProgressSucceeded(t *testing.T) {
	s := &recordingSink{}
	err := runWithProgress(context.Background(), s, "MAP", func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	want := []event{{name: "MAP", state: "running"}, {name: "MAP", state: "succeeded"}}
	if fmt.Sprint(s.events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", s.events, want)
	}
}

func TestRunWithProgressFailed(t *testing.T) {
	s := &recordingSink{}
	boom := errors.New("boom")
	err := runWithProgress(context.Background(), s, "MAP", func(context.Context) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("error not propagated: %v", err)
	}
	if len(s.events) != 2 || s.events[1].state != "failed" || s.events[1].errText != "boom" {
		t.Fatalf("events = %v", s.events)
	}
}

// A category genuinely absent from a monolithic Data.wz (v12 has no Quest)
// records the worker as skipped and returns nil, so the run can still succeed.
func TestRunWithProgressSkipsCategoryAbsent(t *testing.T) {
	s := &recordingSink{}
	err := runWithProgress(context.Background(), s, "QUEST", func(context.Context) error {
		return fmt.Errorf("QUEST open Quest.wz: %w", workers.ErrCategoryAbsent)
	})
	if err != nil {
		t.Fatalf("got %v, want nil (a skipped worker must not fail the run)", err)
	}
	if len(s.events) != 2 || s.events[1].state != "skipped" {
		t.Fatalf("events = %v, want the second to be skipped", s.events)
	}
	if s.events[1].errText != "" {
		t.Fatalf("skipped worker carries error %q", s.events[1].errText)
	}
}

func TestNewRunConfigDefaultsToNoopSink(t *testing.T) {
	if _, ok := newRunConfig(nil).sink.(noopSink); !ok {
		t.Fatal("default sink is not noopSink")
	}
	if _, ok := newRunConfig([]RunOption{WithProgress(nil)}).sink.(noopSink); !ok {
		t.Fatal("WithProgress(nil) replaced the default sink")
	}
	s := &recordingSink{}
	if got := newRunConfig([]RunOption{WithProgress(s)}).sink; got != ProgressSink(s) {
		t.Fatalf("WithProgress did not install the sink, got %T", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test -run 'TestRunWithProgress|TestNewRunConfig' ./data/
```

Expected: FAIL — `undefined: runWithProgress`, `undefined: newRunConfig`, `undefined: noopSink`.

- [ ] **Step 3: Implement**

Create `services/atlas-data/atlas.com/data/data/progress.go`:

```go
package data

import (
	"context"
	"errors"

	"atlas-data/data/workers"
)

// ProgressSink receives per-worker lifecycle transitions from RunWorkers.
//
// Implementations return nothing on purpose: progress reporting is best-effort
// telemetry and must never fail, abort, or slow an ingest run (PRD FR-2.5).
// The Redis-backed implementation lives in runtime/ingest — this package
// cannot import it (runtime/ingest imports data), so the interface is declared
// here and satisfied there.
type ProgressSink interface {
	WorkerStarted(ctx context.Context, name string)
	// WorkerFinished reports a worker's terminal transition. skipped is true
	// for the ErrCategoryAbsent case, where err is nil and the run continues.
	WorkerFinished(ctx context.Context, name string, err error, skipped bool)
}

// noopSink is the default: it makes the compose and unit-test paths — where
// no run record exists — need no special-casing at the call sites.
type noopSink struct{}

func (noopSink) WorkerStarted(context.Context, string)               {}
func (noopSink) WorkerFinished(context.Context, string, error, bool) {}

type runConfig struct {
	sink ProgressSink
}

// RunOption configures RunWorkers. Variadic so every existing call site
// compiles unchanged.
type RunOption func(*runConfig)

// WithProgress routes per-worker transitions to s. A nil sink is ignored,
// leaving the no-op default in place.
func WithProgress(s ProgressSink) RunOption {
	return func(c *runConfig) {
		if s != nil {
			c.sink = s
		}
	}
}

func newRunConfig(opts []RunOption) runConfig {
	c := runConfig{sink: noopSink{}}
	for _, o := range opts {
		if o != nil {
			o(&c)
		}
	}
	return c
}

// runWithProgress invokes fn and reports its lifecycle to sink.
//
// The ErrCategoryAbsent contract lives here rather than inline in RunWorkers so
// it is assertable without MinIO, a database, or Redis: an absent category is
// reported as skipped and swallowed, because a category genuinely missing from
// a monolithic Data.wz (v12 has no Quest) must not fail the whole run.
func runWithProgress(ctx context.Context, sink ProgressSink, name string, fn func(context.Context) error) error {
	sink.WorkerStarted(ctx, name)
	err := fn(ctx)
	if errors.Is(err, workers.ErrCategoryAbsent) {
		sink.WorkerFinished(ctx, name, nil, true)
		return nil
	}
	sink.WorkerFinished(ctx, name, err, false)
	return err
}
```

In `services/atlas-data/atlas.com/data/data/runwz.go`, change the signature and route `runOne` through the helper:

```go
func RunWorkers(l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, opts ...RunOption) func(ctx context.Context, p workers.Params) error {
	return func(ctx context.Context, p workers.Params) error {
		defer workers.CloseMonolith()
		maxParallel := envInt("INGEST_MAX_PARALLEL", 4)
		cfg := newRunConfig(opts)
		…
		// runOne resolves the worker's archive (per-archive object or
		// monolithic Data.wz sub-view) and runs it under the given
		// (tenanted, cancellable) context. runWithProgress owns the
		// ErrCategoryAbsent skip contract (task-172 C-3.4) and reports both
		// transitions to the progress sink. This is the single chokepoint for
		// both the sequential prerequisite phase and the parallel fan-out.
		runOne := func(tctx context.Context, w workers.Worker) error {
			return runWithProgress(tctx, cfg.sink, w.Name(), func(tctx context.Context) error {
				wzFile, cleanup, err := workers.OpenArchive(tctx, l, mc, p, w.ArchiveName())
				if err != nil {
					if errors.Is(err, workers.ErrCategoryAbsent) {
						l.Warnf("%s: %s absent from monolithic Data.wz — skipping worker (category not present in this data set)", w.Name(), w.ArchiveName())
						// Propagated so runWithProgress classifies it as
						// skipped; it swallows the error, preserving the
						// pre-task-203 "return nil" behaviour for callers.
						return err
					}
					return fmt.Errorf("%s open %s: %w", w.Name(), w.ArchiveName(), err)
				}
				defer cleanup()
				if gv := wzFile.GameVersion(); gv != 0 && gv != int(p.MajorVersion) {
					versionWarnOnce.Do(func() {
						l.Warnf("WZ data declares game version %d but ingest params are %s %d.%d — check the upload landed under the intended tenant/version", gv, p.Region, p.MajorVersion, p.MinorVersion)
					})
				}
				return w.Run(tctx, l, db, mc, wzFile, p)
			})
		}
		…
```

Leave the rest of `RunWorkers` (prerequisite loop, semaphore, errgroup) exactly as it is — both phases already call `runOne`.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go test -race ./data/... && go vet ./...
```

Expected: PASS. `TestSplitPrerequisites` still passes unchanged — it already pins the STRING-before-everything ordering the acceptance criteria call for.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/data/progress.go services/atlas-data/atlas.com/data/data/progress_test.go services/atlas-data/atlas.com/data/data/runwz.go
git commit -m "feat(atlas-data): route worker transitions through a ProgressSink

RunWorkers gains a variadic RunOption so a caller can observe per-worker
start/finish at the single runOne chokepoint covering both the sequential
prerequisite phase and the parallel fan-out. Default is a no-op sink, so
every existing call site and the compose path are unchanged."
```

---

### Task 7: The Redis-backed sink in `runtime/ingest`

**Files:**
- Create: `services/atlas-data/atlas.com/data/runtime/ingest/progress.go`
- Modify: `services/atlas-data/atlas.com/data/runtime/ingest/run.go`
- Modify: `services/atlas-data/atlas.com/data/runtime/ingest/heartbeat.go` (add the run-id env reader)
- Test: `services/atlas-data/atlas.com/data/runtime/ingest/progress_test.go`

**Interfaces:**
- Consumes: `data.ProgressSink`, `data.WithProgress`, `ingestrun.*`, `redis.Registry.UpdateWithTTL`, `workers.RegisteredNames`, `ingestJobSuffixFromEnv`.
- Produces (unexported, package `ingest`): `*redisSink` satisfying `data.ProgressSink`, with `newRedisSink(l, reg, suffix, runId)`, `Init(ctx, seed, roster, now)`, `Finish(ctx, runErr, now)`; and `ingestRunIdFromEnv() string`.

**Two hazards handled inside the sink, not at call sites:**
1. When a worker fails, the errgroup cancels its context — the enclosing `Finish(failed)` write would then run under a dead context and always fail, defeating FR-2.4 exactly when it matters most. Every write runs under `context.WithoutCancel(ctx)` wrapped in a 5-second `WithTimeout`.
2. The 5s ceiling bounds a wedged Redis to ~22×5s across a multi-minute run; the result is discarded after a warn log.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-data/atlas.com/data/runtime/ingest/progress_test.go`:

```go
package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"atlas-data/ingestrun"
)

func testSink(t *testing.T, runId string) (*redisSink, *miniredis.Miniredis, string) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	suffix := ingestrun.KeySuffix("shared", "GMS", 83, 1)
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return newRedisSink(l, ingestrun.NewRunRegistry(rdb), suffix, runId), mr, suffix
}

func readRecord(t *testing.T, s *redisSink) ingestrun.Record {
	t.Helper()
	rec, err := s.reg.Get(context.Background(), s.key)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	return rec
}

func seedFor(runId string, at time.Time) ingestrun.Record {
	return ingestrun.NewRecord(runId, "", "shared", "GMS", "83.1", "", at, []string{"STRING", "MAP"})
}

// The REST pod normally wrote the record already; Init must adopt it —
// preserving runId, jobName and startedAt — and only seed the roster.
func TestSinkInitAdoptsExistingRecord(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	created := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)

	existing := ingestrun.NewRecord("run-1", "job-1", "shared", "GMS", "83.1", "", created, nil)
	if err := s.reg.PutWithTTL(ctx, s.key, existing, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	s.Init(ctx, seedFor("run-1", time.Now().UTC()), []string{"STRING", "MAP"}, time.Now().UTC())

	got := readRecord(t, s)
	if !got.StartedAt.Equal(created) {
		t.Fatalf("startedAt = %v, want the REST pod's %v", got.StartedAt, created)
	}
	if got.JobName != "job-1" || got.RunId != "run-1" {
		t.Fatalf("identity not preserved: %+v", got)
	}
	if len(got.Workers) != 2 {
		t.Fatalf("roster size = %d, want 2", len(got.Workers))
	}
	if got.Phase != ingestrun.PhaseRunning {
		t.Fatalf("phase = %s, want running", got.Phase)
	}
}

// Redis may have lost the record between Job creation and pod start.
func TestSinkInitSeedsWhenAbsent(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Date(2026, 8, 8, 11, 0, 0, 0, time.UTC)

	s.Init(ctx, seedFor("run-1", now), []string{"STRING", "MAP"}, now)

	got := readRecord(t, s)
	if got.RunId != "run-1" || got.Phase != ingestrun.PhaseRunning || len(got.Workers) != 2 {
		t.Fatalf("seeded record wrong: %+v", got)
	}
}

func TestSinkWorkerTransitions(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING", "MAP"}, now)

	s.WorkerStarted(ctx, "STRING")
	if got := readRecord(t, s); got.Workers[0].State != ingestrun.WorkerRunning {
		t.Fatalf("STRING = %s, want running", got.Workers[0].State)
	}

	s.WorkerFinished(ctx, "STRING", nil, false)
	if got := readRecord(t, s); got.Workers[0].State != ingestrun.WorkerSucceeded {
		t.Fatalf("STRING = %s, want succeeded", got.Workers[0].State)
	}

	s.WorkerFinished(ctx, "MAP", nil, true)
	if got := readRecord(t, s); got.Workers[1].State != ingestrun.WorkerSkipped {
		t.Fatalf("MAP = %s, want skipped", got.Workers[1].State)
	}

	s.Finish(ctx, nil, time.Now().UTC())
	got := readRecord(t, s)
	if got.Phase != ingestrun.PhaseSucceeded {
		t.Fatalf("phase = %s, want succeeded", got.Phase)
	}
	if got.CompleteCount() != 2 {
		t.Fatalf("CompleteCount = %d, want 2 (skipped counts)", got.CompleteCount())
	}
}

func TestSinkFinishRecordsFailure(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING", "MAP"}, now)
	s.WorkerFinished(ctx, "STRING", errors.New("boom"), false)

	s.Finish(ctx, errors.New("STRING open String.wz: boom"), time.Now().UTC())

	got := readRecord(t, s)
	if got.Phase != ingestrun.PhaseFailed {
		t.Fatalf("phase = %s, want failed", got.Phase)
	}
	if got.Reason == "" {
		t.Fatal("failed run has no reason")
	}
	if got.Workers[0].State != ingestrun.WorkerFailed || got.Workers[0].Error != "boom" {
		t.Fatalf("worker failure not recorded: %+v", got.Workers[0])
	}
}

// When a worker fails the errgroup cancels the context; the terminal write
// must still land, or the run is stuck at `running` forever.
func TestSinkFinishSurvivesCancelledContext(t *testing.T) {
	s, _, _ := testSink(t, "run-1")
	now := time.Now().UTC()
	s.Init(context.Background(), seedFor("run-1", now), []string{"STRING"}, now)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.Finish(ctx, errors.New("boom"), time.Now().UTC())

	if got := readRecord(t, s); got.Phase != ingestrun.PhaseFailed {
		t.Fatalf("phase = %s, want failed (write must not depend on a live ctx)", got.Phase)
	}
}

// An operator re-triggering an ingest while this pod is alive replaces the
// record. A stale pod's writes must be dropped, not overwrite the new run.
func TestSinkDropsWritesForASupersededRun(t *testing.T) {
	s, _, _ := testSink(t, "run-OLD")
	ctx := context.Background()
	now := time.Now().UTC()

	fresh := ingestrun.NewRecord("run-NEW", "job-2", "shared", "GMS", "83.1", "", now, []string{"STRING"})
	if err := s.reg.PutWithTTL(ctx, s.key, fresh, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	s.WorkerFinished(ctx, "STRING", nil, false)
	s.Finish(ctx, nil, time.Now().UTC())

	got := readRecord(t, s)
	if got.RunId != "run-NEW" {
		t.Fatalf("runId = %s, want run-NEW", got.RunId)
	}
	if got.Phase != ingestrun.PhaseRunning {
		t.Fatalf("phase = %s: the stale pod stamped a terminal phase on the new run", got.Phase)
	}
	if got.Workers[0].State != ingestrun.WorkerPending {
		t.Fatalf("STRING = %s: the stale pod wrote into the new run", got.Workers[0].State)
	}
}

// FR-2.5: a Redis outage is warn-logged telemetry, never an ingest failure.
func TestSinkSurvivesRedisOutage(t *testing.T) {
	s, mr, _ := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING"}, now)
	mr.Close()

	// None of these return an error, and none may panic.
	s.WorkerStarted(ctx, "STRING")
	s.WorkerFinished(ctx, "STRING", nil, false)
	s.Finish(ctx, nil, time.Now().UTC())
}

// The TTL must be refreshed on every write — a record that lost its expiry
// would live forever (the reason UpdateWithTTL exists).
func TestSinkWritesKeepTheTTL(t *testing.T) {
	s, mr, suffix := testSink(t, "run-1")
	ctx := context.Background()
	now := time.Now().UTC()
	s.Init(ctx, seedFor("run-1", now), []string{"STRING"}, now)
	s.WorkerStarted(ctx, "STRING")

	key := "atlas:" + ingestrun.Namespace + ":" + suffix + ingestrun.RunKeySuffix
	if ttl := mr.TTL(key); ttl <= 0 {
		t.Fatalf("TTL = %v, want > 0", ttl)
	}
}
```

For the last test, prefer `redis.KeyPrefix()` over the literal `"atlas"` (import `redis "github.com/Chronicle20/atlas/libs/atlas-redis"`), as in Task 4.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./runtime/ingest/...
```

Expected: FAIL — `undefined: newRedisSink`, `undefined: redisSink`.

- [ ] **Step 3: Implement the sink**

Create `services/atlas-data/atlas.com/data/runtime/ingest/progress.go`:

```go
package ingest

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"atlas-data/ingestrun"

	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// progressWriteTimeout caps each progress write. Bounds a wedged Redis to
// ~22 × 5s across a multi-minute run: an ingest that would have succeeded must
// never fail because Redis did (PRD FR-2.5).
const progressWriteTimeout = 5 * time.Second

// redisSink is the Redis-backed data.ProgressSink. It also owns the run's
// Init/Finish bookends, so every write to the run record from the ingest pod
// goes through the same run-id guard and the same context discipline.
type redisSink struct {
	l     logrus.FieldLogger
	reg   *redis.Registry[string, ingestrun.Record]
	key   string
	runId string

	mu     sync.Mutex
	starts map[string]time.Time
}

func newRedisSink(l logrus.FieldLogger, reg *redis.Registry[string, ingestrun.Record], suffix, runId string) *redisSink {
	return &redisSink{
		l:      l,
		reg:    reg,
		key:    suffix + ingestrun.RunKeySuffix,
		runId:  runId,
		starts: make(map[string]time.Time),
	}
}

// writeCtx detaches from the caller's cancellation and bounds the write.
//
// When a worker fails, the errgroup cancels its context — and the terminal
// Finish(failed) write is exactly the write that must still land. Inheriting
// that cancellation would defeat FR-2.4 precisely when it matters most.
func (s *redisSink) writeCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), progressWriteTimeout)
}

// apply mutates the record under the run-id guard.
//
// The guard is evaluated inside the mutator, so it runs against the freshly
// read value on every optimistic-lock retry: a superseded pod can never win
// the race. An empty runId on either side means the guard cannot decide (a
// record written before run ids existed, or a pod with no INGEST_RUN_ID), and
// the write is allowed through.
func (s *redisSink) apply(ctx context.Context, what string, fn func(ingestrun.Record) ingestrun.Record) {
	if s == nil || s.reg == nil {
		return
	}
	wctx, cancel := s.writeCtx(ctx)
	defer cancel()

	stale := false
	_, err := s.reg.UpdateWithTTL(wctx, s.key, ingestrun.RecordTTL, func(rec ingestrun.Record) ingestrun.Record {
		if rec.RunId != "" && s.runId != "" && rec.RunId != s.runId {
			stale = true
			return rec
		}
		stale = false
		return fn(rec)
	})
	if stale {
		s.l.Debugf("ingest progress write dropped (%s): record belongs to run %s, this pod is %s", what, "<newer>", s.runId)
		return
	}
	if err != nil {
		s.l.WithError(err).Warnf("ingest progress write failed (%s, key=%s)", what, s.key)
	}
}

// Init adopts the record the REST pod wrote at Job creation — preserving its
// runId, jobName and startedAt (design Q2) — and only seeds the worker roster
// and confirms phase=running. seed is written only when no record exists, i.e.
// Redis lost it between Job creation and pod start.
func (s *redisSink) Init(ctx context.Context, seed ingestrun.Record, roster []string, now time.Time) {
	if s == nil || s.reg == nil {
		return
	}
	wctx, cancel := s.writeCtx(ctx)
	defer cancel()

	_, err := s.reg.UpdateWithTTL(wctx, s.key, ingestrun.RecordTTL, func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithRoster(roster).WithPhase(ingestrun.PhaseRunning, now, "")
	})
	if errors.Is(err, redis.ErrNotFound) {
		if perr := s.reg.PutWithTTL(wctx, s.key, seed, ingestrun.RecordTTL); perr != nil {
			s.l.WithError(perr).Warnf("ingest progress seed failed (key=%s)", s.key)
		}
		return
	}
	if err != nil {
		s.l.WithError(err).Warnf("ingest progress init failed (key=%s)", s.key)
	}
}

// Finish writes the terminal run phase. Called even when runErr aborted the
// errgroup, under a context that may already be cancelled — see writeCtx.
func (s *redisSink) Finish(ctx context.Context, runErr error, now time.Time) {
	phase := ingestrun.PhaseSucceeded
	reason := ""
	if runErr != nil {
		phase = ingestrun.PhaseFailed
		reason = runErr.Error()
	}
	s.l.Infof("ingest run %s", phase)
	s.apply(ctx, "finish", func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithPhase(phase, now, reason)
	})
}

func (s *redisSink) WorkerStarted(ctx context.Context, name string) {
	s.mu.Lock()
	s.starts[name] = time.Now()
	s.mu.Unlock()

	now := time.Now().UTC()
	s.l.Infof("ingest worker %s: running", name)
	s.apply(ctx, "worker-started:"+name, func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithWorkerRunning(name, now)
	})
}

func (s *redisSink) WorkerFinished(ctx context.Context, name string, err error, skipped bool) {
	s.mu.Lock()
	start, ok := s.starts[name]
	delete(s.starts, name)
	s.mu.Unlock()

	var dur time.Duration
	if ok {
		dur = time.Since(start)
	}

	state := ingestrun.WorkerSucceeded
	msg := ""
	switch {
	case skipped:
		state = ingestrun.WorkerSkipped
	case err != nil:
		state = ingestrun.WorkerFailed
		msg = err.Error()
	}

	// Logged at info so an operator debugging without the UI gets the same
	// information from pod logs (PRD §8 Observability).
	s.l.Infof("ingest worker %s: %s (duration=%s)", name, state, dur.Truncate(time.Millisecond))

	now := time.Now().UTC()
	s.apply(ctx, "worker-finished:"+name, func(rec ingestrun.Record) ingestrun.Record {
		return rec.WithWorkerTerminal(name, state, now, msg)
	})
}
```

Add to `runtime/ingest/heartbeat.go`:

```go
// ingestRunIdFromEnv returns the run identity JobCreator injected into the
// rendered Job. Empty in the compose / unit-test path, which disables the
// superseded-pod guard (there is no competing pod there).
func ingestRunIdFromEnv() string {
	return os.Getenv("INGEST_RUN_ID")
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go test -race ./runtime/ingest/... && go vet ./runtime/...
```

Expected: PASS.

- [ ] **Step 5: Wire the sink into `ingest.Run`**

Replace the heartbeat block and the final return in `services/atlas-data/atlas.com/data/runtime/ingest/run.go`:

```go
	// Refresh the Watchdog heartbeat every 30s while workers run, and publish
	// per-worker progress to the run record the REST pod initialised at Job
	// creation. Both are gated on the same env-derived suffix: absent SCOPE/
	// REGION means the compose / unit-test path, where neither signal has a
	// reader (PRD FR-2.6).
	var sink *redisSink
	if suffix := ingestJobSuffixFromEnv(); suffix != "" {
		rdb := redis.Connect(l)
		reg := ingestrun.NewJobRegistry(rdb)
		routine.Go(l, ctx, func(_ context.Context) { runHeartbeat(ctx, l, reg, suffix) })

		runId := ingestRunIdFromEnv()
		sink = newRedisSink(l, ingestrun.NewRunRegistry(rdb), suffix, runId)
		now := time.Now().UTC()
		sink.Init(ctx, ingestrun.NewRecord(
			runId, "", p.ScopeKey, p.Region,
			fmt.Sprintf("%d.%d", p.MajorVersion, p.MinorVersion),
			os.Getenv("TENANT_ID"), now, workers.RegisteredNames(),
		), workers.RegisteredNames(), now)
	} else {
		l.Info("ingest heartbeat and progress skipped: SCOPE/REGION/MAJOR_VERSION/MINOR_VERSION env not set (compose / test path)")
	}

	// Build the option list conditionally: passing a typed-nil *redisSink
	// through data.WithProgress would produce a non-nil interface whose
	// methods then run on a nil receiver.
	var opts []data.RunOption
	if sink != nil {
		opts = append(opts, data.WithProgress(sink))
	}
	err = data.RunWorkers(l, db, mc, opts...)(ctx, p)
	if sink != nil {
		sink.Finish(ctx, err, time.Now().UTC())
	}
	return err
```

Add `"time"` and `"atlas-data/ingestrun"` to the imports. Note the existing `err` variable from the MinIO block is reused — if the compiler objects, declare `runErr := data.RunWorkers(...)` and return that instead.

- [ ] **Step 6: Verify the wiring builds and the suite is green**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go test -race ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-data/atlas.com/data/runtime/ingest
git commit -m "feat(atlas-data): publish per-worker ingest progress to Redis

The ingest pod adopts the run record the REST pod wrote at Job creation,
flips each worker through running -> terminal at the runOne chokepoint, and
stamps the terminal run phase before exiting. Writes are guarded on
INGEST_RUN_ID so a superseded pod cannot stamp over a newer run, detached
from the caller's cancellation so the failure write still lands, and
best-effort so Redis can never fail an ingest."
```

---

### Task 8: Rewrite `GET /api/data/process`

Scope-resolve the caller, read the one record, cross-check a `running` record, and return JSON:API. `processCreate` switches to the same shared resolver in the same change, deleting its inline duplicate — the operator gate on `shared` cannot then drift between the two verbs.

**Files:**
- Create: `services/atlas-data/atlas.com/data/runtime/rest/ingestrun_model.go`
- Modify: `services/atlas-data/atlas.com/data/runtime/rest/resource.go`
- Modify: `services/atlas-data/atlas.com/data/main.go`
- Modify: `services/atlas-data/docs/rest.md`
- Test: `services/atlas-data/atlas.com/data/runtime/rest/resource_test.go`

**Interfaces:**
- Consumes: `wzinput.ResolveScope`, `wzinput.ErrSharedRequiresOperator`, `IngestRegistries`, `ingestrun.*`, `DefaultWatchdogTimeoutSecs`, `sanitizeLabel`, `jobFailed`.
- Produces:
  - `func InitResource(jc *JobCreator, regs *IngestRegistries) func(si jsonapi.ServerInformation) server.RouteInitializer`
  - `type IngestRunRestModel struct` with `GetName() == "ingestRun"`
  - `func toIngestRunRestModel(rec ingestrun.Record, phase ingestrun.Phase, id string) IngestRunRestModel`
  - `processStatusJob` is **deleted**.

- [ ] **Step 1: Write the failing tests**

Replace `services/atlas-data/atlas.com/data/runtime/rest/resource_test.go` wholesale:

```go
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"atlas-data/ingestrun"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/google/uuid"
)

type fakeServerInfo struct{}

func (fakeServerInfo) GetBaseURL() string { return "" }
func (fakeServerInfo) GetPrefix() string  { return "" }

const testTenantId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

func tenantCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.MustParse(testTenantId), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func newRegs(t *testing.T) (*IngestRegistries, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return NewIngestRegistries(rdb), mr
}

// ingestRunAttributes is the decoded JSON:API attributes object.
type ingestRunAttributes struct {
	RunId           string  `json:"runId"`
	JobName         string  `json:"jobName"`
	Scope           string  `json:"scope"`
	Region          string  `json:"region"`
	Version         string  `json:"version"`
	Tenant          string  `json:"tenant"`
	Phase           string  `json:"phase"`
	StartedAt       *string `json:"startedAt"`
	FinishedAt      *string `json:"finishedAt"`
	Reason          *string `json:"reason"`
	WorkersTotal    int     `json:"workersTotal"`
	WorkersComplete int     `json:"workersComplete"`
	Workers         []struct {
		Name       string  `json:"name"`
		State      string  `json:"state"`
		StartedAt  *string `json:"startedAt"`
		FinishedAt *string `json:"finishedAt"`
		Error      *string `json:"error"`
	} `json:"workers"`
}

func doStatus(t *testing.T, jc *JobCreator, regs *IngestRegistries, query string, operator bool) *httptest.ResponseRecorder {
	t.Helper()
	ctx := tenantCtx(t)
	d := server.NewHandlerDependency(logrus.New(), ctx)
	c := server.NewHandlerContext(fakeServerInfo{})
	h := processStatus(jc, regs)(&d, &c)

	req := httptest.NewRequest(http.MethodGet, "/api/data/process"+query, nil).WithContext(ctx)
	if operator {
		req.Header.Set("X-Atlas-Operator", "1")
	}
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func decodeRun(t *testing.T, rr *httptest.ResponseRecorder) (string, ingestRunAttributes) {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var doc struct {
		Data struct {
			Type       string              `json:"type"`
			Id         string              `json:"id"`
			Attributes ingestRunAttributes `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v; body=%s", err, rr.Body.String())
	}
	if doc.Data.Type != "ingestRun" {
		t.Fatalf("resource type = %q, want ingestRun", doc.Data.Type)
	}
	return doc.Data.Id, doc.Data.Attributes
}

func TestProcessStatusNoRecordReportsNone(t *testing.T) {
	regs, _ := newRegs(t)
	rr := doStatus(t, nil, regs, "", false)
	id, attrs := decodeRun(t, rr)
	if attrs.Phase != string(ingestrun.PhaseNone) {
		t.Fatalf("phase = %s, want none", attrs.Phase)
	}
	if attrs.WorkersTotal != 0 || len(attrs.Workers) != 0 {
		t.Fatalf("none record has workers: %+v", attrs)
	}
	want := "tenants/" + testTenantId + ":GMS:83.1"
	if id != want {
		t.Fatalf("id = %q, want %q", id, want)
	}
}

func TestProcessStatusReturnsOnlyTheCallersScope(t *testing.T) {
	ctx := context.Background()
	regs, _ := newRegs(t)

	mine := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)
	other := ingestrun.KeySuffix("tenants/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "GMS", 83, 1)
	put := func(suffix, runId string) {
		rec := ingestrun.NewRecord(runId, "job-"+runId, "x", "GMS", "83.1", "", time.Now().UTC(), []string{"STRING"})
		rec = rec.WithPhase(ingestrun.PhaseSucceeded, time.Now().UTC(), "")
		if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
			t.Fatal(err)
		}
	}
	put(mine, "run-mine")
	put(other, "run-other")

	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.RunId != "run-mine" {
		t.Fatalf("runId = %q: another tenant's run is visible", attrs.RunId)
	}
}

func TestProcessStatusSharedRequiresOperator(t *testing.T) {
	regs, _ := newRegs(t)
	if rr := doStatus(t, nil, regs, "?scope=shared", false); rr.Code != http.StatusForbidden {
		t.Fatalf("status %d, want 403", rr.Code)
	}
	rr := doStatus(t, nil, regs, "?scope=shared", true)
	id, _ := decodeRun(t, rr)
	if id != "shared:GMS:83.1" {
		t.Fatalf("id = %q, want shared:GMS:83.1", id)
	}
}

func TestProcessStatusBogusScopeIs400(t *testing.T) {
	regs, _ := newRegs(t)
	if rr := doStatus(t, nil, regs, "?scope=bogus", false); rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rr.Code)
	}
}

// A terminal record is returned as stored, with no Kubernetes call at all —
// this is what makes it readable after the Job is garbage-collected.
func TestProcessStatusTerminalRecordServedWithoutK8s(t *testing.T) {
	ctx := context.Background()
	regs, _ := newRegs(t)
	suffix := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)

	rec := ingestrun.NewRecord("run-1", "job-1", "tenants/"+testTenantId, "GMS", "83.1", testTenantId,
		time.Now().UTC().Add(-time.Hour), []string{"STRING", "MAP"})
	rec = rec.WithWorkerTerminal("STRING", ingestrun.WorkerSucceeded, time.Now().UTC(), "")
	rec = rec.WithWorkerTerminal("MAP", ingestrun.WorkerSkipped, time.Now().UTC(), "")
	rec = rec.WithPhase(ingestrun.PhaseSucceeded, time.Now().UTC(), "")
	if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseSucceeded) {
		t.Fatalf("phase = %s, want succeeded", attrs.Phase)
	}
	if attrs.WorkersTotal != 2 || attrs.WorkersComplete != 2 {
		t.Fatalf("workers %d/%d, want 2/2", attrs.WorkersComplete, attrs.WorkersTotal)
	}
	if attrs.FinishedAt == nil {
		t.Fatal("terminal record has no finishedAt")
	}
}

func TestProcessStatusStuckRecordSurfacesReason(t *testing.T) {
	ctx := context.Background()
	regs, _ := newRegs(t)
	suffix := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)

	rec := ingestrun.NewRecord("run-1", "job-1", "tenants/"+testTenantId, "GMS", "83.1", testTenantId,
		time.Now().UTC(), []string{"STRING", "MAP"})
	rec = rec.WithWorkerRunning("MAP", time.Now().UTC())
	rec = rec.WithPhase(ingestrun.PhaseStuck, time.Now().UTC(), "watchdog deleted the ingest Job after 7200s without a heartbeat")
	if err := regs.Run.PutWithTTL(ctx, suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}

	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseStuck) {
		t.Fatalf("phase = %s, want stuck", attrs.Phase)
	}
	if attrs.Reason == nil || *attrs.Reason == "" {
		t.Fatal("stuck record has no reason")
	}
	if attrs.Workers[1].State != string(ingestrun.WorkerRunning) {
		t.Fatalf("MAP = %s, want running (preserved under a terminal phase)", attrs.Workers[1].State)
	}
}

func seedRunning(t *testing.T, regs *IngestRegistries) string {
	t.Helper()
	suffix := ingestrun.KeySuffix("tenants/"+testTenantId, "GMS", 83, 1)
	rec := ingestrun.NewRecord("run-1", "job-1", "tenants/"+testTenantId, "GMS", "83.1", testTenantId,
		time.Now().UTC(), []string{"STRING", "MAP"})
	if err := regs.Run.PutWithTTL(context.Background(), suffix+ingestrun.RunKeySuffix, rec, ingestrun.RecordTTL); err != nil {
		t.Fatal(err)
	}
	return suffix
}

func TestProcessStatusRunningWithFreshHeartbeatStaysRunning(t *testing.T) {
	regs, _ := newRegs(t)
	suffix := seedRunning(t, regs)
	if err := regs.Job.PutWithTTL(context.Background(), suffix+ingestrun.HeartbeatKeySuffix,
		time.Now().UTC().Format(time.RFC3339), time.Hour); err != nil {
		t.Fatal(err)
	}
	// No JobCreator at all: a fresh heartbeat alone is enough.
	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseRunning) {
		t.Fatalf("phase = %s, want running", attrs.Phase)
	}
}

func TestProcessStatusRunningWithStaleHeartbeatAndNoJobIsUnknown(t *testing.T) {
	regs, _ := newRegs(t)
	suffix := seedRunning(t, regs)
	stale := time.Now().UTC().Add(-time.Duration(DefaultWatchdogTimeoutSecs+60) * time.Second)
	if err := regs.Job.PutWithTTL(context.Background(), suffix+ingestrun.HeartbeatKeySuffix,
		stale.Format(time.RFC3339), time.Hour); err != nil {
		t.Fatal(err)
	}
	jc := &JobCreator{K8s: fake.NewSimpleClientset(), Namespace: "ns"}
	_, attrs := decodeRun(t, doStatus(t, jc, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseUnknown) {
		t.Fatalf("phase = %s, want unknown", attrs.Phase)
	}
}

func TestProcessStatusRunningWithNoK8sClientIsUnknown(t *testing.T) {
	regs, _ := newRegs(t)
	_ = seedRunning(t, regs)
	// No heartbeat, no JobCreator: nothing corroborates the running record.
	_, attrs := decodeRun(t, doStatus(t, nil, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseUnknown) {
		t.Fatalf("phase = %s, want unknown", attrs.Phase)
	}
}

// The label selector must narrow to this triple server-side: a live Job for a
// DIFFERENT version must not keep this run looking alive.
func TestProcessStatusJobListIsNarrowedToTheTriple(t *testing.T) {
	regs, _ := newRegs(t)
	_ = seedRunning(t, regs)

	otherVersion := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j-other", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true",
			"scope":     sanitizeLabel("tenants/" + testTenantId),
			"region":    "GMS", "version": "87.1", "tenant": testTenantId,
		},
	}, Status: batchv1.JobStatus{Active: 1}}
	cs := fake.NewSimpleClientset(otherVersion)
	jc := &JobCreator{K8s: cs, Namespace: "ns"}

	_, attrs := decodeRun(t, doStatus(t, jc, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseUnknown) {
		t.Fatalf("phase = %s, want unknown (the live Job is a different version)", attrs.Phase)
	}
}

func TestProcessStatusRunningWithLiveJobStaysRunning(t *testing.T) {
	regs, _ := newRegs(t)
	_ = seedRunning(t, regs)

	live := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name: "j-live", Namespace: "ns",
		Labels: map[string]string{
			labelIngest: "true",
			"scope":     sanitizeLabel("tenants/" + testTenantId),
			"region":    "GMS", "version": "83.1", "tenant": testTenantId,
		},
	}, Status: batchv1.JobStatus{Active: 1}}
	cs := fake.NewSimpleClientset(live)
	jc := &JobCreator{K8s: cs, Namespace: "ns"}

	_, attrs := decodeRun(t, doStatus(t, jc, regs, "", false))
	if attrs.Phase != string(ingestrun.PhaseRunning) {
		t.Fatalf("phase = %s, want running", attrs.Phase)
	}
}

func TestProcessStatusWithNoRegistryReportsNone(t *testing.T) {
	rr := doStatus(t, nil, nil, "", false)
	_, attrs := decodeRun(t, rr)
	if attrs.Phase != string(ingestrun.PhaseNone) {
		t.Fatalf("phase = %s, want none", attrs.Phase)
	}
}
```

Keep any existing `TestProcessCreate*` tests in the file and add the operator/scope cases below if they are missing; the create handler's wire contract is unchanged.

`fake.NewSimpleClientset` applies label selectors server-side, so `TestProcessStatusJobListIsNarrowedToTheTriple` genuinely exercises the selector rather than client-side filtering.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-data/atlas.com/data && go test ./runtime/rest/...
```

Expected: FAIL — `processStatus` takes 1 arg, not 2; `ingestRun` type not produced.

- [ ] **Step 3: Write the REST model**

Create `services/atlas-data/atlas.com/data/runtime/rest/ingestrun_model.go`:

```go
package rest

import (
	"time"

	"github.com/jtumidanski/api2go/jsonapi"

	"atlas-data/ingestrun"
)

// IngestRunWorkerRestModel is one worker's slot in the response.
type IngestRunWorkerRestModel struct {
	Name       string  `json:"name"`
	State      string  `json:"state"`
	StartedAt  *string `json:"startedAt"`
	FinishedAt *string `json:"finishedAt"`
	Error      *string `json:"error"`
}

// IngestRunRestModel is the JSON:API projection of an ingestrun.Record.
//
// workersTotal / workersComplete are derived here rather than stored: a value
// computable from the worker list has no business being persisted twice and
// drifting.
type IngestRunRestModel struct {
	Id              string                     `json:"-"`
	RunId           string                     `json:"runId"`
	JobName         string                     `json:"jobName"`
	Scope           string                     `json:"scope"`
	Region          string                     `json:"region"`
	Version         string                     `json:"version"`
	Tenant          string                     `json:"tenant,omitempty"`
	Phase           string                     `json:"phase"`
	StartedAt       *string                    `json:"startedAt"`
	FinishedAt      *string                    `json:"finishedAt"`
	Reason          *string                    `json:"reason"`
	WorkersTotal    int                        `json:"workersTotal"`
	WorkersComplete int                        `json:"workersComplete"`
	Workers         []IngestRunWorkerRestModel `json:"workers"`
}

func (r IngestRunRestModel) GetName() string { return "ingestRun" }

func (r IngestRunRestModel) GetID() string { return r.Id }

func (r *IngestRunRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r IngestRunRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r IngestRunRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r IngestRunRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *IngestRunRestModel) SetToOneReferenceID(_, _ string) error { return nil }

func (r *IngestRunRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func (r *IngestRunRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func rfc3339Ptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// toIngestRunRestModel projects a stored record onto the wire. phase is passed
// separately because `unknown` is computed at read time and deliberately never
// stored — a record that later proves alive recovers on the next poll with no
// repair path.
func toIngestRunRestModel(rec ingestrun.Record, phase ingestrun.Phase, id string) IngestRunRestModel {
	ws := make([]IngestRunWorkerRestModel, 0, len(rec.Workers))
	for _, w := range rec.Workers {
		ws = append(ws, IngestRunWorkerRestModel{
			Name:       w.Name,
			State:      string(w.State),
			StartedAt:  rfc3339Ptr(w.StartedAt),
			FinishedAt: rfc3339Ptr(w.FinishedAt),
			Error:      strPtr(w.Error),
		})
	}
	m := IngestRunRestModel{
		Id:              id,
		RunId:           rec.RunId,
		JobName:         rec.JobName,
		Scope:           rec.Scope,
		Region:          rec.Region,
		Version:         rec.Version,
		Tenant:          rec.Tenant,
		Phase:           string(phase),
		FinishedAt:      rfc3339Ptr(rec.FinishedAt),
		Reason:          strPtr(rec.Reason),
		WorkersTotal:    len(rec.Workers),
		WorkersComplete: rec.CompleteCount(),
		Workers:         ws,
	}
	if !rec.StartedAt.IsZero() {
		m.StartedAt = rfc3339Ptr(&rec.StartedAt)
	}
	return m
}
```

- [ ] **Step 4: Rewrite the handlers**

In `services/atlas-data/atlas.com/data/runtime/rest/resource.go`:

```go
// InitResource installs POST/GET /data/process.
//
// jc and regs are independently nil-able: no jc makes the create handler
// respond 503 and disables the status handler's live-Job cross-check; no regs
// degrades the status handler to phase "none" (PRD FR-4.5).
func InitResource(jc *JobCreator, regs *IngestRegistries) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/data").Subrouter()
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_create", processCreate(jc))).Methods(http.MethodPost)
			r.HandleFunc("/process", rest.RegisterHandler(l)(si)("process_status", processStatus(jc, regs))).Methods(http.MethodGet)
		}
	}
}

// writeScopeError maps a wzinput.ResolveScope failure onto its HTTP status.
// Shared by both verbs so the operator gate on scope=shared cannot drift.
func writeScopeError(w http.ResponseWriter, err error) {
	if errors.Is(err, wzinput.ErrSharedRequiresOperator) {
		http.Error(w, "operator required", http.StatusForbidden)
		return
	}
	http.Error(w, "invalid scope", http.StatusBadRequest)
}
```

`processCreate` loses its inline switch:

```go
			t := tenant.MustFromContext(d.Context())
			sc, serr := wzinput.ResolveScope(r, t)
			if serr != nil {
				writeScopeError(w, serr)
				return
			}
			name, err := jc.Create(r.Context(), sc.Key, t.Region(), t.MajorVersion(), t.MinorVersion(), t.Id().String(), r.Header.Get("traceparent"))
```

and the response body's `"scope"` field becomes `sc.Key`.

Delete `processStatusJob` and replace `processStatus`:

```go
// processStatus returns the ingest run for the caller's resolved scope.
//
// Redis is the system of record; Kubernetes is a corroborating second opinion
// used only to demote a stale `running` to `unknown`. That direction is
// deliberate: the Job object is the thing that disappears
// (ttlSecondsAfterFinished, Watchdog deletion), so it cannot be the source of
// truth for a feature whose whole point is surviving its disappearance.
func processStatus(jc *JobCreator, regs *IngestRegistries) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			sc, serr := wzinput.ResolveScope(r, t)
			if serr != nil {
				writeScopeError(w, serr)
				return
			}
			suffix := ingestrun.KeySuffix(sc.Key, t.Region(), t.MajorVersion(), t.MinorVersion())
			id := suffix

			var rec ingestrun.Record
			phase := ingestrun.PhaseNone
			if regs != nil && regs.Run != nil {
				stored, err := regs.Run.Get(r.Context(), suffix+ingestrun.RunKeySuffix)
				switch {
				case err == nil:
					rec, phase = stored, stored.Phase
				case errors.Is(err, redis.ErrNotFound):
					// "No ingest has been run" is a valid, actionable answer.
				default:
					d.Logger().WithError(err).Errorf("Unable to read ingest run record %s.", suffix)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
			}

			if phase == ingestrun.PhaseRunning {
				phase = corroborateRunning(r.Context(), jc, regs, suffix, sc.Key, t.Region(), t.MajorVersion(), t.MinorVersion())
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			res := toIngestRunRestModel(rec, phase, id)
			server.MarshalResponse[IngestRunRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

// corroborateRunning decides whether a stored `running` phase is still
// believable. It never rewrites the stored record: `unknown` is computed at
// read time, so a record that later proves alive (a slow Job list, a restarted
// pod) recovers on the next poll with no repair path.
func corroborateRunning(ctx context.Context, jc *JobCreator, regs *IngestRegistries, suffix, scope, region string, major, minor uint16) ingestrun.Phase {
	// A heartbeat inside the Watchdog's staleness window means the pod is
	// alive; believe it regardless of what Kubernetes says. This covers the
	// window between Job creation and pod scheduling, and any Job-list hiccup.
	if regs != nil && regs.Job != nil {
		if ts, err := regs.Job.Get(ctx, suffix+ingestrun.HeartbeatKeySuffix); err == nil && ts != "" {
			if hb, perr := time.Parse(time.RFC3339, ts); perr == nil {
				if time.Since(hb) < time.Duration(DefaultWatchdogTimeoutSecs)*time.Second {
					return ingestrun.PhaseRunning
				}
			}
		}
	}
	if jc == nil || jc.K8s == nil {
		return ingestrun.PhaseUnknown
	}
	// Narrowed server-side to this triple rather than listing the namespace and
	// filtering client-side. Note the selector uses the SANITIZED scope,
	// matching renderJob's label; the raw scope only ever appears in Redis keys.
	selector := fmt.Sprintf("%s=true,scope=%s,region=%s,version=%d.%d",
		labelIngest, sanitizeLabel(scope), region, major, minor)
	list, err := jc.K8s.BatchV1().Jobs(jc.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		// A failed list is not evidence of absence — keep the stored phase.
		return ingestrun.PhaseRunning
	}
	for i := range list.Items {
		j := &list.Items[i]
		if j.Status.Succeeded == 0 && !jobFailed(j) {
			return ingestrun.PhaseRunning
		}
	}
	return ingestrun.PhaseUnknown
}
```

Update the import block: add `"context"`, `"errors"`, `"atlas-data/ingestrun"`, `"atlas-data/wzinput"`, `redis "github.com/Chronicle20/atlas/libs/atlas-redis"`; drop `"encoding/json"` only if `processCreate`'s response encoder no longer needs it (it does — keep it).

In `main.go`:

```go
	var jc *restruntime.JobCreator
	var ingestRegs *restruntime.IngestRegistries
	if os.Getenv("MODE") == "rest" {
		rdb := redis.Connect(l)
		// Built from the client directly, not from the JobCreator: the status
		// handler must still serve the stored run record when the in-cluster
		// config is unavailable and jc is therefore nil (FR-4.5).
		ingestRegs = restruntime.NewIngestRegistries(rdb)
		var jcErr error
		jc, jcErr = restruntime.NewJobCreatorInClusterWithRedis(rdb)
		…
```

and the route line becomes:

```go
		AddRouteInitializer(restruntime.InitResource(jc, ingestRegs)(GetServer())).
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-data/atlas.com/data && go build ./... && go test -race ./... && go vet ./...
```

Expected: PASS.

- [ ] **Step 6: Update `services/atlas-data/docs/rest.md`**

Replace the `### GET /api/data/process` section (currently "Lists active/recent ingest Jobs", the raw-JSON shape, and the 503 row) with:

````markdown
### GET /api/data/process

Returns the most recent ingest run for one `(scope, region, version)` triple.
Region and version come from the standard tenant headers.

#### Query Parameters

- `scope` (optional): `""` or `"tenant"` (default) returns the caller's own
  tenant's run; `"shared"` returns the version-scoped canonical dataset's run
  and requires `X-Atlas-Operator: 1`.

#### Response

- 200: a JSON:API document, resource type `ingestRun`, id
  `<scope>:<region>:<major>.<minor>`.

Attributes: `runId`, `jobName`, `scope`, `region`, `version`, `tenant`,
`phase`, `startedAt`, `finishedAt`, `reason`, `workersTotal`,
`workersComplete`, and `workers` — one entry per registered ingest worker with
`name`, `state`, `startedAt`, `finishedAt`, `error`.

`phase` is one of:

| Phase | Meaning |
|---|---|
| `none` | No run record exists for the triple. Not an error. |
| `running` | A run is in flight, corroborated by a fresh heartbeat or a live Job. |
| `succeeded` | Every worker finished without error. |
| `failed` | The ingest process returned an error; `reason` carries it. |
| `stuck` | The Watchdog deleted the Job for heartbeat staleness; `reason` names the timeout. |
| `unknown` | The record says `running` but neither a fresh heartbeat nor a live Job corroborates it. Computed at read time, never stored. |

Worker `state` is one of `pending`, `running`, `succeeded`, `failed`, or
`skipped`. `skipped` is a category genuinely absent from a monolithic `Data.wz`
(v12 has no Quest); it does not stop a run reaching `succeeded`. A worker left
in `running` under a terminal run phase was the one in flight when the run
ended.

Terminal phases are served straight from Redis with no Kubernetes call, so a
run stays readable after its Job is garbage-collected or watchdog-deleted. Run
records carry a 7-day TTL refreshed on every write; an evicted record reports
`none`.

#### Errors

- 400 Bad Request: invalid `scope` value
- 403 Forbidden: `scope=shared` without `X-Atlas-Operator: 1`

An unavailable Kubernetes client no longer yields 503 — the stored record is
still served, with only the live-Job cross-check degraded.
````

Add one sentence to the `POST /api/data/process` section: "The run record for the target triple is initialised (or reset) at Job creation, so a run that dies before the ingest pod's first write is still represented."

- [ ] **Step 7: Commit**

```bash
git add services/atlas-data/atlas.com/data/runtime/rest services/atlas-data/atlas.com/data/main.go services/atlas-data/docs/rest.md
git commit -m "feat(atlas-data): scope and rewrite GET /api/data/process

The endpoint returned every ingest Job in the namespace with its tenant
label. It now resolves the caller's scope through the same shared resolver
POST uses, returns exactly that triple's run record as a JSON:API ingestRun
document, and cross-checks a running record against the heartbeat and a
label-narrowed Job list. An absent record is 200/none, not 404; an absent
Kubernetes client no longer 503s."
```

---

### Task 9: UI types, fetchers, and hooks

**Files:**
- Modify: `services/atlas-ui/src/services/api/seed.service.ts`
- Modify: `services/atlas-ui/src/lib/hooks/api/useSeed.ts`
- Modify: `services/atlas-ui/src/lib/hooks/api/useCanonicalData.ts`

**Interfaces:**
- Consumes: `fetchJsonApi`, `tenantHeaders`, `canonicalHeaders`, `CanonicalSelection`.
- Produces (exported from `seed.service.ts`): `type IngestPhase`, `type IngestWorkerState`, `interface IngestRunWorker`, `interface IngestRun`; `seedService.getIngestRun(tenant)`, `seedService.getCanonicalIngestRun(sel)`. From the hook modules: `useIngestRun(): UseQueryResult<IngestRun, Error>` and `useCanonicalIngestRun(sel: CanonicalSelection | null): UseQueryResult<IngestRun, Error>`.

- [ ] **Step 1: Add the types and fetchers**

In `services/atlas-ui/src/services/api/seed.service.ts`, next to the other exported status interfaces:

```ts
export type IngestPhase =
  | "none"
  | "running"
  | "succeeded"
  | "failed"
  | "stuck"
  | "unknown";

export type IngestWorkerState =
  | "pending"
  | "running"
  | "succeeded"
  | "failed"
  | "skipped";

export interface IngestRunWorker {
  name: string;
  state: IngestWorkerState;
  startedAt: string | null;
  finishedAt: string | null;
  error: string | null;
}

export interface IngestRun {
  runId: string;
  jobName: string;
  scope: string;
  region: string;
  version: string;
  tenant?: string;
  phase: IngestPhase;
  startedAt: string | null;
  finishedAt: string | null;
  reason: string | null;
  workersTotal: number;
  workersComplete: number;
  workers: IngestRunWorker[];
}
```

Add the two fetchers to the `seedService` object, beside `getDataStatus` and `getCanonicalDataStatus`:

```ts
  async getIngestRun(tenant: Tenant): Promise<IngestRun> {
    return fetchJsonApi<IngestRun>(
      "/api/data/process?scope=tenant",
      tenantHeaders(tenant),
    );
  },
```

```ts
  // canonicalHeaders already bakes in X-Atlas-Operator: 1, which the shared
  // scope requires — one construction path, no drift.
  async getCanonicalIngestRun(sel: CanonicalSelection): Promise<IngestRun> {
    return fetchJsonApi<IngestRun>(
      "/api/data/process?scope=shared",
      canonicalHeaders(sel),
    );
  },
```

Match the surrounding declaration style exactly (the file uses class-or-object methods consistently — copy whichever form `getDataStatus` uses).

- [ ] **Step 2: Add the hooks**

In `services/atlas-ui/src/lib/hooks/api/useSeed.ts`, beside `useDataStatus` (and add an `ingestRunKey` next to the other key builders):

```ts
const ingestRunKey = (tenantId: string) => ["ingestRun", tenantId] as const;

export function useIngestRun(): UseQueryResult<IngestRun, Error> {
  const { activeTenant } = useTenant();
  return useQuery({
    queryKey: activeTenant
      ? ingestRunKey(activeTenant.id)
      : ["ingestRun", "none"],
    queryFn: () => seedService.getIngestRun(activeTenant!),
    enabled: !!activeTenant,
    staleTime: 0,
    refetchInterval: 5000,
    // No retry, and the panel renders from query.isError rather than raising a
    // toast: an unreachable endpoint gives a steady "progress unavailable"
    // panel on a 5s cadence that never escalates.
    retry: false,
  });
}
```

Import the `IngestRun` type from `@/services/api/seed.service`.

In `services/atlas-ui/src/lib/hooks/api/useCanonicalData.ts`, beside `useCanonicalDataStatus`:

```ts
const canonicalIngestRunKey = (sel: CanonicalSelection) =>
  [
    "canonical",
    "ingestRun",
    sel.region,
    sel.majorVersion,
    sel.minorVersion,
  ] as const;

export function useCanonicalIngestRun(
  sel: CanonicalSelection | null,
): UseQueryResult<IngestRun, Error> {
  return useQuery({
    queryKey: sel
      ? canonicalIngestRunKey(sel)
      : ["canonical", "ingestRun", "none"],
    queryFn: () => seedService.getCanonicalIngestRun(sel!),
    enabled: !!sel,
    staleTime: 0,
    refetchInterval: 5000,
    retry: false,
  });
}
```

Add `type IngestRun` to that file's `@/services/api/seed.service` import.

- [ ] **Step 3: Type-check**

```bash
cd services/atlas-ui && . "$NVM_DIR/nvm.sh" && nvm use 22 && npm run build
```

Expected: build clean (it type-checks tests too).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/src/services/api/seed.service.ts services/atlas-ui/src/lib/hooks/api/useSeed.ts services/atlas-ui/src/lib/hooks/api/useCanonicalData.ts
git commit -m "feat(atlas-ui): add ingest-run types, fetchers, and polling hooks"
```

---

### Task 10: The `IngestProgressPanel` component

One shared presentational component owning no fetching, mounted by both pages with a different hook, so the two surfaces cannot drift.

**Files:**
- Create: `services/atlas-ui/src/components/features/setup/ingest-progress.ts`
- Create: `services/atlas-ui/src/components/features/setup/IngestProgressPanel.tsx`
- Test: `services/atlas-ui/src/components/features/setup/__tests__/ingest-progress.test.ts`
- Test: `services/atlas-ui/src/components/features/setup/__tests__/IngestProgressPanel.test.tsx`

**Interfaces:**
- Consumes: `IngestRun`, `IngestPhase`, `IngestRunWorker` from `@/services/api/seed.service`.
- Produces:
  - `ingest-progress.ts`: `const INGEST_BLOCKING_PHASES: readonly IngestPhase[]`, `function ingestPublishBlockReason(run: IngestRun | undefined, isError: boolean): string | null`, `function formatDuration(ms: number): string`, `function ingestElapsedMs(run: IngestRun, now: number): number | null`.
  - `IngestProgressPanel.tsx`: `function IngestProgressPanel(props: { run?: IngestRun; isError: boolean })`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-ui/src/components/features/setup/__tests__/ingest-progress.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import {
  INGEST_BLOCKING_PHASES,
  formatDuration,
  ingestElapsedMs,
  ingestPublishBlockReason,
} from "@/components/features/setup/ingest-progress";
import type { IngestPhase, IngestRun } from "@/services/api/seed.service";

function run(phase: IngestPhase, over: Partial<IngestRun> = {}): IngestRun {
  return {
    runId: "r1",
    jobName: "j1",
    scope: "shared",
    region: "GMS",
    version: "83.1",
    phase,
    startedAt: "2026-08-08T10:00:00Z",
    finishedAt: null,
    reason: null,
    workersTotal: 11,
    workersComplete: 0,
    workers: [],
    ...over,
  };
}

describe("ingestPublishBlockReason", () => {
  // FR-5.6: `none` is the pre-existing state for every scope never ingested
  // through this mechanism, and an evicted record degrades to it — neither may
  // wedge the publish control.
  it.each<IngestPhase>(["none", "succeeded"])("does not block on %s", (p) => {
    expect(ingestPublishBlockReason(run(p), false)).toBeNull();
  });

  it.each<IngestPhase>(["running", "stuck", "failed", "unknown"])(
    "blocks on %s with an explanation",
    (p) => {
      const reason = ingestPublishBlockReason(run(p), false);
      expect(reason).toBeTruthy();
      expect(reason).toContain(p);
    },
  );

  it("does not block when the run is unknown to the UI", () => {
    expect(ingestPublishBlockReason(undefined, false)).toBeNull();
    expect(ingestPublishBlockReason(undefined, true)).toBeNull();
  });

  it("surfaces the recorded reason when there is one", () => {
    const r = run("stuck", { reason: "watchdog deleted the ingest Job" });
    expect(ingestPublishBlockReason(r, false)).toContain(
      "watchdog deleted the ingest Job",
    );
  });

  it("exports the blocking set", () => {
    expect([...INGEST_BLOCKING_PHASES].sort()).toEqual([
      "failed",
      "running",
      "stuck",
      "unknown",
    ]);
  });
});

describe("ingestElapsedMs", () => {
  it("measures to now while in flight", () => {
    const now = Date.parse("2026-08-08T10:05:00Z");
    expect(ingestElapsedMs(run("running"), now)).toBe(5 * 60 * 1000);
  });

  it("measures to finishedAt once terminal", () => {
    const r = run("succeeded", { finishedAt: "2026-08-08T10:02:00Z" });
    const now = Date.parse("2026-08-08T11:00:00Z");
    expect(ingestElapsedMs(r, now)).toBe(2 * 60 * 1000);
  });

  it("is null without a start", () => {
    expect(ingestElapsedMs(run("none", { startedAt: null }), Date.now())).toBeNull();
  });
});

describe("formatDuration", () => {
  it("formats sub-minute, minute, and hour spans", () => {
    expect(formatDuration(4_000)).toBe("4s");
    expect(formatDuration(125_000)).toBe("2m 5s");
    expect(formatDuration(3_725_000)).toBe("1h 2m");
  });
});
```

Create `services/atlas-ui/src/components/features/setup/__tests__/IngestProgressPanel.test.tsx`:

```tsx
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { IngestProgressPanel } from "@/components/features/setup/IngestProgressPanel";
import type { IngestPhase, IngestRun } from "@/services/api/seed.service";

function run(phase: IngestPhase, over: Partial<IngestRun> = {}): IngestRun {
  return {
    runId: "r1",
    jobName: "ingest-shared-gms-83-1-x7f2qa",
    scope: "shared",
    region: "GMS",
    version: "83.1",
    phase,
    startedAt: "2026-08-08T10:00:00Z",
    finishedAt: null,
    reason: null,
    workersTotal: 2,
    workersComplete: 1,
    workers: [
      {
        name: "STRING",
        state: "succeeded",
        startedAt: "2026-08-08T10:00:01Z",
        finishedAt: "2026-08-08T10:02:41Z",
        error: null,
      },
      {
        name: "MAP",
        state: "running",
        startedAt: "2026-08-08T10:02:42Z",
        finishedAt: null,
        error: null,
      },
    ],
    ...over,
  };
}

describe("IngestProgressPanel", () => {
  it("shows the phase and the completed-of-total count", () => {
    render(<IngestProgressPanel run={run("running")} isError={false} />);
    expect(screen.getByText(/running/i)).toBeInTheDocument();
    expect(screen.getByText(/1\s*\/\s*2/)).toBeInTheDocument();
  });

  it("lists every worker with its state", () => {
    render(<IngestProgressPanel run={run("running")} isError={false} />);
    expect(screen.getByText("STRING")).toBeInTheDocument();
    expect(screen.getByText("MAP")).toBeInTheDocument();
    expect(screen.getByText("succeeded")).toBeInTheDocument();
  });

  it("surfaces the reason on a stuck run", () => {
    const r = run("stuck", {
      reason: "watchdog deleted the ingest Job after 7200s without a heartbeat",
    });
    render(<IngestProgressPanel run={r} isError={false} />);
    expect(screen.getByText(/without a heartbeat/)).toBeInTheDocument();
  });

  it("surfaces a worker's error text on a failed run", () => {
    const r = run("failed", {
      reason: "MAP open Map.wz: boom",
      workers: [
        {
          name: "MAP",
          state: "failed",
          startedAt: "2026-08-08T10:00:01Z",
          finishedAt: "2026-08-08T10:00:09Z",
          error: "boom",
        },
      ],
    });
    render(<IngestProgressPanel run={r} isError={false} />);
    expect(screen.getByText(/boom/)).toBeInTheDocument();
  });

  it("renders a worker still running under a terminal phase as interrupted", () => {
    render(<IngestProgressPanel run={run("stuck")} isError={false} />);
    expect(screen.getByText(/interrupted/i)).toBeInTheDocument();
  });

  it("says nothing has been run for phase none", () => {
    const r = run("none", { workers: [], workersTotal: 0, workersComplete: 0 });
    render(<IngestProgressPanel run={r} isError={false} />);
    expect(screen.getByText(/no ingest has been run/i)).toBeInTheDocument();
  });

  it("degrades to progress unavailable on error", () => {
    render(<IngestProgressPanel isError={true} />);
    expect(screen.getByText(/progress unavailable/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-ui && . "$NVM_DIR/nvm.sh" && nvm use 22 && npx vitest run src/components/features/setup
```

Expected: FAIL — cannot resolve `@/components/features/setup/ingest-progress` or `IngestProgressPanel`.

- [ ] **Step 3: Implement the helpers**

Create `services/atlas-ui/src/components/features/setup/ingest-progress.ts`:

```ts
import type { IngestPhase, IngestRun } from "@/services/api/seed.service";

/**
 * Phases that must block publishing a canonical baseline. `none` and
 * `succeeded` are deliberately absent: `none` is the pre-existing state for
 * every scope never ingested through this mechanism, and it is also what an
 * evicted Redis record degrades to — neither may wedge the control.
 */
export const INGEST_BLOCKING_PHASES: readonly IngestPhase[] = [
  "running",
  "stuck",
  "failed",
  "unknown",
];

const PHASE_EXPLANATION: Record<string, string> = {
  running: "an ingest is currently running for this region/version",
  stuck: "the last ingest was terminated by the watchdog",
  failed: "the last ingest failed",
  unknown: "the last ingest's outcome could not be determined",
};

/**
 * Returns the reason publishing is blocked, or null when it is allowed.
 * An unknown run (no data yet, or the endpoint is unreachable) never blocks —
 * progress is telemetry, and losing it must not take the control with it.
 */
export function ingestPublishBlockReason(
  run: IngestRun | undefined,
  _isError: boolean,
): string | null {
  if (!run) return null;
  if (!INGEST_BLOCKING_PHASES.includes(run.phase)) return null;
  const base = `Cannot publish: ${PHASE_EXPLANATION[run.phase] ?? run.phase} (${run.phase}).`;
  return run.reason ? `${base} ${run.reason}` : base;
}

/** Elapsed time for an in-flight run, or total duration once terminal. */
export function ingestElapsedMs(run: IngestRun, now: number): number | null {
  if (!run.startedAt) return null;
  const start = Date.parse(run.startedAt);
  if (Number.isNaN(start)) return null;
  const end = run.finishedAt ? Date.parse(run.finishedAt) : now;
  if (Number.isNaN(end)) return null;
  return Math.max(0, end - start);
}

export function formatDuration(ms: number): string {
  const total = Math.floor(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}
```

Note `ingestPublishBlockReason`'s test asserts the returned string contains the phase name — the `(${run.phase})` suffix satisfies that for every blocking phase.

- [ ] **Step 4: Implement the panel**

Create `services/atlas-ui/src/components/features/setup/IngestProgressPanel.tsx`:

```tsx
import type { IngestRun, IngestRunWorker } from "@/services/api/seed.service";
import {
  formatDuration,
  ingestElapsedMs,
} from "@/components/features/setup/ingest-progress";

interface IngestProgressPanelProps {
  run?: IngestRun;
  isError: boolean;
}

const TERMINAL_PHASES = ["succeeded", "failed", "stuck"];

function phaseClass(phase: string): string {
  switch (phase) {
    case "failed":
    case "stuck":
      return "text-destructive font-medium";
    case "unknown":
      return "text-amber-600 dark:text-amber-500 font-medium";
    case "succeeded":
      return "text-emerald-600 dark:text-emerald-500 font-medium";
    default:
      return "text-muted-foreground font-medium";
  }
}

function workerDuration(w: IngestRunWorker, now: number): string {
  if (!w.startedAt) return "—";
  const start = Date.parse(w.startedAt);
  if (Number.isNaN(start)) return "—";
  const end = w.finishedAt ? Date.parse(w.finishedAt) : now;
  return formatDuration(Math.max(0, end - start));
}

/**
 * Presentational only — it owns no fetching. Both the Setup page (tenant
 * scope) and the Baselines page (shared scope) mount this same component with
 * a different hook, so the two surfaces cannot drift.
 */
export function IngestProgressPanel({ run, isError }: IngestProgressPanelProps) {
  if (isError || !run) {
    return (
      <div className="border-b last:border-0 py-3">
        <p className="text-sm text-muted-foreground">
          Ingest progress unavailable.
        </p>
      </div>
    );
  }

  if (run.phase === "none") {
    return (
      <div className="border-b last:border-0 py-3">
        <p className="text-sm text-muted-foreground">
          No ingest has been run for this region/version yet.
        </p>
      </div>
    );
  }

  const now = Date.now();
  const elapsed = ingestElapsedMs(run, now);
  const terminal = TERMINAL_PHASES.includes(run.phase);

  return (
    <div className="border-b last:border-0 py-3" aria-live="polite">
      <div className="flex items-center justify-between gap-4">
        <p className="text-sm">
          Ingest <span className={phaseClass(run.phase)}>{run.phase}</span>
        </p>
        <p className="text-xs text-muted-foreground">
          {run.workersComplete} / {run.workersTotal} workers
          {elapsed !== null
            ? ` · ${terminal ? "took" : "elapsed"} ${formatDuration(elapsed)}`
            : ""}
        </p>
      </div>

      {run.reason ? (
        <p className="mt-1 text-xs text-destructive">{run.reason}</p>
      ) : null}

      <ul className="mt-2 grid gap-1 sm:grid-cols-2">
        {run.workers.map((w) => (
          <li
            key={w.name}
            className="flex items-baseline justify-between gap-2 text-xs"
          >
            <span className="font-mono">{w.name}</span>
            <span className="text-muted-foreground">
              {/* A worker still `running` under a terminal run phase was the
                  one in flight when the run ended — a derived presentation
                  concern, requiring no extra stored state. */}
              {terminal && w.state === "running" ? "interrupted" : w.state}
              {w.error ? ` — ${w.error}` : ""} · {workerDuration(w, now)}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-ui && . "$NVM_DIR/nvm.sh" && nvm use 22 && npx vitest run src/components/features/setup && npm run build
```

Expected: PASS, build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/setup
git commit -m "feat(atlas-ui): add the shared ingest progress panel

Presentational component plus the publish-gate helpers, covered for every
phase including the interrupted-worker rendering and the progress-unavailable
degradation."
```

---

### Task 11: Mount the panel on both pages and gate the publish control

**Files:**
- Modify: `services/atlas-ui/src/pages/SetupPage.tsx`
- Modify: `services/atlas-ui/src/pages/BaselinesPage.tsx`
- Test: `services/atlas-ui/src/pages/__tests__/BaselinesPage.test.tsx`

**Interfaces:**
- Consumes: `useIngestRun`, `useCanonicalIngestRun`, `IngestProgressPanel`, `ingestPublishBlockReason`.
- Produces: no new exports.

- [ ] **Step 1: Write the failing tests**

Read `services/atlas-ui/src/pages/__tests__/BaselinesPage.test.tsx` first and follow its existing mocking style. Append cases equivalent to:

```tsx
describe("BaselinesPage publish gate", () => {
  it("disables Publish Baseline while a shared ingest is running", async () => {
    // Mock useCanonicalIngestRun to return { data: run("running"), isError: false }
    // alongside the existing useCanonicalDataStatus mock (documentCount > 0),
    // then render and select a region/version exactly as the file's existing
    // publish tests do.
    const button = await screen.findByRole("button", { name: /publish baseline/i });
    expect(button).toBeDisabled();
    expect(screen.getByText(/cannot publish/i)).toBeInTheDocument();
  });

  it("enables Publish Baseline when the shared ingest succeeded", async () => {
    // Same setup, phase "succeeded".
    const button = await screen.findByRole("button", { name: /publish baseline/i });
    expect(button).toBeEnabled();
  });

  it("enables Publish Baseline when no ingest has been recorded", async () => {
    // Same setup, phase "none" — FR-5.6: must not regress today's behaviour.
    const button = await screen.findByRole("button", { name: /publish baseline/i });
    expect(button).toBeEnabled();
  });
});
```

The exhaustive six-phase truth table lives in `ingest-progress.test.ts` (Task 10); these three cases prove the page is actually wired to it.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/atlas-ui && . "$NVM_DIR/nvm.sh" && nvm use 22 && npx vitest run src/pages/__tests__/BaselinesPage.test.tsx
```

Expected: FAIL — the button is enabled while `running`, and no "Cannot publish" text exists.

- [ ] **Step 3: Wire `BaselinesPage`**

Add the imports:

```tsx
import { IngestProgressPanel } from "@/components/features/setup/IngestProgressPanel";
import { ingestPublishBlockReason } from "@/components/features/setup/ingest-progress";
```

and `useCanonicalIngestRun` to the existing `@/lib/hooks/api/useCanonicalData` import.

Next to the other queries in the component body:

```tsx
  const ingestRun = useCanonicalIngestRun(sel);
```

Extend the publish gate (replacing the existing `publishDisabled` line):

```tsx
  const ingestBlockReason = ingestPublishBlockReason(
    ingestRun.data,
    ingestRun.isError,
  );
  const publishDisabled =
    !sel ||
    !docData ||
    docData.documentCount === 0 ||
    publish.isPending ||
    ingestBlockReason !== null;
```

Mount the panel as its own row directly beneath the "Process Data" `SetupRow`:

```tsx
          <IngestProgressPanel
            run={ingestRun.data}
            isError={ingestRun.isError}
          />
```

Give the "Publish Baseline" `SetupRow` the explanation through its existing `warning` slot:

```tsx
            warning={
              ingestBlockReason ? (
                <p className="text-xs text-destructive">{ingestBlockReason}</p>
              ) : undefined
            }
```

- [ ] **Step 4: Wire `SetupPage`**

Add `useIngestRun` to the `@/lib/hooks/api/useSeed` import and:

```tsx
import { IngestProgressPanel } from "@/components/features/setup/IngestProgressPanel";
```

In the component body, next to `useDataStatus()`:

```tsx
  const ingestRun = useIngestRun();
```

Mount the panel directly beneath the "Process Data" `SetupRow` (line ~374), leaving the existing document-count badge exactly as it is — the two answer different questions ("how many rows landed" vs "which workers have run"), and a completed run with a zero document count is a real, diagnosable state that folding them together would hide:

```tsx
          <IngestProgressPanel
            run={ingestRun.data}
            isError={ingestRun.isError}
          />
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd services/atlas-ui && . "$NVM_DIR/nvm.sh" && nvm use 22 && npx vitest run && npm run build
```

Expected: PASS, build clean. If `SetupPage.test.tsx` fails because `useIngestRun` is unmocked, add it to that file's existing `useSeed` mock returning `{ data: undefined, isError: false }`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/pages
git commit -m "feat(atlas-ui): surface ingest progress and gate baseline publish

Both pages mount the shared panel — Setup at tenant scope, Baselines at
shared scope — and the Publish Baseline control is disabled with an
explanation while the selected region/version's ingest is running, stuck,
failed, or unknown."
```

---

### Task 12: Full verification sweep

Nothing here is optional. `go build`/`go test` against the workspace `go.work` will not catch a missing `COPY libs/...` line in the shared Dockerfile — only `docker buildx bake` will, and `libs/atlas-redis` changed in Task 1.

**Files:** none (fix-forward only).

- [ ] **Step 1: Go modules**

```bash
cd libs/atlas-redis && go test -race ./... && go vet ./... && cd -
cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./... && cd -
```

Expected: all clean.

- [ ] **Step 2: Docker bake**

```bash
docker buildx bake atlas-data
```

Expected: success. No new shared lib was added, so no `Dockerfile`/`go.work` edit is expected — the bake is what proves it.

- [ ] **Step 3: Guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. `tools/lint.sh --check` needs node 22 on PATH — source nvm first (`. "$NVM_DIR/nvm.sh" && nvm use 22`) or it false-fails. If it reports formatting diffs, run `tools/lint.sh` (no flags) to fix in place and amend.

Guards deliberately not run: `service-registration-guard` (no new service, no `services.json`/`deploy/k8s`/`docker-bake.hcl`/`go.work` change), the four template guards (no template change), `skill-job-id-guard`, `buff-duration-guard`.

- [ ] **Step 4: UI**

```bash
cd services/atlas-ui && . "$NVM_DIR/nvm.sh" && nvm use 22 && npm run build
```

Expected: clean. `npm run build` type-checks tests, so this is the per-task gate, not `vitest` alone.

- [ ] **Step 5: Walk the PRD acceptance criteria**

Confirm each `docs/tasks/task-203-ingest-progress-visibility/prd.md` §10 checkbox has a test or a verification command behind it, and note in the commit message any that are runtime-only (they require a live cluster: the live-poll monotonicity criterion and the garbage-collection criterion are covered structurally by `TestProcessStatusTerminalRecordServedWithoutK8s` and `TestSinkWorkerTransitions`, not by an actual GC event).

- [ ] **Step 6: Commit any fix-ups**

```bash
git add -A
git commit -m "chore(task-203): verification sweep fix-ups"
```

If nothing needed fixing, skip the commit.

---

## Self-Review

**Spec coverage.**

| Requirement | Task |
|---|---|
| FR-1.1/1.2 run record per triple, overwriting | 2, 4 |
| FR-1.3 four stored phases | 2 (enum), 4 (`running`), 5 (`stuck`), 7 (`succeeded`/`failed`) |
| FR-1.4 five worker states with timings | 2, 7 |
| FR-1.5 `skipped` for `ErrCategoryAbsent`, non-blocking | 2 (`CompleteCount`), 6 (`runWithProgress`) |
| FR-1.6 roster from `workers.Registered` | 3, 4 |
| FR-2.1 init before the prerequisite phase | 7 |
| FR-2.2 transitions at the `runOne` chokepoint, both phases | 6 |
| FR-2.3 optimistic-lock `Update`, not `Get`+`Put` | 1, 7 |
| FR-2.4 terminal write even when the errgroup aborts | 7 (`writeCtx`, `Finish`) |
| FR-2.5 write failure never fails the run | 6 (interface returns nothing), 7 (`TestSinkSurvivesRedisOutage`) |
| FR-2.6 skip silently without env | 7 (`ingestJobSuffixFromEnv` gate) |
| FR-3.1 Watchdog writes `stuck`, still removes heartbeat keys | 5 |
| FR-3.2 record initialised at Job creation | 4 |
| FR-4.1 shared scope resolution | 8 (`wzinput.ResolveScope`) |
| FR-4.2 caller's scope only | 8 |
| FR-4.3 Redis+k8s merge, `unknown` | 8 (`corroborateRunning`) |
| FR-4.4 absent record → 200/`none` | 8 |
| FR-4.5 no 503 for an absent k8s client | 8 (`IngestRegistries` independent of `JobCreator`) |
| FR-4.6 JSON:API `ingestRun` | 8 |
| FR-5.1 Setup panel, 5s poll | 9, 11 |
| FR-5.2 Baselines panel, shared scope + operator header | 9 (`canonicalHeaders`), 11 |
| FR-5.3 phase, elapsed, count, per-worker list | 10 |
| FR-5.4 `failed`/`stuck` visually distinct, reason surfaced | 10 |
| FR-5.5 `publishDisabled` extended with an explanation | 10, 11 |
| FR-5.6 `none`/`succeeded` do not block | 10 |
| FR-5.7 no error-toast loop | 9 (`retry: false`), 10 |
| F-1 `UpdateWithTTL` | 1 |
| F-2 label round-trip | 4 |
| F-3 registries independent of `JobCreator` | 4, 8 |
| F-4 interface in `data`, impl in `runtime/ingest` | 6, 7 |
| §3.1 superseded-pod guard | 4 (`INGEST_RUN_ID`), 7 (guard) |
| Q1 preserve worker states on `stuck` | 5, 10 (interrupted rendering) |
| Q2 elapsed from the record's `startedAt` | 4, 7 (`Init` adopts) |
| Q3 keep the document-count badge | 11 |
| Docs (`rest.md`) | 8 |
| Verification checklist | 12 |

**Placeholder scan.** No TBD/TODO/"similar to Task N"/"add error handling" remains. Every code step carries real code. Task 11's page tests are described against the existing test file's mocking style rather than reproduced verbatim, because that file's mock scaffolding must be read first — the assertions themselves are given in full.

**Type consistency.** `ingestrun.Record`/`WorkerEntry` field names are identical in Tasks 2, 4, 5, 7, 8. `KeySuffix`, `RunKeySuffix`, `HeartbeatKeySuffix`, `RecordTTL`, `NewJobRegistry`, `NewRunRegistry` are used with their Task-2 spellings everywhere. `IngestRegistries{Job, Run}` is spelled the same in Tasks 4, 5, 8. `ProgressSink.WorkerStarted/WorkerFinished` match between Tasks 6 and 7. `IngestRun`/`IngestPhase`/`IngestRunWorker` match between Tasks 9, 10, 11. `ingestPublishBlockReason(run, isError)` has the same signature in Tasks 10 and 11.
