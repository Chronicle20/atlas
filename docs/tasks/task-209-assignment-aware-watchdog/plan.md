# Assignment-Aware Consumer Watchdog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate `libs/atlas-kafka/consumer` from `*kafka.Reader`-with-`GroupID` to kafka-go's lower-level `ConsumerGroup` + `Generation` API so a consumer can read its own partition assignment, never self-recreates while unassigned, and recovers a stalled partition in place instead of by rejoining the group.

**Architecture:** Option A from [`design.md`](design.md) §3 — one `kafka.ConsumerGroup` per `Consumer` (per topic), configured with the same group ID and the same single-topic subscription as today, so broker-visible group topology, member count and offset keys are bit-identical. The generation loop reads `Generation.Assignments[topic]`; zero assignments parks the consumer as healthy-idle (no ticks, no recreate, no warn logs); each assigned partition gets a plain non-group `*kafka.Reader` positioned by `SetOffset`, whose failures rebuild only that reader. Offsets commit via `Generation.CommitOffsets` at exactly `msg.Offset + 1` — the same value `*kafka.Reader` writes — so `KAFKA_CONSUMER_ENGINE=reader` is a pure pod-restart rollback. The legacy engine is moved verbatim into its own file and retained for one release.

**Tech Stack:** Go 1.26, `github.com/segmentio/kafka-go v0.4.51`, `logrus`, `libs/atlas-routine` (`routine.Go`), `libs/atlas-model`, testify, `logrus/hooks/test`, testcontainers-go (kafka module, `-tags integration`).

## Global Constraints

- **Exported API frozen (FR-3.1).** `Manager`, `GetManager`, `ResetInstance`, `ConfigReaderProducer`, `ManagerConfig`, `Manager.AddConsumer`, `Manager.AddConsumerAndRegister`, `Manager.RegisterHandler`, `Manager.RemoveHandler`, `Manager.Consumers`, `Consumer`, `Consumer.Snapshot`, `Config`, `NewConfig`, `SetStartOffset`, `SetMaxWait`, `SetHeaderParsers`, `SetFetchTimeout`, `SetMaxConsecutiveTimeouts`, `SetMaxInFlight` keep their current signatures and semantics. New capability arrives as **additive** symbols only.
- **No service source changes (FR-3.2).** `git diff --stat services/` MUST be empty at the end of the branch. This is a machine-checked acceptance criterion.
- **Committed offset value is `msg.Offset + 1`, exactly** (design §5; kafka-go `reader.go:1529`, `reader.go:846`). Both engines write the same number for the same delivered message. This is what makes rollback safe in both directions.
- **Never advance the commit cursor on a commit failure (FR-1.3).** Log at Warn, keep the high-water mark, retry it on the next advance. Loss is worse than duplication (risks.md R1).
- **`maxInFlight` default stays 1 (serial)** and the serial path must remain behaviourally identical to today.
- **Every goroutine goes through `routine.Go`** (`libs/atlas-routine`) — including goroutines in test fakes. `tools/goroutine-guard.sh` sweeps `./...` in every module under `services/` and `libs/`, test files included. `kafka.Generation.Start` is a library method call, not a bare `go`, so it passes.
- **`WatchPartitionChanges` is `false`** on the new engine, matching today's behaviour (today's `kafka.ReaderConfig` at `manager.go:161-167` never sets it, and `reader.go:731-732` forwards that `false` into the internal `ConsumerGroupConfig`). Do not enable it as a side effect of this migration (PRD §9.5, risks.md R4).
- **Drain bound is 5 s**, well below kafka-go's `defaultRebalanceTimeout` of 30 s (`consumergroup.go:47`), matching `defaultTimeout` (`consumergroup.go:63`). A slow drain stalls the whole group's rebalance.
- **No `// TODO`, no stubs, no 501s in landed commits** (CLAUDE.md). Every commit in this plan is complete and green on its own.
- **Preserve line endings; never write absolute home paths into committed files** (CLAUDE.md).
- Repo root for all shell commands below is the worktree root: `.worktrees/task-209-assignment-aware-watchdog/`. Module root for Go commands is `libs/atlas-kafka/`.

---

## File Structure

| File | Responsibility |
|---|---|
| `libs/atlas-kafka/consumer/manager.go` (modify) | `Manager`, `Consumer` struct, shared state recorders, `Snapshot`, `processMessage`, `safeHandle`, `fetchBackoff`, the shared `KafkaReader`/`ReaderProducer` seam. Trimmed of the legacy fetch loop. |
| `libs/atlas-kafka/consumer/engine_reader.go` (create) | The legacy `*kafka.Reader` path, moved **verbatim**: `start`, `runFetchLoop`, `runFetchLoopSerial`, `runFetchLoopParallel`, `errFetchWedged`. |
| `libs/atlas-kafka/consumer/cursor.go` (create) | Per-partition prefix-commit cursor: `inflight`, `cursor`, `track`, `mark`, `advance`, `resumeOffset`, `reset`. |
| `libs/atlas-kafka/consumer/group.go` (create) | `Group`/`Generation` interfaces, kafka-go adapters, `GroupProducer`, `PartitionReaderProducer`, `ConfigGroupProducer`, `ConfigPartitionReaderProducer`, and the two config builders. |
| `libs/atlas-kafka/consumer/partition.go` (create) | `partitionState`, per-partition run loop, fetch loop (serial + parallel), quiesce/drain. |
| `libs/atlas-kafka/consumer/engine_group.go` (create) | `startGroupEngine`, `runGenerations`, `onAssignment`, healthy-idle park. |
| `libs/atlas-kafka/consumer/engine.go` (create) | `EngineName`, `KAFKA_CONSUMER_ENGINE` resolution, `ConfigEngine`, `Consumer.start` dispatch. |
| `libs/atlas-kafka/consumer/debug.go` (modify) | Four new debug attributes. |
| `libs/atlas-kafka/consumer/cursor_test.go` (create) | Cursor unit tests. |
| `libs/atlas-kafka/consumer/group_test.go` (create) | Config-builder and adapter tests. |
| `libs/atlas-kafka/consumer/fakegroup_test.go` (create) | Scriptable fake `Group`/`Generation` shared by the new-engine tests. |
| `libs/atlas-kafka/consumer/partition_test.go` (create) | Partition-loop unit tests (commit value, wedge rebuild, drain). |
| `libs/atlas-kafka/consumer/engine_group_test.go` (create) | Assignment-awareness unit tests (FR-2.1 / 2.4 / 2.5). |
| `libs/atlas-kafka/consumer/engine_test.go` (create) | Engine-resolution unit tests. |
| `libs/atlas-kafka/consumer/manager_test.go`, `idle_stuck_test.go`, `timing_test.go`, `debug_test.go` (modify) | Pin to the legacy engine via `ConfigEngine`. |
| `libs/atlas-kafka/consumer/dwell_integration_test.go` (modify) | S6 (members > partitions) and the cross-engine offset round-trip. |
| `libs/atlas-kafka/README.md` (modify) | Document the engine switch, the offset contract, and the `WatchPartitionChanges` opt-in. |
| `docs/tasks/task-209-assignment-aware-watchdog/verification.md` (create) | Recorded output of the branch verification gates. |

---

### Task 1: Move the legacy engine verbatim into `engine_reader.go`

A pure code move, no behaviour edits. It makes the new engine's review diff a clean addition and keeps `git blame` on the legacy path intact for the one release it survives (design §4).

**Files:**
- Create: `libs/atlas-kafka/consumer/engine_reader.go`
- Modify: `libs/atlas-kafka/consumer/manager.go` (remove lines 22-26 and 472-681 — `errFetchWedged`, `start`, `runFetchLoop`, `runFetchLoopSerial`, `runFetchLoopParallel`)
- Test: no new test — the existing suite is the gate.

**Interfaces:**
- Consumes: nothing.
- Produces: `engine_reader.go` holding `errFetchWedged`, `(*Consumer).start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup)`, `(*Consumer).runFetchLoop`, `(*Consumer).runFetchLoopSerial`, `(*Consumer).runFetchLoopParallel` — all with unchanged signatures and bodies.

- [ ] **Step 1: Capture the pre-move baseline**

```bash
cd libs/atlas-kafka && go test -race ./consumer/... 2>&1 | tail -20
```
Expected: `ok  	github.com/Chronicle20/atlas/libs/atlas-kafka/consumer`

- [ ] **Step 2: Create `engine_reader.go` with the moved code**

Create `libs/atlas-kafka/consumer/engine_reader.go`. Cut — do not retype — the following from `manager.go` and paste them under this header:

```go
package consumer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// This file holds the LEGACY consumer engine: one *kafka.Reader per Consumer
// with GroupID set, so kafka-go owns partition assignment, offset commits and
// rebalance handling internally. It is selected by
// KAFKA_CONSUMER_ENGINE=reader and is retained for one release as the
// rollback path for task-209 (see engine.go). Do not add behaviour here —
// the supported engine is engine_group.go.
```

Then move, unmodified:
- `errFetchWedged` (`manager.go:22-26`, including its doc comment)
- `func (c *Consumer) start(...)` (`manager.go:472-518`, including its doc comment)
- `func (c *Consumer) runFetchLoop(...)` (`manager.go:520-528`)
- `func (c *Consumer) runFetchLoopSerial(...)` (`manager.go:530-576`)
- `func (c *Consumer) runFetchLoopParallel(...)` (`manager.go:578-681`)

- [ ] **Step 3: Prune now-unused imports from `manager.go`**

`manager.go` keeps `context`, `errors` (used by `RegisterHandler`/`RemoveHandler`), `fmt`, `sync`, `sync/atomic` (used by `processMessage`), `time`, `uuid`, `kafka`, `logrus`, `otel`, `trace`, `handler`, `model`, `routine`. Verify with the build in Step 4 rather than by eye.

- [ ] **Step 4: Verify the move changed nothing**

```bash
cd libs/atlas-kafka && go build ./... && go vet ./... && go test -race ./consumer/...
```
Expected: build and vet silent; tests PASS with the same test names as Step 1.

- [ ] **Step 5: Confirm the diff is a pure move**

```bash
git diff --stat libs/atlas-kafka/consumer/
```
Expected: `manager.go` loses ~215 lines, `engine_reader.go` gains ~230 (the header comment plus imports). No other file touched.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/consumer/manager.go libs/atlas-kafka/consumer/engine_reader.go
git commit -m "refactor(atlas-kafka): move legacy reader engine to engine_reader.go verbatim"
```

---

### Task 2: Per-partition prefix-commit cursor

The offset cursor is the highest-risk surface on the branch (risks.md R1). It is built and tested standalone before anything wires it up.

**Files:**
- Create: `libs/atlas-kafka/consumer/cursor.go`
- Test: `libs/atlas-kafka/consumer/cursor_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces (all unexported, package `consumer`):
  - `type inflight struct { offset int64; done atomic.Bool; ok atomic.Bool }` — `offset` is the raw `kafka.Message.Offset`.
  - `type cursor struct { … }` with `func newCursor() *cursor`
  - `func (c *cursor) track(offset int64) *inflight`
  - `func (in *inflight) mark(ok bool)`
  - `func (c *cursor) len() int`
  - `func (c *cursor) advance(commit func(offset int64) error) error`
  - `func (c *cursor) committedOffset() int64` — last successfully committed value (`msg.Offset+1`), `-1` when none.
  - `func (c *cursor) resumeOffset(fallback int64) int64`
  - `func (c *cursor) reset()`

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-kafka/consumer/cursor_test.go`:

```go
package consumer

import (
	"errors"
	"sync"
	"testing"
)

// recordingCommitter captures every offset handed to advance's commit func
// and can be armed to fail a fixed number of times.
type recordingCommitter struct {
	mu       sync.Mutex
	offsets  []int64
	failNext int
}

func (r *recordingCommitter) commit(offset int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failNext > 0 {
		r.failNext--
		return errors.New("commit failed")
	}
	r.offsets = append(r.offsets, offset)
	return nil
}

func (r *recordingCommitter) seen() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]int64, len(r.offsets))
	copy(out, r.offsets)
	return out
}

// TestCursorCommitsOffsetPlusOne pins the exact committed value. *kafka.Reader
// commits msg.Offset+1 (kafka-go reader.go:1529); both engines must write the
// identical number or a KAFKA_CONSUMER_ENGINE rollback replays or skips.
func TestCursorCommitsOffsetPlusOne(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	in := cur.track(41)
	in.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}

	got := rc.seen()
	if len(got) != 1 || got[0] != 42 {
		t.Fatalf("committed %v, want [42]", got)
	}
	if cur.committedOffset() != 42 {
		t.Fatalf("committedOffset = %d, want 42", cur.committedOffset())
	}
}

// TestCursorCommitsOnlyContiguousPrefix: message 1 completes before 0, so
// nothing may commit until 0 completes; then a single commit covers both.
func TestCursorCommitsOnlyContiguousPrefix(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	first := cur.track(0)
	second := cur.track(1)

	second.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := rc.seen(); len(got) != 0 {
		t.Fatalf("committed %v before the head completed, want none", got)
	}

	first.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := rc.seen(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("committed %v, want [2]", got)
	}
}

// TestCursorFailedMessageBlocksCursor: a handler failure at the head must
// stop the cursor permanently, so the failed message is redelivered.
func TestCursorFailedMessageBlocksCursor(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	head := cur.track(10)
	tail := cur.track(11)
	head.mark(false)
	tail.mark(true)

	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := rc.seen(); len(got) != 0 {
		t.Fatalf("committed %v past a failed message, want none", got)
	}
	if cur.committedOffset() != -1 {
		t.Fatalf("committedOffset = %d, want -1", cur.committedOffset())
	}
}

// TestCursorCommitFailureDoesNotAdvance is FR-1.3: a failed commit leaves the
// high-water mark in place and the next advance retries the same value.
func TestCursorCommitFailureDoesNotAdvance(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{failNext: 1}

	in := cur.track(7)
	in.mark(true)

	if err := cur.advance(rc.commit); err == nil {
		t.Fatal("advance returned nil on a failing commit, want the error")
	}
	if cur.committedOffset() != -1 {
		t.Fatalf("committedOffset = %d after a failed commit, want -1", cur.committedOffset())
	}

	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("retry advance: %v", err)
	}
	if got := rc.seen(); len(got) != 1 || got[0] != 8 {
		t.Fatalf("retry committed %v, want [8]", got)
	}
}

// TestCursorAdvanceIsIdempotent: advancing with nothing new must not re-issue
// a commit for an offset already committed.
func TestCursorAdvanceIsIdempotent(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	in := cur.track(3)
	in.mark(true)
	for i := 0; i < 3; i++ {
		if err := cur.advance(rc.commit); err != nil {
			t.Fatalf("advance %d: %v", i, err)
		}
	}
	if got := rc.seen(); len(got) != 1 {
		t.Fatalf("committed %v, want exactly one commit", got)
	}
}

// TestCursorResumeOffset: after a partition-reader rebuild the loop must
// resume at the first uncommitted message, or at the committed high-water
// mark, or at the assignment's fallback offset — in that order.
func TestCursorResumeOffset(t *testing.T) {
	cur := newCursor()
	if got := cur.resumeOffset(-2); got != -2 {
		t.Fatalf("empty cursor resumeOffset = %d, want the fallback -2", got)
	}

	rc := &recordingCommitter{}
	done := cur.track(5)
	done.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if got := cur.resumeOffset(-2); got != 6 {
		t.Fatalf("resumeOffset = %d after committing 6, want 6", got)
	}

	cur.track(6)
	if got := cur.resumeOffset(-2); got != 6 {
		t.Fatalf("resumeOffset = %d with offset 6 in flight, want 6", got)
	}
}

// TestCursorResetKeepsCommitted: reset discards pending entries (they will be
// re-fetched by the rebuilt reader) but must never rewind the committed mark.
func TestCursorResetKeepsCommitted(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}
	in := cur.track(0)
	in.mark(true)
	if err := cur.advance(rc.commit); err != nil {
		t.Fatalf("advance: %v", err)
	}
	cur.track(1)
	cur.reset()

	if cur.len() != 0 {
		t.Fatalf("len = %d after reset, want 0", cur.len())
	}
	if cur.committedOffset() != 1 {
		t.Fatalf("committedOffset = %d after reset, want 1", cur.committedOffset())
	}
}

// TestCursorConcurrentAdvance: many handlers completing at once must not
// commit out of order. Under -race this also proves the cursor is safe for
// concurrent use from maxInFlight handler goroutines.
func TestCursorConcurrentAdvance(t *testing.T) {
	cur := newCursor()
	rc := &recordingCommitter{}

	const n = 50
	ins := make([]*inflight, n)
	for i := 0; i < n; i++ {
		ins[i] = cur.track(int64(i))
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ins[idx].mark(true)
			_ = cur.advance(rc.commit)
		}(i)
	}
	wg.Wait()
	_ = cur.advance(rc.commit)

	got := rc.seen()
	if len(got) == 0 {
		t.Fatal("no commits recorded")
	}
	for i := 1; i < len(got); i++ {
		if got[i] <= got[i-1] {
			t.Fatalf("commits went backwards: %v", got)
		}
	}
	if got[len(got)-1] != int64(n) {
		t.Fatalf("final commit %d, want %d", got[len(got)-1], n)
	}
}
```

Note: this test file is package `consumer` (internal), matching `config_test.go`. The bare `go` statements in `TestCursorConcurrentAdvance` are inside `_test.go` and `tools/goroutine-guard.sh` sweeps `./...` — so replace them with `routine.Go`. Use this form instead of the raw `go func` above:

```go
	l, _ := test.NewNullLogger()
	for i := 0; i < n; i++ {
		wg.Add(1)
		idx := i
		routine.Go(l, context.Background(), func(_ context.Context) {
			defer wg.Done()
			ins[idx].mark(true)
			_ = cur.advance(rc.commit)
		})
	}
```

with imports `"context"`, `"github.com/sirupsen/logrus/hooks/test"` and `routine "github.com/Chronicle20/atlas/libs/atlas-routine"` added to the file.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run TestCursor -v
```
Expected: FAIL — `undefined: newCursor`, `undefined: inflight`.

- [ ] **Step 3: Implement the cursor**

Create `libs/atlas-kafka/consumer/cursor.go`:

```go
package consumer

import (
	"sync"
	"sync/atomic"
)

// noCommit marks "no offset has been committed yet". Real committed values
// are msg.Offset+1 and therefore always >= 1, so -1 is unambiguous.
const noCommit int64 = -1

// inflight is one fetched message tracked by a cursor. offset is the raw
// kafka.Message.Offset — the +1 conversion happens once, in advance.
type inflight struct {
	offset int64
	done   atomic.Bool
	ok     atomic.Bool
}

// mark records the outcome of the message's handlers. It must be called
// exactly once per inflight.
func (in *inflight) mark(ok bool) {
	in.ok.Store(ok)
	in.done.Store(true)
}

// cursor is the per-partition prefix-commit cursor. It commits only the
// highest CONTIGUOUSLY-completed offset, so a failed message blocks every
// commit behind it and that message is redelivered (at-least-once).
//
// OFFSET CONTRACT (task-209 design §5): the value handed to the commit func
// is msg.Offset+1 — identical to what *kafka.Reader writes
// (kafka-go reader.go:1529). Both consumer engines therefore write the same
// number for the same delivered message, which is what makes flipping
// KAFKA_CONSUMER_ENGINE a restart rather than a migration.
//
// mu guards the queue and the offset marks. cmu serializes the commit call
// itself so two concurrent advances cannot write offsets out of order; it is
// never held together with mu across a network call.
type cursor struct {
	mu            sync.Mutex
	cmu           sync.Mutex
	pending       []*inflight
	pendingCommit int64 // highest contiguous msg.Offset+1 ready to commit
	committed     int64 // highest msg.Offset+1 successfully committed
}

func newCursor() *cursor {
	return &cursor{pendingCommit: noCommit, committed: noCommit}
}

// track registers a freshly fetched message. Messages must be tracked in
// fetch order — the prefix walk depends on it.
func (c *cursor) track(offset int64) *inflight {
	in := &inflight{offset: offset}
	c.mu.Lock()
	c.pending = append(c.pending, in)
	c.mu.Unlock()
	return in
}

// len reports how many messages are still queued. The parallel fetch loop
// uses it for back-pressure.
func (c *cursor) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pending)
}

// advance walks the completed-and-successful prefix, drops it from the queue,
// and commits the resulting high-water mark. On a commit error the mark is
// RETAINED (FR-1.3) — committed does not move, so the next advance retries
// the same value. Returns the commit error, if any.
func (c *cursor) advance(commit func(offset int64) error) error {
	c.mu.Lock()
	i := 0
	for i < len(c.pending) && c.pending[i].done.Load() && c.pending[i].ok.Load() {
		i++
	}
	if i > 0 {
		c.pendingCommit = c.pending[i-1].offset + 1
		c.pending = c.pending[i:]
	}
	c.mu.Unlock()

	c.cmu.Lock()
	defer c.cmu.Unlock()

	c.mu.Lock()
	target := c.pendingCommit
	already := c.committed
	c.mu.Unlock()
	if target == noCommit || target <= already {
		return nil
	}

	if err := commit(target); err != nil {
		return err
	}

	c.mu.Lock()
	if target > c.committed {
		c.committed = target
	}
	c.mu.Unlock()
	return nil
}

// committedOffset returns the last successfully committed value, or noCommit.
func (c *cursor) committedOffset() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.committed
}

// resumeOffset returns the offset a rebuilt partition reader must start from:
// the first still-uncommitted message if one is queued, else the committed
// high-water mark, else fallback (the generation's PartitionAssignment.Offset,
// which may be the kafka.FirstOffset/LastOffset sentinel).
func (c *cursor) resumeOffset(fallback int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) > 0 {
		return c.pending[0].offset
	}
	if c.committed != noCommit {
		return c.committed
	}
	return fallback
}

// reset discards queued entries after the caller has quiesced in-flight
// handlers and computed a resume offset. The committed mark is never rewound.
func (c *cursor) reset() {
	c.mu.Lock()
	c.pending = nil
	c.pendingCommit = noCommit
	c.mu.Unlock()
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run TestCursor -v
```
Expected: all eight `TestCursor*` tests PASS.

- [ ] **Step 5: Run the whole module suite**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./...
```
Expected: PASS, vet silent.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/consumer/cursor.go libs/atlas-kafka/consumer/cursor_test.go
git commit -m "feat(atlas-kafka): add per-partition prefix-commit cursor"
```

---

### Task 3: `Group`/`Generation` seams and the two config builders

**Files:**
- Create: `libs/atlas-kafka/consumer/group.go`
- Modify: `libs/atlas-kafka/consumer/manager.go` (add `gp`/`prp` to `Manager` with defaults; add `maxWait`, `startOffset`, `gp`, `prp` to `Consumer`; populate them in `AddConsumer`)
- Test: `libs/atlas-kafka/consumer/group_test.go`

**Interfaces:**
- Consumes: `KafkaReader` (`manager.go:28`), `ManagerConfig` (`manager.go:76`), `Manager`, `Consumer`.
- Produces:
  - `type Group interface { Next(ctx context.Context) (Generation, error); Close() error }`
  - `type Generation interface { ID() int32; Assignments() map[string][]kafka.PartitionAssignment; Start(fn func(ctx context.Context)); CommitOffsets(offsets map[string]map[int]int64) error }`
  - `type GroupProducer func(cfg kafka.ConsumerGroupConfig) (Group, error)`
  - `type PartitionReaderProducer func(cfg kafka.ReaderConfig, offset int64) KafkaReader`
  - `func ConfigGroupProducer(gp GroupProducer) ManagerConfig`
  - `func ConfigPartitionReaderProducer(prp PartitionReaderProducer) ManagerConfig`
  - `func (c *Consumer) groupConfig() kafka.ConsumerGroupConfig`
  - `func (c *Consumer) partitionReaderConfig(partition int) kafka.ReaderConfig`
  - New `Consumer` fields: `maxWait time.Duration`, `startOffset int64`, `gp GroupProducer`, `prp PartitionReaderProducer`.
  - New `Manager` fields: `gp GroupProducer`, `prp PartitionReaderProducer`.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-kafka/consumer/group_test.go`:

```go
package consumer

import (
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// TestGroupConfigMirrorsTodaysTopology pins Option A: the new engine joins
// with the same group ID and the same single-topic subscription as the legacy
// reader, so broker-visible group topology is unchanged and rollback needs no
// group-state migration (FR-3.4, FR-5.2).
func TestGroupConfigMirrorsTodaysTopology(t *testing.T) {
	c := &Consumer{
		topic:       "EVENT_TOPIC_TEST",
		groupId:     "Test Service",
		brokers:     []string{"broker-a:9092", "broker-b:9092"},
		maxWait:     10 * time.Second,
		startOffset: kafka.FirstOffset,
	}

	cfg := c.groupConfig()

	if cfg.ID != "Test Service" {
		t.Fatalf("ID = %q, want %q", cfg.ID, "Test Service")
	}
	if len(cfg.Topics) != 1 || cfg.Topics[0] != "EVENT_TOPIC_TEST" {
		t.Fatalf("Topics = %v, want [EVENT_TOPIC_TEST]", cfg.Topics)
	}
	if len(cfg.Brokers) != 2 || cfg.Brokers[0] != "broker-a:9092" {
		t.Fatalf("Brokers = %v", cfg.Brokers)
	}
	if cfg.StartOffset != kafka.FirstOffset {
		t.Fatalf("StartOffset = %d, want FirstOffset (%d)", cfg.StartOffset, kafka.FirstOffset)
	}
	// Today's ReaderConfig never sets WatchPartitionChanges, and kafka-go
	// forwards that false into its internal ConsumerGroupConfig
	// (reader.go:731-732). Enabling it here would be a behaviour change
	// smuggled into a migration (PRD §9.5).
	if cfg.WatchPartitionChanges {
		t.Fatal("WatchPartitionChanges = true, want false to match the legacy engine")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestGroupConfigHonoursLastOffset covers SetStartOffset(kafka.LastOffset).
func TestGroupConfigHonoursLastOffset(t *testing.T) {
	c := &Consumer{
		topic:       "t",
		groupId:     "g",
		brokers:     []string{"b:9092"},
		maxWait:     time.Second,
		startOffset: kafka.LastOffset,
	}
	if got := c.groupConfig().StartOffset; got != kafka.LastOffset {
		t.Fatalf("StartOffset = %d, want LastOffset (%d)", got, kafka.LastOffset)
	}
}

// TestPartitionReaderConfigHasNoGroupID: a partition reader is positioned by
// SetOffset, which kafka-go rejects on a group reader
// (errNotAvailableWithGroup, reader.go:36). Setting both Partition and GroupID
// also fails ReaderConfig.Validate (reader.go:545-547).
func TestPartitionReaderConfigHasNoGroupID(t *testing.T) {
	c := &Consumer{
		topic:       "EVENT_TOPIC_TEST",
		groupId:     "Test Service",
		brokers:     []string{"broker-a:9092"},
		maxWait:     7 * time.Second,
		startOffset: kafka.FirstOffset,
	}

	cfg := c.partitionReaderConfig(3)

	if cfg.GroupID != "" {
		t.Fatalf("GroupID = %q, want empty", cfg.GroupID)
	}
	if cfg.Partition != 3 {
		t.Fatalf("Partition = %d, want 3", cfg.Partition)
	}
	if cfg.Topic != "EVENT_TOPIC_TEST" {
		t.Fatalf("Topic = %q", cfg.Topic)
	}
	if cfg.MaxWait != 7*time.Second {
		t.Fatalf("MaxWait = %v, want 7s", cfg.MaxWait)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestManagerDefaultProducersArePresent: GetManager must install working
// defaults so a service that configures nothing still gets a real group and
// real partition readers.
func TestManagerDefaultProducersArePresent(t *testing.T) {
	ResetInstance()
	m := GetManager()
	if m.gp == nil {
		t.Fatal("Manager.gp is nil; GetManager must install a default GroupProducer")
	}
	if m.prp == nil {
		t.Fatal("Manager.prp is nil; GetManager must install a default PartitionReaderProducer")
	}
	ResetInstance()
}

// TestConfigProducersOverrideDefaults covers the additive test seams.
func TestConfigProducersOverrideDefaults(t *testing.T) {
	ResetInstance()
	var gpCalled, prpCalled bool
	m := GetManager(
		ConfigGroupProducer(func(kafka.ConsumerGroupConfig) (Group, error) { gpCalled = true; return nil, nil }),
		ConfigPartitionReaderProducer(func(kafka.ReaderConfig, int64) KafkaReader { prpCalled = true; return nil }),
	)
	_, _ = m.gp(kafka.ConsumerGroupConfig{})
	_ = m.prp(kafka.ReaderConfig{}, 0)
	if !gpCalled || !prpCalled {
		t.Fatalf("configured producers not used: gp=%v prp=%v", gpCalled, prpCalled)
	}
	ResetInstance()
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestGroupConfig|TestPartitionReaderConfig|TestManagerDefaultProducers|TestConfigProducersOverride' -v
```
Expected: FAIL — `c.groupConfig undefined`, `m.gp undefined`, `undefined: ConfigGroupProducer`.

- [ ] **Step 3: Create `group.go`**

```go
package consumer

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// Group is the subset of *kafka.ConsumerGroup this package depends on.
// Defining it as an interface is what lets the assignment-awareness tests
// script generations without a broker (task-209 design §6).
type Group interface {
	// Next blocks until the previous generation has ended and the next one
	// is ready. It returns ErrGroupClosed after Close, or the ctx error.
	Next(ctx context.Context) (Generation, error)
	Close() error
}

// Generation is the subset of *kafka.Generation this package depends on.
//
// IMPORTANT: a function passed to Start ENDS THE GENERATION when it returns
// (kafka-go consumergroup.go:387-405). Every recoverable error must therefore
// be handled inside the function; returning is reserved for "the generation
// is already ending". That inversion — recover in place, never by rejoining —
// is the point of the migration (design §1.1).
type Generation interface {
	ID() int32
	Assignments() map[string][]kafka.PartitionAssignment
	Start(fn func(ctx context.Context))
	CommitOffsets(offsets map[string]map[int]int64) error
}

// GroupProducer builds a Group from a kafka-go consumer-group config.
type GroupProducer func(cfg kafka.ConsumerGroupConfig) (Group, error)

// PartitionReaderProducer builds a positioned, non-group reader for a single
// partition. The offset is a producer argument rather than a KafkaReader
// method so the existing KafkaReader seam stays frozen (FR-3.5): every mock
// written against KafkaReader works unchanged against the new fetch loop.
type PartitionReaderProducer func(cfg kafka.ReaderConfig, offset int64) KafkaReader

//goland:noinspection GoUnusedExportedFunction
func ConfigGroupProducer(gp GroupProducer) ManagerConfig {
	return func(m *Manager) {
		m.gp = gp
	}
}

//goland:noinspection GoUnusedExportedFunction
func ConfigPartitionReaderProducer(prp PartitionReaderProducer) ManagerConfig {
	return func(m *Manager) {
		m.prp = prp
	}
}

// defaultGroupProducer wraps kafka.NewConsumerGroup.
func defaultGroupProducer(cfg kafka.ConsumerGroupConfig) (Group, error) {
	cg, err := kafka.NewConsumerGroup(cfg)
	if err != nil {
		return nil, err
	}
	return &kafkaGroup{cg: cg}, nil
}

// defaultPartitionReaderProducer builds a plain (non-group) reader and
// positions it. SetOffset only fails on a closed reader or a group reader
// (kafka-go reader.go SetOffset); this reader is neither, and a no-op when
// offset already equals the reader's default FirstOffset.
func defaultPartitionReaderProducer(cfg kafka.ReaderConfig, offset int64) KafkaReader {
	r := kafka.NewReader(cfg)
	_ = r.SetOffset(offset)
	return r
}

type kafkaGroup struct {
	cg *kafka.ConsumerGroup
}

func (g *kafkaGroup) Next(ctx context.Context) (Generation, error) {
	gen, err := g.cg.Next(ctx)
	if err != nil {
		return nil, err
	}
	return &kafkaGeneration{gen: gen}, nil
}

func (g *kafkaGroup) Close() error { return g.cg.Close() }

type kafkaGeneration struct {
	gen *kafka.Generation
}

func (g *kafkaGeneration) ID() int32 { return g.gen.ID }

func (g *kafkaGeneration) Assignments() map[string][]kafka.PartitionAssignment {
	return g.gen.Assignments
}

func (g *kafkaGeneration) Start(fn func(ctx context.Context)) { g.gen.Start(fn) }

func (g *kafkaGeneration) CommitOffsets(offsets map[string]map[int]int64) error {
	return g.gen.CommitOffsets(offsets)
}

// groupConfig builds the consumer-group config for this Consumer. It is a
// deliberate 1:1 mirror of what kafka-go derives internally from today's
// ReaderConfig (reader.go:717-733): same group ID, same single-topic
// subscription, same StartOffset, WatchPartitionChanges left false.
func (c *Consumer) groupConfig() kafka.ConsumerGroupConfig {
	return kafka.ConsumerGroupConfig{
		ID:                    c.groupId,
		Brokers:               append([]string(nil), c.brokers...),
		Topics:                []string{c.topic},
		StartOffset:           c.startOffset,
		WatchPartitionChanges: false,
	}
}

// partitionReaderConfig builds the config for one assigned partition's
// reader. GroupID is deliberately absent: group membership belongs to the
// ConsumerGroup, and kafka-go rejects a reader with both Partition and
// GroupID set (reader.go:545-547).
func (c *Consumer) partitionReaderConfig(partition int) kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers:   append([]string(nil), c.brokers...),
		Topic:     c.topic,
		Partition: partition,
		MaxWait:   c.maxWait,
	}
}
```

- [ ] **Step 4: Wire the producers through `Manager` and `Consumer`**

In `libs/atlas-kafka/consumer/manager.go`, extend the `Manager` struct (currently `manager.go:85-89`):

```go
type Manager struct {
	mu        *sync.Mutex
	consumers map[string]*Consumer
	rp        ReaderProducer
	gp        GroupProducer
	prp       PartitionReaderProducer
}
```

and the `GetManager` defaults (currently `manager.go:103-114`) — add the two new defaults next to `rp`, before the configurator loop:

```go
			rp: func(config kafka.ReaderConfig) KafkaReader {
				return kafka.NewReader(config)
			},
			gp:  defaultGroupProducer,
			prp: defaultPartitionReaderProducer,
```

Extend the `Consumer` struct (`manager.go:237-278`), in the "Read-only after construction" block:

```go
	// Read-only after construction; copied from Config in AddConsumer.
	fetchTimeout           time.Duration
	maxConsecutiveTimeouts int
	maxInFlight            int
	maxWait                time.Duration
	startOffset            int64
```

and add the two producers next to `rp` in the top block:

```go
	rp            ReaderProducer
	gp            GroupProducer
	prp           PartitionReaderProducer
```

Finally, populate them in `AddConsumer`'s `con := &Consumer{...}` literal (`manager.go:173-185`):

```go
			maxInFlight:            maxInFlight,
			maxWait:                c.maxWait,
			startOffset:            c.startOffset,
			gp:                     m.gp,
			prp:                    m.prp,
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestGroupConfig|TestPartitionReaderConfig|TestManagerDefaultProducers|TestConfigProducersOverride' -v
```
Expected: all five PASS.

- [ ] **Step 6: Run the whole module suite**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./...
```
Expected: PASS. The existing tests are untouched — the legacy engine is still what `start` runs.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-kafka/consumer/group.go libs/atlas-kafka/consumer/group_test.go libs/atlas-kafka/consumer/manager.go
git commit -m "feat(atlas-kafka): add ConsumerGroup/Generation seams and config builders"
```

---

### Task 4: Per-partition watchdog state and assignment observability

The watchdog counters move from `Consumer`-level scalars to a `map[int]*partitionState`, because a two-partition topic where one partition wedges and the other flows would otherwise have the wedge masked by the healthy partition's resets (design §7). The legacy engine keys the map with `legacyPartition = -1`, so its single entry aggregates to exactly today's `Snapshot` values (FR-4.4) and its 1400 lines of tests stay green unchanged.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go` (state fields, recorders, `Snapshot`)
- Modify: `libs/atlas-kafka/consumer/engine_reader.go` (pass `legacyPartition` to the recorders)
- Modify: `libs/atlas-kafka/consumer/debug.go` (three new attributes; the fourth arrives in Task 7)
- Test: `libs/atlas-kafka/consumer/state_test.go` (create)

**Interfaces:**
- Consumes: `Consumer`, `Snapshot`, `fetchBackoff` (`manager.go:448-470`).
- Produces:
  - `const legacyPartition = -1`
  - `type partitionState struct { consecutiveTimeouts int; lastTimeoutAt, lastIdleTickAt, lastNoProgressAt, lastFetchAt time.Time; idleTicks, noProgressTicks int; backoff *fetchBackoff }`
  - `func (c *Consumer) partitionStateFor(partition int) *partitionState`
  - `func (c *Consumer) onAssignment(genID int32, partitions []int)`
  - Recorders re-signed with a leading `partition int`: `onReaderCreated(partition, attempt int)`, `recordFetch(partition int)`, `recordIdleTick(partition int)`, `recordNoProgressTick(partition int) int`, `handleFetchDeadline(l logrus.FieldLogger, reader KafkaReader, partition int) error`
  - `Snapshot` gains `AssignedPartitions []int`, `GenerationID int32`, `LastAssignmentAt time.Time`.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-kafka/consumer/state_test.go`:

```go
package consumer

import (
	"testing"
	"time"
)

func newTestConsumer() *Consumer {
	return &Consumer{
		topic:                  "t",
		groupId:                "g",
		maxConsecutiveTimeouts: 3,
		partitions:             make(map[int]*partitionState),
	}
}

// TestSnapshotAggregatesAcrossPartitions pins the aggregation rules from
// design §7: max for ConsecutiveTimeouts, sum for tick counts, most-recent
// for timestamps. With a single partition (every atlas-main topic today) the
// aggregate is bit-identical to the pre-task scalars — FR-4.4.
func TestSnapshotAggregatesAcrossPartitions(t *testing.T) {
	c := newTestConsumer()

	p0 := c.partitionStateFor(0)
	p1 := c.partitionStateFor(1)

	early := time.Now().Add(-time.Minute)
	late := time.Now()

	p0.consecutiveTimeouts = 1
	p0.idleTicks = 2
	p0.noProgressTicks = 1
	p0.lastTimeoutAt = early
	p0.lastFetchAt = early

	p1.consecutiveTimeouts = 4
	p1.idleTicks = 3
	p1.noProgressTicks = 5
	p1.lastTimeoutAt = late
	p1.lastFetchAt = late

	s := c.Snapshot()

	if s.ConsecutiveTimeouts != 4 {
		t.Fatalf("ConsecutiveTimeouts = %d, want max 4", s.ConsecutiveTimeouts)
	}
	if s.IdleTicks != 5 {
		t.Fatalf("IdleTicks = %d, want sum 5", s.IdleTicks)
	}
	if s.NoProgressTicks != 6 {
		t.Fatalf("NoProgressTicks = %d, want sum 6", s.NoProgressTicks)
	}
	if !s.LastTimeoutAt.Equal(late) {
		t.Fatalf("LastTimeoutAt = %v, want the most recent %v", s.LastTimeoutAt, late)
	}
	if !s.LastFetchAt.Equal(late) {
		t.Fatalf("LastFetchAt = %v, want the most recent %v", s.LastFetchAt, late)
	}
}

// TestSnapshotAssignedPartitionsIsNeverNil: the debug route serialises this
// field, and a nil slice renders as JSON null rather than [].
func TestSnapshotAssignedPartitionsIsNeverNil(t *testing.T) {
	c := newTestConsumer()
	s := c.Snapshot()
	if s.AssignedPartitions == nil {
		t.Fatal("AssignedPartitions is nil, want an empty slice")
	}
	if len(s.AssignedPartitions) != 0 {
		t.Fatalf("AssignedPartitions = %v, want empty", s.AssignedPartitions)
	}
}

// TestOnAssignmentRecordsGenerationAndPartitions covers FR-4.1.
func TestOnAssignmentRecordsGenerationAndPartitions(t *testing.T) {
	c := newTestConsumer()
	before := time.Now()

	c.onAssignment(7, []int{2, 0})

	s := c.Snapshot()
	if s.GenerationID != 7 {
		t.Fatalf("GenerationID = %d, want 7", s.GenerationID)
	}
	if len(s.AssignedPartitions) != 2 || s.AssignedPartitions[0] != 0 || s.AssignedPartitions[1] != 2 {
		t.Fatalf("AssignedPartitions = %v, want sorted [0 2]", s.AssignedPartitions)
	}
	if s.LastAssignmentAt.Before(before) {
		t.Fatalf("LastAssignmentAt = %v, want >= %v", s.LastAssignmentAt, before)
	}
}

// TestOnAssignmentResetsNewlyAssignedPartitionState is FR-2.4: a partition
// that arrives after a period of being unassigned starts with clean
// no-progress state, so stale ticks cannot push it straight into a wedge.
func TestOnAssignmentResetsNewlyAssignedPartitionState(t *testing.T) {
	c := newTestConsumer()

	c.onAssignment(1, []int{0})
	st := c.partitionStateFor(0)
	st.consecutiveTimeouts = 2
	st.noProgressTicks = 2

	// Generation 2 assigns nothing: partition 0's state is dropped.
	c.onAssignment(2, nil)
	if got := c.Snapshot().NoProgressTicks; got != 0 {
		t.Fatalf("NoProgressTicks = %d after losing the assignment, want 0", got)
	}

	// Generation 3 re-assigns it: state starts clean.
	c.onAssignment(3, []int{0})
	if got := c.partitionStateFor(0).consecutiveTimeouts; got != 0 {
		t.Fatalf("consecutiveTimeouts = %d on re-assignment, want 0", got)
	}
}

// TestOnAssignmentKeepsStateForRetainedPartitions: a generation change that
// keeps the same partition must not wipe its counters — operators read the
// accumulated idle-tick history from the debug route.
func TestOnAssignmentKeepsStateForRetainedPartitions(t *testing.T) {
	c := newTestConsumer()
	c.onAssignment(1, []int{0})
	c.partitionStateFor(0).idleTicks = 5

	c.onAssignment(2, []int{0})

	if got := c.Snapshot().IdleTicks; got != 5 {
		t.Fatalf("IdleTicks = %d after a generation change that kept partition 0, want 5", got)
	}
}

// TestRecordersAreKeyedByPartition proves one partition's wedge is not masked
// by another's healthy resets — the correctness reason the state moved off
// Consumer-level scalars (design §7).
func TestRecordersAreKeyedByPartition(t *testing.T) {
	c := newTestConsumer()

	c.recordNoProgressTick(0)
	c.recordNoProgressTick(0)
	c.recordIdleTick(1)

	if got := c.partitionStateFor(0).consecutiveTimeouts; got != 2 {
		t.Fatalf("partition 0 consecutiveTimeouts = %d, want 2", got)
	}
	if got := c.partitionStateFor(1).consecutiveTimeouts; got != 0 {
		t.Fatalf("partition 1 consecutiveTimeouts = %d, want 0", got)
	}
	if got := c.Snapshot().ConsecutiveTimeouts; got != 2 {
		t.Fatalf("Snapshot.ConsecutiveTimeouts = %d, want max 2", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestSnapshotAggregates|TestSnapshotAssignedPartitions|TestOnAssignment|TestRecordersAreKeyed' -v
```
Expected: FAIL — `undefined: partitionState`, `c.partitions undefined`, `c.onAssignment undefined`.

- [ ] **Step 3: Replace the scalar state with the per-partition map**

In `libs/atlas-kafka/consumer/manager.go`, replace these fields in the `Consumer` struct's "Observable state" block:

```go
	consecutiveTimeouts int
	lastTimeoutAt       time.Time
	idleTicks           int
	lastIdleTickAt      time.Time
	noProgressTicks     int
	lastNoProgressAt    time.Time
	lastFetchAt         time.Time
```

with:

```go
	// Watchdog counters live per assigned partition. The legacy engine has
	// no partition of its own and uses the legacyPartition key, so its
	// single-entry map aggregates in Snapshot to exactly the scalars this
	// replaced (task-209 design §7).
	partitions map[int]*partitionState

	// Assignment state — meaningful on the consumergroup engine only.
	assignedPartitions []int
	generationID       int32
	lastAssignmentAt   time.Time
```

(`aliveSince`, `lastErrorAt`, `lastError`, `recreateCount` and every phase-timing field stay `Consumer`-level and unchanged.)

Add, next to the `Consumer` struct:

```go
// legacyPartition is the partitionState key used by the legacy *kafka.Reader
// engine, which never sees a partition id of its own. Real partition ids are
// non-negative, so -1 can never collide.
const legacyPartition = -1

// partitionState holds the liveness-watchdog counters for one partition. A
// scalar would let a healthy partition's resets mask a sibling partition's
// wedge; keying by partition is why the wedge stays detectable on a
// multi-partition topic.
type partitionState struct {
	consecutiveTimeouts int
	lastTimeoutAt       time.Time
	idleTicks           int
	lastIdleTickAt      time.Time
	noProgressTicks     int
	lastNoProgressAt    time.Time
	lastFetchAt         time.Time
	backoff             *fetchBackoff
}

func newPartitionState() *partitionState {
	return &partitionState{backoff: newFetchBackoff()}
}

// partitionStateFor returns the state for partition, creating it on first use.
func (c *Consumer) partitionStateFor(partition int) *partitionState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.partitionStateLocked(partition)
}

func (c *Consumer) partitionStateLocked(partition int) *partitionState {
	if c.partitions == nil {
		c.partitions = make(map[int]*partitionState)
	}
	st, ok := c.partitions[partition]
	if !ok {
		st = newPartitionState()
		c.partitions[partition] = st
	}
	return st
}

// onAssignment records the generation's assignment for this consumer's topic.
// State for partitions the consumer no longer holds is dropped, so a
// partition that comes back after a gap starts with clean no-progress
// counters (FR-2.4); state for partitions retained across the generation is
// preserved so operators keep the accumulated tick history.
func (c *Consumer) onAssignment(genID int32, partitions []int) {
	sorted := append([]int(nil), partitions...)
	sort.Ints(sorted)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.generationID = genID
	c.lastAssignmentAt = time.Now()
	c.assignedPartitions = sorted

	next := make(map[int]*partitionState, len(sorted))
	for _, p := range sorted {
		if st, ok := c.partitions[p]; ok {
			next[p] = st
		} else {
			next[p] = newPartitionState()
		}
	}
	c.partitions = next
}
```

Add `"sort"` to `manager.go`'s imports.

- [ ] **Step 4: Re-sign the recorders and `handleFetchDeadline`**

Replace the bodies of `onReaderCreated`, `recordFetch`, `recordIdleTick`, `recordNoProgressTick` and `handleFetchDeadline` in `manager.go`:

```go
func (c *Consumer) onReaderCreated(partition int, attempt int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.aliveSince = now
	c.readerCreatedAt = now
	c.awaitingFirstFetch = true
	if attempt > 0 {
		c.recreateCount++
		c.lastError = ""
		st := c.partitionStateLocked(partition)
		st.consecutiveTimeouts = 0
		st.lastTimeoutAt = time.Time{}
	}
}

func (c *Consumer) recordFetch(partition int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.lastError = ""
	st := c.partitionStateLocked(partition)
	st.lastFetchAt = now
	st.consecutiveTimeouts = 0
	if c.awaitingFirstFetch {
		c.timeToFirstFetch = now.Sub(c.readerCreatedAt)
		c.awaitingFirstFetch = false
	}
}

// recordIdleTick marks one deadline expiration on a reader that is still
// making fetch attempts. Idle is healthy: it resets the no-progress
// escalation counter and touches no error state.
func (c *Consumer) recordIdleTick(partition int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.partitionStateLocked(partition)
	st.idleTicks++
	st.lastIdleTickAt = time.Now()
	st.consecutiveTimeouts = 0
}

// recordNoProgressTick marks one deadline expiration with zero reader
// progress — a stall suspect. Returns the new consecutive count so callers
// can branch on the threshold without a second mutex acquisition.
func (c *Consumer) recordNoProgressTick(partition int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	st := c.partitionStateLocked(partition)
	st.lastTimeoutAt = now
	st.lastNoProgressAt = now
	st.noProgressTicks++
	st.consecutiveTimeouts++
	return st.consecutiveTimeouts
}

// handleFetchDeadline classifies one expired fetch deadline: an idle tick
// (reader made progress — normal on a no-traffic topic) or a no-progress
// tick (stall suspect). Returns errFetchWedged once consecutive no-progress
// ticks reach the threshold, nil otherwise.
//
// On the consumergroup engine this is only ever reached from a reader that
// HOLDS a partition assignment, which is what makes FR-2.5 structural: an
// unassigned consumer has no partition loop and so cannot emit these warns.
func (c *Consumer) handleFetchDeadline(l logrus.FieldLogger, reader KafkaReader, partition int) error {
	if readerMadeProgress(reader) {
		c.recordIdleTick(partition)
		l.Debugf("Fetch deadline expired on idle topic [%s]; reader healthy, continuing.", c.topic)
		return nil
	}
	consecutive := c.recordNoProgressTick(partition)
	if consecutive >= c.maxConsecutiveTimeouts {
		l.Warnf("FetchMessage wedged: %d consecutive no-progress ticks on topic [%s] (group [%s]); forcing reader recreate.",
			consecutive, c.topic, c.groupId)
		return errFetchWedged
	}
	l.Warnf("FetchMessage made no progress on topic [%s] (group [%s]) (consecutive=%d/%d); stall suspect.",
		c.topic, c.groupId, consecutive, c.maxConsecutiveTimeouts)
	return nil
}
```

- [ ] **Step 5: Update the legacy engine's call sites**

In `engine_reader.go`, change exactly four call sites to pass `legacyPartition`:
- `c.onReaderCreated(attempt)` → `c.onReaderCreated(legacyPartition, attempt)`
- `c.recordFetch()` → `c.recordFetch(legacyPartition)` (two occurrences: serial and parallel loops)
- `c.handleFetchDeadline(l, reader)` → `c.handleFetchDeadline(l, reader, legacyPartition)` (two occurrences)

- [ ] **Step 6: Aggregate in `Snapshot` and add the new fields**

Add to the `Snapshot` struct in `manager.go`, after `LastNoProgressAt`:

```go
	// AssignedPartitions is the sorted partition list this consumer holds in
	// the current generation. Always non-nil; empty means healthy-idle (or
	// the legacy engine, which does not observe assignment).
	AssignedPartitions []int
	GenerationID       int32
	LastAssignmentAt   time.Time
```

and replace `Consumer.Snapshot`'s body with:

```go
func (c *Consumer) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	var (
		consecutive      int
		idleTicks        int
		noProgressTicks  int
		lastTimeoutAt    time.Time
		lastIdleTickAt   time.Time
		lastNoProgressAt time.Time
		lastFetchAt      time.Time
	)
	latest := func(dst *time.Time, v time.Time) {
		if v.After(*dst) {
			*dst = v
		}
	}
	for _, st := range c.partitions {
		if st.consecutiveTimeouts > consecutive {
			consecutive = st.consecutiveTimeouts
		}
		idleTicks += st.idleTicks
		noProgressTicks += st.noProgressTicks
		latest(&lastTimeoutAt, st.lastTimeoutAt)
		latest(&lastIdleTickAt, st.lastIdleTickAt)
		latest(&lastNoProgressAt, st.lastNoProgressAt)
		latest(&lastFetchAt, st.lastFetchAt)
	}

	return Snapshot{
		Name:                c.name,
		Topic:               c.topic,
		GroupID:             c.groupId,
		Brokers:             append([]string(nil), c.brokers...),
		AliveSince:          c.aliveSince,
		LastFetchAt:         lastFetchAt,
		LastErrorAt:         c.lastErrorAt,
		LastError:           c.lastError,
		RecreateCount:       c.recreateCount,
		HandlerCount:        len(c.handlers),
		LastTimeoutAt:       lastTimeoutAt,
		ConsecutiveTimeouts: consecutive,
		IdleTicks:           idleTicks,
		LastIdleTickAt:      lastIdleTickAt,
		NoProgressTicks:     noProgressTicks,
		LastNoProgressAt:    lastNoProgressAt,
		AssignedPartitions:  append([]int{}, c.assignedPartitions...),
		GenerationID:        c.generationID,
		LastAssignmentAt:    c.lastAssignmentAt,
		TimeToFirstFetch:    c.timeToFirstFetch,
		LastFetchDuration:   c.lastFetchDuration,
		MaxFetchDuration:    c.maxFetchDuration,
		LastHandlerDuration: c.lastHandlerDuration,
		MaxHandlerDuration:  c.maxHandlerDuration,
		TotalBackoff:        c.totalBackoff,
	}
}
```

Also document `RecreateCount`'s narrowed meaning by replacing its line in the `Snapshot` struct with:

```go
	// RecreateCount counts reader rebuilds. On the legacy engine each rebuild
	// is a consumer-group REJOIN (it rebalances every member of the group);
	// on the consumergroup engine it is a local partition-reader rebuild with
	// no broker-visible group effect. Do not compare the number across a
	// KAFKA_CONSUMER_ENGINE rollback.
	RecreateCount int
```

- [ ] **Step 7: Add the three debug attributes**

In `libs/atlas-kafka/consumer/debug.go`, add to `debugAttributes` after `LastNoProgressAt`:

```go
	AssignedPartitions []int     `json:"assignedPartitions"`
	GenerationID       int32     `json:"generationId"`
	LastAssignmentAt   time.Time `json:"lastAssignmentAt"`
```

and to `snapshotToAttributes`:

```go
		AssignedPartitions: s.AssignedPartitions,
		GenerationID:       s.GenerationID,
		LastAssignmentAt:   s.LastAssignmentAt,
```

- [ ] **Step 8: Run the new tests**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestSnapshotAggregates|TestSnapshotAssignedPartitions|TestOnAssignment|TestRecordersAreKeyed' -v
```
Expected: all six PASS.

- [ ] **Step 9: Run the whole module suite — this is the FR-4.4 gate**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./...
```
Expected: PASS, **including `idle_stuck_test.go`, `timing_test.go` and `debug_test.go` unchanged**. If any of them fails, the aggregation is not equivalent to the scalars for a single-entry map — fix the aggregation, do not edit the test.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-kafka/consumer/manager.go libs/atlas-kafka/consumer/engine_reader.go libs/atlas-kafka/consumer/debug.go libs/atlas-kafka/consumer/state_test.go
git commit -m "feat(atlas-kafka): key watchdog state by partition and expose assignment in Snapshot"
```

---

### Task 5: Per-partition fetch loop with in-place recovery

**Files:**
- Create: `libs/atlas-kafka/consumer/partition.go`
- Create: `libs/atlas-kafka/consumer/fakegroup_test.go`
- Test: `libs/atlas-kafka/consumer/partition_test.go`

**Interfaces:**
- Consumes: `cursor` (Task 2), `Generation`, `PartitionReaderProducer`, `partitionReaderConfig` (Task 3), `partitionStateFor`, `onReaderCreated`, `recordFetch`, `handleFetchDeadline` (Task 4), `processMessage`, `recordError`, `recordBackoff`, `recordFetchDuration`, `recordHandlerDuration`, `errFetchWedged`.
- Produces:
  - `const drainTimeout = 5 * time.Second`
  - `func (c *Consumer) runPartition(l logrus.FieldLogger, ctx context.Context, gctx context.Context, gen Generation, pa kafka.PartitionAssignment)`
  - `func (c *Consumer) runPartitionFetchLoop(l logrus.FieldLogger, pctx context.Context, partition int, rd KafkaReader, cur *cursor, wg *sync.WaitGroup, commit func(offset int64) error) error`
  - `func (c *Consumer) quiesce(l logrus.FieldLogger, wg *sync.WaitGroup, cur *cursor, commit func(offset int64) error)`
  - Test fake (in `fakegroup_test.go`): `fakeGroup`, `fakeGeneration` with `newFakeGeneration(id int32, assignments map[string][]kafka.PartitionAssignment) *fakeGeneration`, `(*fakeGeneration).commits() []committedOffset`, `(*fakeGeneration).end()`, `(*fakeGeneration).ended() bool`, `type committedOffset struct { Topic string; Partition int; Offset int64 }`.

- [ ] **Step 1: Write the fake group/generation**

Create `libs/atlas-kafka/consumer/fakegroup_test.go`:

```go
package consumer

import (
	"context"
	"errors"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// errFakeGroupClosed stands in for kafka.ErrGroupClosed.
var errFakeGroupClosed = errors.New("fake group closed")

type committedOffset struct {
	Topic     string
	Partition int
	Offset    int64
}

// fakeGeneration is a scriptable Generation. Start runs fn on a routine.Go
// with a generation-scoped context, exactly like kafka-go: the first fn to
// return ends the generation.
type fakeGeneration struct {
	id          int32
	assignments map[string][]kafka.PartitionAssignment

	mu        sync.Mutex
	committed []committedOffset
	commitErr error
	closed    bool
	done      chan struct{}
	wg        sync.WaitGroup
}

func newFakeGeneration(id int32, assignments map[string][]kafka.PartitionAssignment) *fakeGeneration {
	return &fakeGeneration{
		id:          id,
		assignments: assignments,
		done:        make(chan struct{}),
	}
}

func (g *fakeGeneration) ID() int32 { return g.id }

func (g *fakeGeneration) Assignments() map[string][]kafka.PartitionAssignment {
	return g.assignments
}

func (g *fakeGeneration) Start(fn func(ctx context.Context)) {
	l, _ := test.NewNullLogger()
	g.wg.Add(1)
	routine.Go(l, context.Background(), func(_ context.Context) {
		defer g.wg.Done()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stop := context.AfterFunc(genDoneContext(g), cancel)
		defer stop()
		fn(ctx)
		g.end()
	})
}

// genDoneContext adapts the generation's done channel to a context, mirroring
// kafka-go's genCtx (consumergroup.go:276-301).
func genDoneContext(g *fakeGeneration) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	l, _ := test.NewNullLogger()
	routine.Go(l, context.Background(), func(_ context.Context) {
		<-g.done
		cancel()
	})
	return ctx
}

// end closes the generation, mirroring kafka-go's "first Start fn to return
// ends the generation" rule.
func (g *fakeGeneration) end() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed {
		g.closed = true
		close(g.done)
	}
}

func (g *fakeGeneration) ended() bool {
	select {
	case <-g.done:
		return true
	default:
		return false
	}
}

// wait blocks until every Start'd function has exited.
func (g *fakeGeneration) wait() { g.wg.Wait() }

func (g *fakeGeneration) setCommitErr(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.commitErr = err
}

func (g *fakeGeneration) CommitOffsets(offsets map[string]map[int]int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.commitErr != nil {
		return g.commitErr
	}
	for topic, parts := range offsets {
		for partition, offset := range parts {
			g.committed = append(g.committed, committedOffset{Topic: topic, Partition: partition, Offset: offset})
		}
	}
	return nil
}

func (g *fakeGeneration) commits() []committedOffset {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]committedOffset, len(g.committed))
	copy(out, g.committed)
	return out
}

// fakeGroup hands out scripted generations in order, then blocks on ctx (a
// real ConsumerGroup.Next parks until the next rebalance).
type fakeGroup struct {
	mu     sync.Mutex
	gens   []*fakeGeneration
	next   int
	nextN  int
	closed bool
}

func newFakeGroup(gens ...*fakeGeneration) *fakeGroup {
	return &fakeGroup{gens: gens}
}

func (g *fakeGroup) Next(ctx context.Context) (Generation, error) {
	g.mu.Lock()
	g.nextN++
	if g.next > 0 {
		prev := g.gens[g.next-1]
		g.mu.Unlock()
		// Mirror kafka-go: Next never returns a new generation until the
		// previous one has ended.
		select {
		case <-prev.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		g.mu.Lock()
	}
	if g.next >= len(g.gens) {
		g.mu.Unlock()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	gen := g.gens[g.next]
	g.next++
	g.mu.Unlock()
	return gen, nil
}

// nextCalls reports how many times Next has been entered. The FR-2.3 test
// uses it to prove a wedged partition reader rebuilds WITHOUT ending the
// generation.
func (g *fakeGroup) nextCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nextN
}

func (g *fakeGroup) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	return nil
}

var (
	_ Group      = (*fakeGroup)(nil)
	_ Generation = (*fakeGeneration)(nil)
)

// silentLogger returns a logger whose entries the caller can inspect.
func silentLogger() (*logrus.Logger, *test.Hook) {
	return test.NewNullLogger()
}
```

- [ ] **Step 2: Write the failing partition tests**

Create `libs/atlas-kafka/consumer/partition_test.go`:

```go
package consumer

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// blockingReader serves a scripted sequence of messages, then blocks until
// its context is done. Each entry may instead be an error.
type scriptedPartitionReader struct {
	mu     sync.Mutex
	msgs   []kafka.Message
	errs   []error
	idx    int
	closed int
}

func (r *scriptedPartitionReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.mu.Lock()
	i := r.idx
	r.idx++
	var (
		msg kafka.Message
		err error
	)
	if i < len(r.msgs) {
		msg = r.msgs[i]
	}
	if i < len(r.errs) {
		err = r.errs[i]
	}
	past := i >= len(r.msgs) && i >= len(r.errs)
	r.mu.Unlock()

	if past {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	if err != nil {
		return kafka.Message{}, err
	}
	return msg, nil
}

func (r *scriptedPartitionReader) CommitMessages(context.Context, ...kafka.Message) error {
	return errors.New("partition readers must never commit through the reader")
}

func (r *scriptedPartitionReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed++
	return nil
}

func (r *scriptedPartitionReader) closeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

// TestRunPartitionCommitsOffsetPlusOneThroughGeneration is the end-to-end
// form of the offset contract: a delivered message must reach
// Generation.CommitOffsets as msg.Offset+1, keyed by topic and partition.
func TestRunPartitionCommitsOffsetPlusOneThroughGeneration(t *testing.T) {
	l, _ := silentLogger()
	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 41, Value: []byte("x")}}}

	c := newTestConsumer()
	c.topic = "EVENT_TOPIC_TEST"
	c.fetchTimeout = 50 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"EVENT_TOPIC_TEST": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	done := make(chan struct{})
	go func() {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	}()

	waitFor(t, func() bool { return len(gen.commits()) > 0 }, "no commit observed")
	cancel()
	<-done

	got := gen.commits()
	if got[0].Offset != 42 {
		t.Fatalf("committed offset %d, want 42 (msg.Offset+1)", got[0].Offset)
	}
	if got[0].Topic != "EVENT_TOPIC_TEST" || got[0].Partition != 0 {
		t.Fatalf("committed %+v, want topic EVENT_TOPIC_TEST partition 0", got[0])
	}
}

// TestRunPartitionRebuildsReaderWithoutEndingGeneration is FR-2.3 plus the
// design §1.1 inversion: a reader error rebuilds THAT reader in place; the
// generation must survive, because ending it would rejoin the group and
// rebalance every topic in it.
func TestRunPartitionRebuildsReaderWithoutEndingGeneration(t *testing.T) {
	l, _ := silentLogger()

	var built int
	var mu sync.Mutex
	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 50 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader {
		mu.Lock()
		built++
		n := built
		mu.Unlock()
		if n == 1 {
			return &scriptedPartitionReader{errs: []error{io.EOF}}
		}
		return &scriptedPartitionReader{}
	}

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	done := make(chan struct{})
	go func() {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	}()

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return built >= 2
	}, "reader was not rebuilt after io.EOF")

	if gen.ended() {
		t.Fatal("generation ended on a partition-reader error; recovery must stay local")
	}
	if got := c.Snapshot().RecreateCount; got < 1 {
		t.Fatalf("RecreateCount = %d, want >= 1", got)
	}

	cancel()
	<-done
}

// TestRunPartitionWedgeRebuildsReader: a reader that makes no progress for
// maxConsecutiveTimeouts ticks is still recovered (FR-2.3) — the task-136
// behaviour, now scoped to one partition reader.
func TestRunPartitionWedgeRebuildsReader(t *testing.T) {
	l, hook := silentLogger()

	var built int
	var mu sync.Mutex
	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 20 * time.Millisecond
	c.maxConsecutiveTimeouts = 2
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader {
		mu.Lock()
		built++
		mu.Unlock()
		return &scriptedPartitionReader{} // blocks → every tick is a deadline
	}

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	done := make(chan struct{})
	go func() {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	}()

	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return built >= 2
	}, "wedged reader was not rebuilt")

	if !hasLogContaining(hook, "wedged") {
		t.Fatal("no wedge warning logged for an ASSIGNED partition; FR-2.3 requires it")
	}
	if gen.ended() {
		t.Fatal("generation ended on a wedge; recovery must stay local")
	}

	cancel()
	<-done
}

// TestRunPartitionDrainCommitsCompletedWork is FR-1.6: work finished before
// the generation ends must be committed, not left for redelivery.
func TestRunPartitionDrainCommitsCompletedWork(t *testing.T) {
	l, _ := silentLogger()
	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0}, {Offset: 1}}}

	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 50 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})

	ctx := context.Background()
	gctx, gcancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	}()

	waitFor(t, func() bool { return len(gen.commits()) >= 2 }, "messages were not committed")
	gcancel()
	<-done

	last := gen.commits()
	if last[len(last)-1].Offset != 2 {
		t.Fatalf("final committed offset %d, want 2", last[len(last)-1].Offset)
	}
	if rd.closeCount() == 0 {
		t.Fatal("partition reader was not closed on generation end")
	}
}

// TestRunPartitionCommitFailureDoesNotAdvance is the FR-1.3 wiring test: a
// failing CommitOffsets must be logged and retried, never silently skipped.
func TestRunPartitionCommitFailureDoesNotAdvance(t *testing.T) {
	l, hook := silentLogger()
	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0}}}

	c := newTestConsumer()
	c.topic = "t"
	c.fetchTimeout = 20 * time.Millisecond
	c.maxInFlight = 1
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})
	gen.setCommitErr(errors.New("broker unavailable"))

	ctx, cancel := context.WithCancel(context.Background())
	gctx, gcancel := context.WithCancel(context.Background())
	defer gcancel()

	done := make(chan struct{})
	go func() {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	}()

	waitFor(t, func() bool { return hasLogContaining(hook, "commit") }, "no commit-failure warning logged")
	if len(gen.commits()) != 0 {
		t.Fatalf("commits recorded despite a failing committer: %v", gen.commits())
	}

	gen.setCommitErr(nil)
	waitFor(t, func() bool { return len(gen.commits()) > 0 }, "commit was never retried")
	if got := gen.commits()[0].Offset; got != 1 {
		t.Fatalf("retried commit offset %d, want 1", got)
	}

	cancel()
	<-done
}
```

Add these shared helpers to `libs/atlas-kafka/consumer/fakegroup_test.go`:

```go
// waitFor polls cond until it holds or 3s elapse.
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal(msg)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// hasLogContaining reports whether any captured entry's message contains sub.
func hasLogContaining(hook *test.Hook, sub string) bool {
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, sub) {
			return true
		}
	}
	return false
}
```

with `"strings"`, `"testing"` and `"time"` added to that file's imports.

Replace the three bare `go func() { … }()` launches in `partition_test.go` with `routine.Go` (goroutine guard sweeps test files):

```go
	tl, _ := silentLogger()
	done := make(chan struct{})
	routine.Go(tl, context.Background(), func(_ context.Context) {
		c.runPartition(l, ctx, gctx, gen, kafka.PartitionAssignment{ID: 0, Offset: kafka.FirstOffset})
		close(done)
	})
```

adding `routine "github.com/Chronicle20/atlas/libs/atlas-routine"` to the imports.

- [ ] **Step 3: Run the tests to verify they fail**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run TestRunPartition -v
```
Expected: FAIL — `c.runPartition undefined`.

- [ ] **Step 4: Implement `partition.go`**

```go
package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// drainTimeout bounds how long a partition loop waits for in-flight handlers
// when its generation ends. It MUST stay well below kafka-go's
// defaultRebalanceTimeout of 30s (consumergroup.go:47) — a slow drain holds
// up the whole group's rebalance, which is exactly the stall this task
// removes. 5s matches kafka-go's defaultTimeout (consumergroup.go:63).
// Handlers still running at the deadline are abandoned uncommitted and are
// redelivered in the next generation: at-least-once, erring toward
// duplication rather than loss (risks.md R1).
const drainTimeout = 5 * time.Second

// runPartition owns one assigned partition for the lifetime of one
// generation: create a positioned reader → fetch/handle/commit → on any
// recoverable error rebuild THAT READER ONLY → repeat.
//
// It must not return until the generation or the process is shutting down.
// kafka-go ends the generation as soon as any function passed to
// Generation.Start returns (consumergroup.go:387-405), so returning early
// would rejoin the group and rebalance every topic in it — the failure mode
// this task exists to remove (design §1.1).
func (c *Consumer) runPartition(l logrus.FieldLogger, ctx context.Context, gctx context.Context, gen Generation, pa kafka.PartitionAssignment) {
	pl := l.WithField("partition", pa.ID)

	// pctx is cancelled by EITHER the process context or the generation
	// context. context.AfterFunc gives us that without a goroutine of our own.
	pctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(gctx, cancel)
	defer stop()

	st := c.partitionStateFor(pa.ID)
	cur := newCursor()
	var inflight sync.WaitGroup

	commit := func(offset int64) error {
		return gen.CommitOffsets(map[string]map[int]int64{
			c.topic: {pa.ID: offset},
		})
	}

	offset := pa.Offset
	for attempt := 0; ; attempt++ {
		if pctx.Err() != nil {
			break
		}

		rd := c.prp(c.partitionReaderConfig(pa.ID), offset)
		c.onReaderCreated(pa.ID, attempt)
		if attempt == 0 {
			pl.Infof("Start consuming topic [%s] partition %d from offset %d.", c.topic, pa.ID, offset)
		} else {
			pl.Infof("Rebuilt partition reader for topic [%s] partition %d from offset %d (attempt %d).", c.topic, pa.ID, offset, attempt)
		}

		err := c.runPartitionFetchLoop(pl, pctx, pa.ID, rd, cur, &inflight, commit)
		if cerr := rd.Close(); cerr != nil {
			pl.WithError(cerr).Debugf("Error closing partition reader during rebuild.")
		}

		if pctx.Err() != nil {
			break
		}

		c.recordError(err)
		pl.WithError(err).Warnf("Partition reader for topic [%s] partition %d exited; rebuilding after backoff.", c.topic, pa.ID)

		// Let in-flight handlers finish and commit what they can before
		// choosing a resume offset, so the rebuilt reader re-reads only
		// genuinely uncommitted messages.
		c.quiesce(pl, &inflight, cur, commit)
		offset = cur.resumeOffset(pa.Offset)
		cur.reset()

		wait := st.backoff.next()
		select {
		case <-pctx.Done():
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}

	c.quiesce(pl, &inflight, cur, commit)
	pl.Debugf("Partition loop for topic [%s] partition %d stopped.", c.topic, pa.ID)
}

// runPartitionFetchLoop is the task-136 fetch loop rebased onto a single
// partition: the per-call fetchTimeout is still a liveness tick, an expired
// deadline on a reader that is still making fetch attempts is still a healthy
// idle tick, and maxConsecutiveTimeouts no-progress ticks still return
// errFetchWedged. The only substitutions are that offsets commit through the
// generation-scoped cursor rather than reader.CommitMessages, and that the
// loop's cancel signal is pctx (process OR generation).
func (c *Consumer) runPartitionFetchLoop(l logrus.FieldLogger, pctx context.Context, partition int, rd KafkaReader, cur *cursor, wg *sync.WaitGroup, commit func(offset int64) error) error {
	maxQueue := 4 * c.maxInFlight
	var sem chan struct{}
	if c.maxInFlight > 1 {
		sem = make(chan struct{}, c.maxInFlight)
	}

	for {
		if pctx.Err() != nil {
			return pctx.Err()
		}

		// Back-pressure: stop fetching when the queue is full (the head is
		// stuck on a failure). Wait one tick, then try to advance.
		if c.maxInFlight > 1 && cur.len() >= maxQueue {
			select {
			case <-pctx.Done():
				return pctx.Err()
			case <-time.After(c.fetchTimeout):
			}
			c.tryAdvance(l, cur, commit)
			continue
		}

		fetchCtx, cancelFetch := context.WithTimeout(pctx, c.fetchTimeout)
		fetchStart := time.Now()
		msg, err := rd.FetchMessage(fetchCtx)
		cancelFetch()
		c.recordFetchDuration(time.Since(fetchStart))

		if err != nil {
			if pctx.Err() != nil || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, rd, partition); werr != nil {
					return werr
				}
				c.tryAdvance(l, cur, commit)
				continue
			}
			return err
		}

		c.recordFetch(partition)
		l.Debugf("Message received %s.", string(msg.Value))

		in := cur.track(msg.Offset)

		if c.maxInFlight <= 1 {
			handlerStart := time.Now()
			ok := c.processMessage(l, pctx, msg)
			c.recordHandlerDuration(time.Since(handlerStart))
			in.mark(ok)
			c.tryAdvance(l, cur, commit)
			continue
		}

		sem <- struct{}{}
		m := msg
		tracked := in
		wg.Add(1)
		routine.Go(l, pctx, func(hctx context.Context) {
			defer wg.Done()
			defer func() { <-sem }()
			handlerStart := time.Now()
			ok := c.processMessage(l, hctx, m)
			c.recordHandlerDuration(time.Since(handlerStart))
			tracked.mark(ok)
			c.tryAdvance(l, cur, commit)
		})
	}
}

// tryAdvance advances the commit cursor, logging a commit failure at Warn.
// The cursor retains its high-water mark on failure, so the next call retries
// the same offset (FR-1.3) — the message is never silently skipped.
func (c *Consumer) tryAdvance(l logrus.FieldLogger, cur *cursor, commit func(offset int64) error) {
	if err := cur.advance(commit); err != nil {
		l.WithError(err).Warnf("Could not commit message offset for topic [%s]; it may be redelivered.", c.topic)
	}
}

// quiesce waits up to drainTimeout for in-flight handlers, then makes a final
// commit attempt (FR-1.6). Handlers still running at the deadline are
// abandoned uncommitted and redelivered.
func (c *Consumer) quiesce(l logrus.FieldLogger, wg *sync.WaitGroup, cur *cursor, commit func(offset int64) error) {
	done := make(chan struct{})
	routine.Go(l, context.Background(), func(_ context.Context) {
		wg.Wait()
		close(done)
	})

	select {
	case <-done:
	case <-time.After(drainTimeout):
		l.Warnf("Drain timeout (%v) elapsed on topic [%s]; abandoning in-flight handlers uncommitted for redelivery.", drainTimeout, c.topic)
	}

	c.tryAdvance(l, cur, commit)
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run TestRunPartition -v
```
Expected: all five `TestRunPartition*` tests PASS.

- [ ] **Step 6: Run the whole module suite**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./...
```
Expected: PASS.

- [ ] **Step 7: Check the goroutine guard now, not at the end**

```bash
cd ../.. && tools/goroutine-guard.sh
```
Expected: exit 0. If it flags a bare `go` in a test file, convert it to `routine.Go`.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-kafka/consumer/partition.go libs/atlas-kafka/consumer/partition_test.go libs/atlas-kafka/consumer/fakegroup_test.go
git commit -m "feat(atlas-kafka): add per-partition fetch loop with in-place reader recovery"
```

---

### Task 6: Generation loop and assignment-aware liveness

This is the task that satisfies the PRD's core requirement: a consumer holding zero partitions is `healthy-idle`, never ticks, never warns, never recreates.

**Files:**
- Create: `libs/atlas-kafka/consumer/engine_group.go`
- Test: `libs/atlas-kafka/consumer/engine_group_test.go`

**Interfaces:**
- Consumes: `Group`, `Generation`, `GroupProducer`, `groupConfig` (Task 3), `onAssignment` (Task 4), `runPartition` (Task 5), `fetchBackoff`, `recordError`, `recordBackoff`.
- Produces:
  - `func (c *Consumer) startGroupEngine(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup)`
  - `func (c *Consumer) runGenerations(l logrus.FieldLogger, ctx context.Context, grp Group) error`
  - `func partitionIDs(pas []kafka.PartitionAssignment) []int`

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-kafka/consumer/engine_group_test.go`:

```go
package consumer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// newGroupConsumer builds a Consumer wired to a scripted group. It is the
// unit-test equivalent of AddConsumer on the consumergroup engine.
func newGroupConsumer(topic string, grp Group) *Consumer {
	c := newTestConsumer()
	c.topic = topic
	c.groupId = "Test Service"
	c.brokers = []string{"broker:9092"}
	c.fetchTimeout = 20 * time.Millisecond
	c.maxConsecutiveTimeouts = 2
	c.maxInFlight = 1
	c.gp = func(kafka.ConsumerGroupConfig) (Group, error) { return grp, nil }
	return c
}

// TestZeroAssignmentIsHealthyIdle is THE acceptance test for FR-2.1/2.2/2.5:
// with 237 single-partition topics and replicas: 2, one member of every group
// is always unassigned. It must not tick, must not warn, and must not
// recreate — the recreate is what rejoins the group and stalls gameplay on
// every other topic in it.
func TestZeroAssignmentIsHealthyIdle(t *testing.T) {
	l, hook := silentLogger()

	gen := newFakeGeneration(4, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return c.Snapshot().GenerationID == 4 }, "generation was never observed")

	// Park long enough that the legacy watchdog would have wedged twice over
	// (maxConsecutiveTimeouts=2 × fetchTimeout=20ms).
	time.Sleep(300 * time.Millisecond)

	s := c.Snapshot()
	if s.RecreateCount != 0 {
		t.Fatalf("RecreateCount = %d while unassigned, want 0", s.RecreateCount)
	}
	if s.NoProgressTicks != 0 {
		t.Fatalf("NoProgressTicks = %d while unassigned, want 0", s.NoProgressTicks)
	}
	if s.ConsecutiveTimeouts != 0 {
		t.Fatalf("ConsecutiveTimeouts = %d while unassigned, want 0", s.ConsecutiveTimeouts)
	}
	if len(s.AssignedPartitions) != 0 {
		t.Fatalf("AssignedPartitions = %v while unassigned, want empty", s.AssignedPartitions)
	}
	if hasLogContaining(hook, "wedged") {
		t.Fatal("logged a wedge warning while unassigned; FR-2.5 forbids it")
	}
	if hasLogContaining(hook, "stall suspect") {
		t.Fatal("logged a stall-suspect warning while unassigned; FR-2.5 forbids it")
	}

	// FR-2.1's "parks rather than spins": exactly one Next call is
	// outstanding, blocked on the generation that never ends.
	if n := grp.nextCalls(); n > 2 {
		t.Fatalf("Next called %d times while parked; the zero-assignment branch is spinning", n)
	}

	cancel()
	wg.Wait()
}

// TestAssignedGenerationStartsOnePartitionLoop covers FR-1.2 and FR-4.1/4.2.
func TestAssignedGenerationStartsOnePartitionLoop(t *testing.T) {
	l, _ := silentLogger()

	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0, Value: []byte("m")}}}
	gen := newFakeGeneration(9, map[string][]kafka.PartitionAssignment{
		"EVENT_TOPIC_TEST": {{ID: 0, Offset: kafka.FirstOffset}},
	})
	grp := newFakeGroup(gen)
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return len(gen.commits()) > 0 }, "assigned partition never delivered")

	s := c.Snapshot()
	if s.GenerationID != 9 {
		t.Fatalf("GenerationID = %d, want 9", s.GenerationID)
	}
	if len(s.AssignedPartitions) != 1 || s.AssignedPartitions[0] != 0 {
		t.Fatalf("AssignedPartitions = %v, want [0]", s.AssignedPartitions)
	}
	if s.LastAssignmentAt.IsZero() {
		t.Fatal("LastAssignmentAt is zero after an assignment")
	}

	cancel()
	wg.Wait()
}

// TestUnassignedToAssignedResetsNoProgressState is FR-2.4 end to end.
func TestUnassignedToAssignedResetsNoProgressState(t *testing.T) {
	l, _ := silentLogger()

	idle := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{})
	active := newFakeGeneration(2, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})
	grp := newFakeGroup(idle, active)
	c := newGroupConsumer("t", grp)
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return &scriptedPartitionReader{} }

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return c.Snapshot().GenerationID == 1 }, "first generation not observed")
	idle.end() // rebalance: the member picks up partition 0

	waitFor(t, func() bool { return c.Snapshot().GenerationID == 2 }, "second generation not observed")
	if got := c.partitionStateFor(0).consecutiveTimeouts; got != 0 {
		t.Fatalf("consecutiveTimeouts = %d on newly assigned partition, want 0", got)
	}

	cancel()
	wg.Wait()
}

// TestGroupProducerErrorBacksOffAndRetries: a failure to join must not kill
// the consumer goroutine.
func TestGroupProducerErrorBacksOffAndRetries(t *testing.T) {
	l, _ := silentLogger()

	var mu sync.Mutex
	var calls int
	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)

	c := newGroupConsumer("t", grp)
	c.gp = func(kafka.ConsumerGroupConfig) (Group, error) {
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			return nil, context.DeadlineExceeded
		}
		return grp, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return c.Snapshot().GenerationID == 1 }, "group was never retried after a join failure")

	cancel()
	wg.Wait()
}

// TestPartitionIDsSorted pins the helper's output ordering, which
// Snapshot.AssignedPartitions and the assignment log line both rely on.
func TestPartitionIDsSorted(t *testing.T) {
	got := partitionIDs([]kafka.PartitionAssignment{{ID: 3}, {ID: 1}, {ID: 2}})
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("partitionIDs = %v, want [1 2 3]", got)
	}
	if partitionIDs(nil) == nil {
		t.Fatal("partitionIDs(nil) is nil, want an empty slice")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestZeroAssignment|TestAssignedGeneration|TestUnassignedToAssigned|TestGroupProducerError|TestPartitionIDs' -v
```
Expected: FAIL — `c.startGroupEngine undefined`, `undefined: partitionIDs`.

- [ ] **Step 3: Implement `engine_group.go`**

```go
package consumer

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// startGroupEngine owns the consumer-group lifecycle: join → consume
// generations → on failure, back off and rejoin. Only a cancelled parent
// context means shutdown.
func (c *Consumer) startGroupEngine(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	l.Infof("Creating topic consumer (engine consumergroup).")

	backoff := newFetchBackoff()
	for {
		if ctx.Err() != nil {
			l.Infof("Parent context canceled; shutting down topic consumer.")
			return
		}

		grp, err := c.gp(c.groupConfig())
		if err == nil {
			err = c.runGenerations(l, ctx, grp)
			if cerr := grp.Close(); cerr != nil {
				l.WithError(cerr).Debugf("Error closing consumer group.")
			}
		}

		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			l.Infof("Topic consumer stopped.")
			return
		}

		c.recordError(err)
		l.WithError(err).Errorf("Consumer group for topic [%s] exited; rejoining after backoff.", c.topic)
		wait := backoff.next()
		select {
		case <-ctx.Done():
			l.Infof("Topic consumer stopped during backoff.")
			return
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}
}

// runGenerations consumes generations until the group closes or ctx is done.
//
// Next blocks until the current generation ends (kafka-go consumergroup.go:701
// receives on the unbuffered cg.next, which is fed only after <-gen.done in
// nextGeneration), so the zero-assignment branch below PARKS — it does not
// spin.
func (c *Consumer) runGenerations(l logrus.FieldLogger, ctx context.Context, grp Group) error {
	for {
		gen, err := grp.Next(ctx)
		if err != nil {
			return err
		}

		parts := gen.Assignments()[c.topic]
		ids := partitionIDs(parts)
		c.onAssignment(gen.ID(), ids)

		if len(parts) == 0 {
			// HEALTHY-IDLE (FR-2.1/2.2/2.3/2.5). Every atlas-main topic has a
			// single partition while services run replicas: 2, so exactly one
			// member of every group lands here permanently. It holds nothing,
			// so it must not tick, must not warn, and must not recreate — the
			// recreate is what rejoins the group and rebalances every OTHER
			// topic in it. Heartbeats are generation-scoped and started
			// unconditionally by kafka-go (consumergroup.go:840), so doing
			// nothing here keeps this member alive and eligible for the next
			// assignment. Debug, not Warn: this state is expected (FR-4.3).
			l.Debugf("Consumer for topic [%s] (group [%s]) holds no partition assignment in generation %d; healthy-idle.",
				c.topic, c.groupId, gen.ID())
			continue
		}

		l.Infof("Consumer for topic [%s] (group [%s]) assigned partitions %v in generation %d.",
			c.topic, c.groupId, ids, gen.ID())

		for _, pa := range parts {
			assignment := pa
			gen.Start(func(gctx context.Context) {
				c.runPartition(l, ctx, gctx, gen, assignment)
			})
		}
	}
}

// partitionIDs extracts a sorted partition-id list. Always non-nil so
// Snapshot.AssignedPartitions serialises as [] rather than null.
func partitionIDs(pas []kafka.PartitionAssignment) []int {
	out := make([]int, 0, len(pas))
	for _, pa := range pas {
		out = append(out, pa.ID)
	}
	sort.Ints(out)
	return out
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestZeroAssignment|TestAssignedGeneration|TestUnassignedToAssigned|TestGroupProducerError|TestPartitionIDs' -v
```
Expected: all five PASS.

- [ ] **Step 5: Run the whole module suite**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/consumer/engine_group.go libs/atlas-kafka/consumer/engine_group_test.go
git commit -m "feat(atlas-kafka): add assignment-aware generation loop"
```

---

### Task 7: Engine selection, default flip, and rollback switch

**Files:**
- Create: `libs/atlas-kafka/consumer/engine.go`
- Create: `libs/atlas-kafka/consumer/engine_test.go`
- Modify: `libs/atlas-kafka/consumer/manager.go` (`Manager.engine`, `Consumer.engine`, `Snapshot.Engine`)
- Modify: `libs/atlas-kafka/consumer/debug.go` (`engine` attribute)
- Modify: `libs/atlas-kafka/consumer/engine_reader.go` (rename `start` → `startReaderEngine`)
- Modify: `libs/atlas-kafka/consumer/manager_test.go`, `idle_stuck_test.go`, `timing_test.go`, `debug_test.go` (pin to the legacy engine)

**Interfaces:**
- Consumes: everything from Tasks 1–6.
- Produces:
  - `type EngineName string`, `const EngineReader EngineName = "reader"`, `const EngineConsumerGroup EngineName = "consumergroup"`
  - `const engineEnvVar = "KAFKA_CONSUMER_ENGINE"`
  - `func ConfigEngine(e EngineName) ManagerConfig`
  - `func resolveEngine(l logrus.FieldLogger) EngineName`
  - `func (c *Consumer) start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup)` — now the dispatcher.
  - `Snapshot.Engine string`

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-kafka/consumer/engine_test.go`:

```go
package consumer

import (
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// TestResolveEngineDefaultsToConsumerGroup: unset means the new engine
// (FR-5.1). The default is what ships; `reader` is the rollback.
func TestResolveEngineDefaultsToConsumerGroup(t *testing.T) {
	l, _ := test.NewNullLogger()
	t.Setenv(engineEnvVar, "")
	if got := resolveEngine(l); got != EngineConsumerGroup {
		t.Fatalf("resolveEngine = %q, want %q", got, EngineConsumerGroup)
	}
}

// TestResolveEngineHonoursReader is the rollback path: one env var, one pod
// restart, no state migration (FR-5.2).
func TestResolveEngineHonoursReader(t *testing.T) {
	l, _ := test.NewNullLogger()
	t.Setenv(engineEnvVar, "reader")
	if got := resolveEngine(l); got != EngineReader {
		t.Fatalf("resolveEngine = %q, want %q", got, EngineReader)
	}
}

// TestResolveEngineFailsSoftOnGarbage: a typo in a deployment env var must
// not take a service's consumers offline. Warn and use the default.
func TestResolveEngineFailsSoftOnGarbage(t *testing.T) {
	l, hook := test.NewNullLogger()
	t.Setenv(engineEnvVar, "readr")
	if got := resolveEngine(l); got != EngineConsumerGroup {
		t.Fatalf("resolveEngine = %q, want the default %q", got, EngineConsumerGroup)
	}
	found := false
	for _, e := range hook.AllEntries() {
		if e.Level.String() == "warning" {
			found = true
		}
	}
	if !found {
		t.Fatal("no warning logged for an unrecognised engine value")
	}
}

// TestConfigEngineOverridesEnv: the explicit configurator wins, which is how
// the legacy-engine tests pin themselves without racing on process env.
func TestConfigEngineOverridesEnv(t *testing.T) {
	ResetInstance()
	t.Setenv(engineEnvVar, "consumergroup")
	m := GetManager(ConfigEngine(EngineReader))
	if m.engine != EngineReader {
		t.Fatalf("Manager.engine = %q, want %q", m.engine, EngineReader)
	}
	ResetInstance()
}

// TestSnapshotReportsEngine: during a staged rollout both engines run in one
// cluster, so "which engine is this pod on?" must be answerable from the
// debug route rather than from the pod's env (design §7).
func TestSnapshotReportsEngine(t *testing.T) {
	c := newTestConsumer()
	c.engine = EngineConsumerGroup
	if got := c.Snapshot().Engine; got != "consumergroup" {
		t.Fatalf("Snapshot.Engine = %q, want consumergroup", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestResolveEngine|TestConfigEngine|TestSnapshotReportsEngine' -v
```
Expected: FAIL — `undefined: resolveEngine`, `undefined: EngineReader`.

- [ ] **Step 3: Create `engine.go`**

```go
package consumer

import (
	"context"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// EngineName selects the consumer implementation.
type EngineName string

const (
	// EngineConsumerGroup is the supported engine: kafka.ConsumerGroup +
	// Generation, so partition assignment is directly observable and a
	// stalled partition is recovered in place instead of by rejoining the
	// group (task-209).
	EngineConsumerGroup EngineName = "consumergroup"

	// EngineReader is the legacy *kafka.Reader path, retained for one
	// release as the rollback target. Both engines use the same group IDs
	// and commit the same offset value (msg.Offset+1), so switching is a pod
	// restart with no topic, offset or group-state migration (FR-5.2/5.3).
	EngineReader EngineName = "reader"
)

// engineEnvVar selects the engine at process start. Unset or empty means
// EngineConsumerGroup; an unrecognised value warns and falls back to the
// default, because a typo in a deployment env var must not take a service's
// consumers offline.
const engineEnvVar = "KAFKA_CONSUMER_ENGINE"

// ConfigEngine pins the engine explicitly, overriding engineEnvVar. It exists
// so tests can select an engine without mutating process env, and so an
// embedder can hard-select one.
//
//goland:noinspection GoUnusedExportedFunction
func ConfigEngine(e EngineName) ManagerConfig {
	return func(m *Manager) {
		m.engine = e
	}
}

func resolveEngine(l logrus.FieldLogger) EngineName {
	v := os.Getenv(engineEnvVar)
	switch EngineName(v) {
	case "":
		return EngineConsumerGroup
	case EngineConsumerGroup:
		return EngineConsumerGroup
	case EngineReader:
		return EngineReader
	default:
		l.Warnf("Unrecognised %s value [%s]; using [%s].", engineEnvVar, v, EngineConsumerGroup)
		return EngineConsumerGroup
	}
}

// start dispatches to the configured engine. It is the single entry point
// AddConsumer launches per consumer.
func (c *Consumer) start(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) {
	if c.engine == EngineReader {
		c.startReaderEngine(l, ctx, wg)
		return
	}
	c.startGroupEngine(l, ctx, wg)
}
```

- [ ] **Step 4: Rename the legacy entry point and wire the engine through**

In `engine_reader.go`, rename `func (c *Consumer) start(` to `func (c *Consumer) startReaderEngine(` and update its doc comment's first word accordingly (`startReaderEngine owns the full reader lifecycle: …`).

In `manager.go`:
- add `engine EngineName` to the `Manager` struct;
- inside `GetManager`'s `once.Do`, resolve the default **before** the configurator loop so `ConfigEngine` can override it. `GetManager` has no logger, so resolve with a package-level logger:

```go
			engine: resolveEngine(logrus.StandardLogger()),
```

- add `engine EngineName` to the `Consumer` struct and set `engine: m.engine` in `AddConsumer`'s `con := &Consumer{...}` literal;
- add `Engine string` to the `Snapshot` struct (after `LastAssignmentAt`) with the doc comment:

```go
	// Engine is the consumer implementation this consumer is running:
	// "consumergroup" or "reader". During a staged rollout both run in one
	// cluster, so this is how an operator tells them apart.
	Engine string
```

- and set it in `Snapshot()`: `Engine: string(c.engine),`.

In `AddConsumer`, log the engine once per consumer so it is visible in service startup logs:

```go
		l := cl.WithFields(logrus.Fields{"originator": c.topic, "type": "kafka_consumer", "engine": string(con.engine)})
```

(replacing the existing `l := cl.WithFields(...)` line at `manager.go:189`).

In `debug.go`, add `Engine string \`json:"engine"\`` to `debugAttributes` and `Engine: s.Engine,` to `snapshotToAttributes`.

- [ ] **Step 5: Pin the legacy-path tests to the legacy engine**

Every existing test that injects a whole-loop mock through `ConfigReaderProducer` exercises the legacy engine and must say so, now that the default has flipped. In each of `manager_test.go`, `idle_stuck_test.go`, `timing_test.go` and `debug_test.go`, add `consumer.ConfigEngine(consumer.EngineReader)` to every `consumer.GetManager(...)` call. For example, in `manager_test.go`:

```go
	cm := consumer.GetManager(
		consumer.ConfigEngine(consumer.EngineReader),
		consumer.ConfigReaderProducer(func(kafka.ReaderConfig) consumer.KafkaReader { return mock }),
	)
```

Find every call site with:

```bash
cd libs/atlas-kafka && grep -n "GetManager(" consumer/*_test.go
```

`dwell_integration_test.go`'s call (inside `dwellSetup`) is handled in Task 8, not here — it is behind the `integration` build tag and needs the engine matrix rather than a fixed pin.

Change **only** the `GetManager` call — no assertion, mock or timing in those files may be edited. If an assertion fails after the pin, the legacy engine has been altered and the fix belongs in `engine_reader.go`, not in the test.

- [ ] **Step 6: Run the engine tests**

```bash
cd libs/atlas-kafka && go test -race ./consumer/ -run 'TestResolveEngine|TestConfigEngine|TestSnapshotReportsEngine' -v
```
Expected: all five PASS.

- [ ] **Step 7: Run the whole module suite**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./...
```
Expected: PASS — every pre-existing test green with only its `GetManager` line changed.

- [ ] **Step 8: Prove FR-3.2 mechanically**

```bash
cd ../.. && git diff --stat services/
```
Expected: empty output.

- [ ] **Step 9: Commit**

```bash
git add libs/atlas-kafka/consumer/
git commit -m "feat(atlas-kafka): select consumer engine via KAFKA_CONSUMER_ENGINE, default consumergroup"
```

---

### Task 8: Integration scenarios — S6 and the cross-engine offset round-trip

Unit tests cannot prove rebalance behaviour or offset compatibility against a real broker. These two scenarios are the evidence the PRD's acceptance criteria name.

**Files:**
- Modify: `libs/atlas-kafka/consumer/dwell_integration_test.go`

**Interfaces:**
- Consumes: existing helpers in the same file — `startDwellKafka(t) []string`, `createDwellTopics(t, brokers, topics)`, `publishStamped(t, brokers, topic, n, interval)`, `dwellSetup(t, brokers, idleCount, otherCount, rp []consumer.ManagerConfig, idleDecorators ...model.Decorator[consumer.Config]) (*consumer.Manager, *latencyRecorder, context.CancelFunc, *sync.WaitGroup)`, `totalRecreates(cm)`, `dumpSnapshots(t, cm)`, `snapshotForTopic(t, cm, topic)`, `forceErrReader`; plus `consumer.ConfigEngine`, `consumer.ConfigPartitionReaderProducer`, `consumer.EngineReader`, `consumer.EngineConsumerGroup`.
- Produces: `dwellEngines`, `TestDwellS6_MembersExceedPartitions`, `TestCrossEngineOffsetRoundTrip`.

- [ ] **Step 1: Run S1–S4 against both engines**

`dwellSetup` already forwards its `rp []consumer.ManagerConfig` argument straight to `consumer.GetManager` (`dwell_integration_test.go:185`), so the engine is selected by adding one `ManagerConfig` — no other edit to the helper. Add near the top of the file:

```go
// dwellEngines is the engine matrix the dwell scenarios run against. The
// legacy engine is retained for one release (FR-5.1), so its coverage must
// not lapse while it is still a supported rollback target.
var dwellEngines = []consumer.EngineName{consumer.EngineConsumerGroup, consumer.EngineReader}
```

Wrap the bodies of `TestDwellS1_SteadyStateLatency`, `TestDwellS2_IdleTickChurn`, `TestDwellS3_ForcedRecreateBounded` and `TestDwellS4_TickControl` in:

```go
	for _, engine := range dwellEngines {
		engine := engine
		t.Run(string(engine), func(t *testing.T) {
			// ... existing body ...
		})
	}
```

and thread the engine into each one's `dwellSetup` call:

- S1 (`dwell_integration_test.go:217`): `dwellSetup(t, brokers, 15, 4, nil)` → `dwellSetup(t, brokers, 15, 4, []consumer.ManagerConfig{consumer.ConfigEngine(engine)})`
- S2 (`:239`) and S4 (`:360`): same substitution for their `nil` `rp` argument, decorators unchanged.
- S3 (`:324-331`): its `forceErrReader` is injected through `ConfigReaderProducer`, which only the legacy engine consults — the group engine builds partition readers through `prp`. Supply **both** so the scenario is engine-agnostic:

```go
			var arm atomic.Bool
			wrap := func(cfg kafka.ReaderConfig, inner consumer.KafkaReader) consumer.KafkaReader {
				if cfg.Topic == dwellActiveTopic {
					return &forceErrReader{inner: inner, arm: &arm}
				}
				return inner
			}
			cfgs := []consumer.ManagerConfig{
				consumer.ConfigEngine(engine),
				consumer.ConfigReaderProducer(func(cfg kafka.ReaderConfig) consumer.KafkaReader {
					return wrap(cfg, kafka.NewReader(cfg))
				}),
				consumer.ConfigPartitionReaderProducer(func(cfg kafka.ReaderConfig, offset int64) consumer.KafkaReader {
					r := kafka.NewReader(cfg)
					_ = r.SetOffset(offset)
					return wrap(cfg, r)
				}),
			}
			cm, rec, cancel, wg := dwellSetup(t, brokers, 5, 0, cfgs)
```

S5 (`TestDwellS5_MaxWaitIdleFetchRate`) drives raw `kafka.Reader`s directly and never touches the `Manager`, so it is engine-agnostic — leave it unchanged.

**Do not change any existing assertion or threshold.** S1 asserts `p99 < 1s` (`dwell_integration_test.go:228`), not the 22 ms figure; 22.0 ms / 87.1 ms are the *measured* task-136 S1 numbers the PRD says the new engine must meet or beat. Compare against them in `verification.md` (Task 9); tightening the in-test bound is a separate decision, not part of this migration.

- [ ] **Step 2: Write S6**

Append to `libs/atlas-kafka/consumer/dwell_integration_test.go`:

```go
// TestDwellS6_MembersExceedPartitions reproduces the exact atlas-main
// topology behind the incident: a single-partition topic with TWO group
// members (services run replicas: 2 against 237 single-partition topics), so
// one member is permanently unassigned. On the legacy engine that member
// recreates itself every maxConsecutiveTimeouts×fetchTimeout, and each
// recreate rejoins the group and stalls every other topic in it. On the
// consumergroup engine it must sit healthy-idle: zero recreates across the
// whole window, with the assigned member still delivering.
func TestDwellS6_MembersExceedPartitions(t *testing.T) {
	brokers := startDwellKafka(t)
	const topic = "dwell.s6.single"
	const groupID = "S6 Service"
	createDwellTopics(t, brokers, []string{topic})

	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Compressed ticks: a full wedge cycle is 2×200ms, so a 6s window covers
	// ~15 cycles — well past the point the legacy engine would have recreated.
	const fetchTimeout = 200 * time.Millisecond
	const maxConsecutive = 2

	var delivered atomic.Int64

	// Two independent Managers stand in for two pods in the same group.
	// ResetInstance between them is what makes GetManager hand back a second,
	// separate Manager rather than the memoized singleton.
	managers := make([]*consumer.Manager, 0, 2)
	wgs := make([]*sync.WaitGroup, 0, 2)
	for i := 0; i < 2; i++ {
		consumer.ResetInstance()
		cm := consumer.GetManager(consumer.ConfigEngine(consumer.EngineConsumerGroup))
		wg := &sync.WaitGroup{}
		// AddConsumerAndRegister takes no decorators; use AddConsumer (which
		// does) then RegisterHandler, the same shape dwellSetup uses.
		cm.AddConsumer(l, ctx, wg)(
			consumer.NewConfig(brokers, fmt.Sprintf("s6-member-%d", i), topic, groupID),
			consumer.SetFetchTimeout(fetchTimeout),
			consumer.SetMaxConsecutiveTimeouts(maxConsecutive),
			// maxWait must stay well under fetchTimeout so an ASSIGNED idle
			// reader completes >=1 fetch long-poll per deadline tick.
			consumer.SetMaxWait(50*time.Millisecond),
		)
		_, err := cm.RegisterHandler(topic, func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
			delivered.Add(1)
			return true, nil
		})
		require.NoError(t, err)
		managers = append(managers, cm)
		wgs = append(wgs, wg)
	}
	defer func() {
		cancel()
		for _, wg := range wgs {
			wg.Wait()
		}
	}()

	// Let both members join and the balancer settle.
	time.Sleep(3 * time.Second)

	publishStamped(t, brokers, topic, 5, 100*time.Millisecond)
	require.Eventually(t, func() bool { return delivered.Load() >= 5 },
		20*time.Second, 100*time.Millisecond,
		"the assigned member stopped delivering")

	// Dwell past several would-be wedge cycles.
	time.Sleep(6 * time.Second)

	totalRecreates := 0
	unassignedSeen := false
	for _, cm := range managers {
		for _, c := range cm.Consumers() {
			s := c.Snapshot()
			totalRecreates += s.RecreateCount
			if len(s.AssignedPartitions) == 0 {
				unassignedSeen = true
			}
		}
	}

	require.True(t, unassignedSeen,
		"no member ended up unassigned; the scenario did not reproduce members > partitions")
	require.Zero(t, totalRecreates,
		"an unassigned member recreated itself — this is the rebalance churn task-209 removes")

	for _, e := range hook.AllEntries() {
		require.NotContains(t, e.Message, "wedged",
			"an unassigned member logged a wedge warning (FR-2.5)")
		require.NotContains(t, e.Message, "stall suspect",
			"an unassigned member logged a stall-suspect warning (FR-2.5)")
	}
}
```

S6 runs on the `consumergroup` engine only — on `reader` it is a reproduction of the bug, not a test of the fix, and would fail by design.

- [ ] **Step 3: Write the cross-engine offset round-trip**

This is the executable form of FR-5.3 — the single test that proves a rollback neither replays nor loses.

```go
// TestCrossEngineOffsetRoundTrip is FR-5.3 made executable: consume half the
// messages on one engine, stop, resume the same group on the other engine,
// and assert the second run sees exactly the remainder — no replay, no gap.
// Run in both directions, because a rollback must be safe both ways.
func TestCrossEngineOffsetRoundTrip(t *testing.T) {
	directions := []struct {
		name   string
		first  consumer.EngineName
		second consumer.EngineName
	}{
		{"consumergroup-then-reader", consumer.EngineConsumerGroup, consumer.EngineReader},
		{"reader-then-consumergroup", consumer.EngineReader, consumer.EngineConsumerGroup},
	}

	for _, d := range directions {
		d := d
		t.Run(d.name, func(t *testing.T) {
			brokers := startDwellKafka(t)
			topic := "dwell.roundtrip." + d.name
			groupID := "RoundTrip " + d.name
			createDwellTopics(t, brokers, []string{topic})
			publishDwellMessages(t, brokers, topic, 10)

			l, _ := test.NewNullLogger()

			run := func(engine consumer.EngineName, want int) []string {
				var mu sync.Mutex
				var seen []string
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				consumer.ResetInstance()
				cm := consumer.GetManager(consumer.ConfigEngine(engine))
				wg := &sync.WaitGroup{}
				_, err := cm.AddConsumerAndRegister(l, ctx, wg)(
					consumer.NewConfig(brokers, "roundtrip", topic, groupID),
					func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
						mu.Lock()
						defer mu.Unlock()
						if len(seen) < want {
							seen = append(seen, string(msg.Value))
						}
						return true, nil
					},
				)
				require.NoError(t, err)

				require.Eventually(t, func() bool {
					mu.Lock()
					defer mu.Unlock()
					return len(seen) >= want
				}, 30*time.Second, 100*time.Millisecond, "engine %s did not deliver %d messages", engine, want)

				// Give the commit for the last handled message time to land
				// before tearing the consumer down.
				time.Sleep(2 * time.Second)
				cancel()
				wg.Wait()

				mu.Lock()
				defer mu.Unlock()
				return append([]string(nil), seen...)
			}

			firstHalf := run(d.first, 5)
			secondHalf := run(d.second, 5)

			require.Len(t, firstHalf, 5)
			require.Len(t, secondHalf, 5)
			for _, v := range secondHalf {
				require.NotContains(t, firstHalf, v,
					"engine %s replayed a message engine %s already committed", d.second, d.first)
			}
		})
	}
}
```

- [ ] **Step 4: Run the integration suite**

```bash
cd libs/atlas-kafka && go test -race -tags integration -timeout 30m ./consumer/... -v 2>&1 | tail -60
```
Expected: `TestDwellS1_…` through `TestDwellS4_…` PASS for **both** engine subtests, `TestDwellS5_…` PASS (engine-agnostic), `TestDwellS6_MembersExceedPartitions` PASS with `totalRecreates == 0`, and `TestCrossEngineOffsetRoundTrip` PASS in both directions.

If S6 fails with "no member ended up unassigned", the two `Manager`s joined as a single member — check that `ResetInstance()` ran between them and that each got its own `AddConsumerAndRegister`. If it fails with a non-zero recreate count, that is a real defect in the zero-assignment branch, not a flaky test.

- [ ] **Step 5: Record the S1 latency numbers**

Each dwell scenario logs its own numbers (`t.Logf("S1: p99=%v max=%v …")`). Capture S1's p99 and max **per engine** from the run output — they go into `verification.md` in Task 9, where they are compared against the task-136 S1 measurements the PRD's NFR names: p99 22.0 ms, max 87.1 ms. If the `consumergroup` engine is materially worse, that is risks.md R6 and a merge blocker; report the numbers rather than re-running until one looks good.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/consumer/dwell_integration_test.go
git commit -m "test(atlas-kafka): add S6 members-exceed-partitions and cross-engine offset round-trip"
```

---

### Task 9: Documentation and the full branch verification gate

**Files:**
- Modify: `libs/atlas-kafka/README.md`
- Create: `docs/tasks/task-209-assignment-aware-watchdog/verification.md`

**Interfaces:**
- Consumes: everything above.
- Produces: the recorded evidence the branch is done.

- [ ] **Step 1: Document the engine switch in the lib README**

Append to `libs/atlas-kafka/README.md`:

```markdown
## Consumer engines

`consumer` ships two implementations, selected at process start by
`KAFKA_CONSUMER_ENGINE`:

| Value | Engine | Notes |
|---|---|---|
| unset / `consumergroup` | `kafka.ConsumerGroup` + `Generation` | Default. Partition assignment is read directly from the generation; a consumer holding zero partitions is healthy-idle and never recreates; a stalled partition rebuilds only its own reader, with no group rejoin. |
| `reader` | legacy `*kafka.Reader` with `GroupID` | Rollback path, retained for one release (task-209 FR-5.1). |

Both engines use the same consumer-group IDs and commit the same offset value
for the same delivered message — `msg.Offset + 1` — so switching between them
is a pod restart. No topic, offset, consumer-group or database migration is
involved, and it is safe in either direction.

`GET /api/debug/consumers` reports `engine`, `assignedPartitions`,
`generationId` and `lastAssignmentAt` per consumer. An empty
`assignedPartitions` on the `consumergroup` engine means healthy-idle, not
stuck: with single-partition topics and two replicas, one member of every
group is expected to hold nothing.

`recreateCount` means different things per engine. On `reader` it counts group
rejoins (each rebalances every member of the group); on `consumergroup` it
counts local partition-reader rebuilds. Do not compare the number across a
rollback.

### Partition-count changes

Neither engine watches for partition-count changes: today's `ReaderConfig`
never sets `WatchPartitionChanges`, and kafka-go forwards that `false` into
its internal `ConsumerGroupConfig`, so `groupConfig()` matches it deliberately.
If a topic is ever repartitioned, set
`ConsumerGroupConfig.WatchPartitionChanges = true` in
`consumer/group.go`'s `groupConfig()` (kafka-go polls every
`PartitionWatchInterval`, default 5 s). That is a deliberate opt-in, not
something to enable as a side effect of another change.
```

- [ ] **Step 2: Run the full Go verification sweep**

```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean.

- [ ] **Step 3: Run tests across every module that depends on `atlas-kafka`**

63 modules under `services/` and `libs/` declare a dependency on `atlas-kafka`. Sweep them:

```bash
for mod in $(grep -rl "atlas-kafka" --include=go.mod services libs | xargs -n1 dirname); do
  echo "== $mod"
  ( cd "$mod" && go build ./... && go vet ./... && go test -race ./... ) || echo "FAILED: $mod"
done 2>&1 | tee /tmp/task209-modules.log
grep -c FAILED /tmp/task209-modules.log
```
Expected: `0` failures. Investigate any `FAILED:` line before proceeding — do not proceed with a partial pass.

- [ ] **Step 4: Run every repo guard**

```bash
cd ../.. \
  && tools/redis-key-guard.sh \
  && tools/goroutine-guard.sh \
  && tools/lint.sh --check
```
Expected: all exit 0. `tools/lint.sh --check` needs nvm on PATH — if it false-fails with a node error, source nvm first (`. ~/.nvm/nvm.sh && nvm use 22`). Run `tools/lint.sh` with no flags to auto-fix formatting, then re-run `--check`.

The service-registration, template, skill/job-id, buff-duration and movement-types guards do not apply — this branch touches no `services.json`, no deploy manifest, no tenant template and no job/skill constant. Confirm with `git diff --name-only main...HEAD`.

- [ ] **Step 5: Build every service image**

`libs/atlas-kafka`'s `go.mod` is not touched by this branch, but every service links the library, so the bake is the only thing that catches a Dockerfile/`go.work` mismatch:

```bash
docker buildx bake all-go-services 2>&1 | tail -30
```
Expected: every target builds. Expect multiple fix-and-rebuild cycles if anything is off; do not shortcut this step.

- [ ] **Step 6: Prove FR-3.2 and the acceptance criteria mechanically**

```bash
git diff --stat services/                                     # must be empty
git diff main...HEAD --name-only                              # only libs/atlas-kafka + docs/tasks/task-209-*
grep -rn "GroupID:" libs/atlas-kafka/consumer/group.go        # must find nothing
grep -rn "TODO\|FIXME" libs/atlas-kafka/consumer/             # must find nothing
```
Expected: empty, a file list confined to `libs/atlas-kafka/` and `docs/tasks/task-209-assignment-aware-watchdog/`, and no matches for the last two.

- [ ] **Step 7: Record the evidence**

Create `docs/tasks/task-209-assignment-aware-watchdog/verification.md` with the **actual** output of Steps 2–6 — the unit-test summary line, the module-sweep failure count, each guard's exit status, the bake result, the S1 p99/max per engine from Task 8 Step 5, and the S6 `totalRecreates`. Quote real numbers; if a step was skipped or partially run, say so explicitly rather than implying a clean sweep.

Include a "Post-deploy measurements still outstanding" section listing the two live checks the PRD requires and that no test can substitute:
- `count_over_time({namespace="atlas-main"} |= "Recreated reader for topic" [1h])` ~0 per service (baseline 19–246/hour/service).
- An attack→drop-visible trace over a 10-minute play session with no multi-second gap (baseline: 4.7 s and 4.2 s stalls in trace `bd9b801a…`).

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-kafka/README.md docs/tasks/task-209-assignment-aware-watchdog/verification.md
git commit -m "docs(task-209): document consumer engines and record branch verification"
```

- [ ] **Step 9: Code review before the PR**

Per CLAUDE.md, run the code-review step before opening a PR — do not skip it even though the plan is complete. Invoke `superpowers:requesting-code-review`; it dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (Go files changed; no TS, so no frontend reviewer). Findings land in `docs/tasks/task-209-assignment-aware-watchdog/audit.md`. Pin the reviewer subagents to Sonnet — review workflows use the cheaper model.

---

## Open items carried forward (not in scope for this branch)

These are recorded in the PRD §9 and design §10 and are deliberately **not**
tasks here. Do not silently expand scope to cover them:

- **Group blast radius (PRD §9.2).** One group spans N topics, so any rejoin
  still stalls all of them. Option B in design §3 (one shared `ConsumerGroup`
  per group ID) is the structural fix and is unblocked by this task — it needs
  its own task because it changes `AddConsumer`'s start semantics at every
  call site.
- **Zombie members (PRD §9.3).** 9 members for 2 pods on `Monster Registry
  Service`. Expected to shrink once rejoin churn stops; confirm post-deploy
  with the new `Snapshot.GenerationID` before deciding whether it needs a task.
- **Legacy engine removal (PRD §9.4).** Remove `engine_reader.go` and the
  `KAFKA_CONSUMER_ENGINE` switch in the release *after* every service has run
  `consumergroup` through at least two full deploy cycles with a recreate rate
  of ~0. A follow-up task, not a dangling TODO on this branch.
- **Producer-side latency.** `BatchTimeout: 50ms` and the one-message-at-a-time
  loop in `libs/atlas-kafka/producer/producer.go:61` — tracked separately.
- **Prometheus scraping.** `atlas_*` series are not scraped in `atlas-main`.
  This task deliberately routes operator-facing signals to logs and `Snapshot`
  rather than to metrics that would be invisible.
