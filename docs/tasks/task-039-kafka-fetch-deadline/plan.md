# Kafka FetchMessage Deadline + Tick-and-Escalate — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Kafka consumers in `libs/atlas-kafka` self-recover from indefinite `FetchMessage` blocks (the failure shape behind the 2026-04-30 atlas-maps/atlas-monsters silent wedge) by adding a per-call deadline plus a tick-and-escalate state machine that recreates the reader after `maxConsecutiveTimeouts` consecutive timeouts.

**Architecture:** Local change in `libs/atlas-kafka/consumer/`. Three layers: outer `Consumer.start` reader-lifecycle loop (existing, untouched), inner `runFetchLoop` (rewritten — per-call `context.WithTimeout`, eager `cancel()`, branch on err to tick/escalate/return), `Consumer` observable state (4 new mutex-guarded fields exposed via `Snapshot` and the existing `/api/debug/consumers` route). Defaults `5 * time.Minute` × 3 escalations → ~15 minute worst-case detection latency. Sentinel `errFetchWedged` flows into the existing outer recreate-with-backoff path; no new error-handling surface.

**Tech Stack:** Go 1.21+, `github.com/segmentio/kafka-go`, `github.com/Chronicle20/atlas/libs/atlas-model/model.Decorator[T]`, `github.com/sirupsen/logrus`, `context.WithTimeout` / `errors.Is`. Existing test fakes (`MockReader`, `ChannelMockReader`, `scriptedReader`, `readerFactory`) plus `runtime.NumGoroutine()` for the leak guard.

---

## Branching

This work lives on a dedicated branch. Per CLAUDE.md and the saved feedback memory, never commit directly to `main` — branch protection will block the push.

- [ ] **Step 0.1: Create the feature branch (if not already on it)**

```bash
git checkout -b task-039-kafka-fetch-deadline 2>/dev/null || git checkout task-039-kafka-fetch-deadline
git status
```

Expected: on branch `task-039-kafka-fetch-deadline`. Working tree may already contain the `docs/tasks/task-039-kafka-fetch-deadline/` folder; that is fine.

---

## Task 1: Test fake fidelity fix

Real kafka-go's `FetchMessage` returns `ctx.Err()` (which is `DeadlineExceeded` on timeout, `Canceled` on cancel). The three test fakes currently return the literal `context.Canceled`, which would mask the new state machine's `DeadlineExceeded` branch. This is a one-line fix per fake. Existing tests are unaffected because they only cancel the parent ctx — `ctx.Err()` returns `Canceled` in that path, same value as the literal.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager_test.go` (3 sites: `MockReader.FetchMessage`, `ChannelMockReader.FetchMessage`, `scriptedReader.FetchMessage`)

- [ ] **Step 1.1: Apply the fix to `MockReader.FetchMessage`**

Replace at `libs/atlas-kafka/consumer/manager_test.go:32-40`:

```go
func (r *MockReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if !r.read {
		r.read = true
		return r.msg, nil
	}

	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}
```

(Only the last line changes: `context.Canceled` → `ctx.Err()`.)

- [ ] **Step 1.2: Apply the fix to `ChannelMockReader.FetchMessage`**

Replace at `libs/atlas-kafka/consumer/manager_test.go:117-124`:

```go
func (r *ChannelMockReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	select {
	case m := <-r.msgCh:
		return m, nil
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	}
}
```

(Only the `<-ctx.Done()` branch changes.)

- [ ] **Step 1.3: Apply the fix to `scriptedReader.FetchMessage`**

Replace at `libs/atlas-kafka/consumer/manager_test.go:443-461` — change only the trailing block:

```go
	r.mu.Unlock()
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
```

(Only the last line changes.)

- [ ] **Step 1.4: Run existing tests to confirm no regression**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1
```

Expected: PASS. All existing tests should still pass; the `ctx.Err()` value for parent-ctx-cancel is `Canceled`, identical to the previous literal.

- [ ] **Step 1.5: Commit**

```bash
git add libs/atlas-kafka/consumer/manager_test.go
git commit -m "test(kafka-consumer): return ctx.Err() from fake readers

Fakes previously returned the literal context.Canceled after <-ctx.Done().
Real segmentio/kafka-go returns ctx.Err(), which is DeadlineExceeded on
timeout and Canceled on cancel. The upcoming FetchMessage-deadline state
machine distinguishes the two branches; the fakes need to do likewise."
```

---

## Task 2: Add `Config` fields and decorators

Two new fields on `Config`, two new decorators (`SetFetchTimeout`, `SetMaxConsecutiveTimeouts`), defaults wired into `NewConfig`. TDD: extend `config_test.go` first.

**Files:**
- Modify: `libs/atlas-kafka/consumer/config.go`
- Modify: `libs/atlas-kafka/consumer/config_test.go`

- [ ] **Step 2.1: Add the failing test**

Append to `libs/atlas-kafka/consumer/config_test.go` (note: file is in package `consumer`, not `consumer_test`):

```go
func TestFetchTimeoutDefaultsAndOverride(t *testing.T) {
	c := NewConfig([]string{"test"}, "test", "test_topic", "test_group")

	if c.fetchTimeout != 5*time.Minute {
		t.Fatalf("expected default fetchTimeout=5m, got %v", c.fetchTimeout)
	}
	if c.maxConsecutiveTimeouts != 3 {
		t.Fatalf("expected default maxConsecutiveTimeouts=3, got %d", c.maxConsecutiveTimeouts)
	}

	c, err := model.Decorate(model.Decorators(SetFetchTimeout(20*time.Minute)))(c)
	if err != nil || c.fetchTimeout != 20*time.Minute {
		t.Fatalf("expected SetFetchTimeout to override to 20m, got %v (err=%v)", c.fetchTimeout, err)
	}

	c, err = model.Decorate(model.Decorators(SetMaxConsecutiveTimeouts(7)))(c)
	if err != nil || c.maxConsecutiveTimeouts != 7 {
		t.Fatalf("expected SetMaxConsecutiveTimeouts to override to 7, got %d (err=%v)", c.maxConsecutiveTimeouts, err)
	}
}
```

- [ ] **Step 2.2: Run the test to verify it fails**

```bash
cd libs/atlas-kafka && go test ./consumer/ -run TestFetchTimeoutDefaultsAndOverride -count=1
```

Expected: FAIL — compile error referencing undefined fields `fetchTimeout`, `maxConsecutiveTimeouts` and undefined functions `SetFetchTimeout`, `SetMaxConsecutiveTimeouts`.

- [ ] **Step 2.3: Add the fields and decorators**

Replace `libs/atlas-kafka/consumer/config.go` in full:

```go
package consumer

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

//goland:noinspection GoUnusedExportedFunction
func NewConfig(brokers []string, name string, topic string, groupId string) Config {
	return Config{
		brokers:                brokers,
		name:                   name,
		topic:                  topic,
		groupId:                groupId,
		maxWait:                50 * time.Millisecond,
		startOffset:            kafka.FirstOffset,
		fetchTimeout:           5 * time.Minute,
		maxConsecutiveTimeouts: 3,
	}
}

type Config struct {
	brokers                []string
	name                   string
	topic                  string
	groupId                string
	maxWait                time.Duration
	headerParsers          []HeaderParser
	startOffset            int64
	fetchTimeout           time.Duration
	maxConsecutiveTimeouts int
}

//goland:noinspection GoUnusedExportedFunction
func SetStartOffset(startOffset int64) model.Decorator[Config] {
	return func(config Config) Config {
		config.startOffset = startOffset
		return config
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetMaxWait(duration time.Duration) model.Decorator[Config] {
	return func(config Config) Config {
		config.maxWait = duration
		return config
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetHeaderParsers(parsers ...HeaderParser) model.Decorator[Config] {
	return func(config Config) Config {
		config.headerParsers = parsers
		return config
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetFetchTimeout(d time.Duration) model.Decorator[Config] {
	return func(config Config) Config {
		config.fetchTimeout = d
		return config
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetMaxConsecutiveTimeouts(n int) model.Decorator[Config] {
	return func(config Config) Config {
		config.maxConsecutiveTimeouts = n
		return config
	}
}
```

- [ ] **Step 2.4: Run the test to verify it passes**

```bash
cd libs/atlas-kafka && go test ./consumer/ -run TestFetchTimeoutDefaultsAndOverride -count=1
```

Expected: PASS.

- [ ] **Step 2.5: Run the full consumer test suite to verify no regression**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1
```

Expected: PASS.

- [ ] **Step 2.6: Commit**

```bash
git add libs/atlas-kafka/consumer/config.go libs/atlas-kafka/consumer/config_test.go
git commit -m "feat(kafka-consumer): add fetchTimeout and maxConsecutiveTimeouts to Config

Adds two fields with defaults 5m and 3, plus SetFetchTimeout and
SetMaxConsecutiveTimeouts decorators. Wired into the upcoming
FetchMessage-deadline state machine via AddConsumer. Defaults yield
~15-minute worst-case detection latency for an indefinitely-wedged
FetchMessage call."
```

---

## Task 3: Add `Consumer` state, `Snapshot` extension, and `recordTimeout`

Four new fields on `Consumer` (`fetchTimeout`, `maxConsecutiveTimeouts`, `consecutiveTimeouts`, `lastTimeoutAt`); two new fields on `Snapshot` (`LastTimeoutAt`, `ConsecutiveTimeouts`); new method `recordTimeout`. `recordFetch` reset of counter and `onReaderCreated` reset on recreate are deferred to Task 4 to keep this commit focused on additive surface.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go` (Consumer struct lines 180-197, Snapshot struct lines 199-212, Snapshot() method lines 214-231, add new method)

- [ ] **Step 3.1: Add fields to `Consumer` struct**

Replace `libs/atlas-kafka/consumer/manager.go:180-197` with:

```go
// Consumer owns a single Kafka reader for a single topic. Its reader is
// allowed to die (EOF, retry exhaustion, rebalance errors) — the outer
// lifecycle loop in start rebuilds the reader and rejoins the consumer
// group without disturbing the surrounding process.
type Consumer struct {
	name          string
	topic         string
	groupId       string
	brokers       []string
	readerConfig  kafka.ReaderConfig
	rp            ReaderProducer
	handlers      map[string]handler.Handler
	headerParsers []HeaderParser
	mu            sync.Mutex

	// Read-only after construction; copied from Config in AddConsumer.
	fetchTimeout           time.Duration
	maxConsecutiveTimeouts int

	// Observable state — protected by mu.
	aliveSince          time.Time
	lastFetchAt         time.Time
	lastErrorAt         time.Time
	lastError           string
	recreateCount       int
	consecutiveTimeouts int
	lastTimeoutAt       time.Time
}
```

- [ ] **Step 3.2: Extend the `Snapshot` struct and `Snapshot()` method**

Replace `libs/atlas-kafka/consumer/manager.go:199-231` with:

```go
// Snapshot is a point-in-time view of a Consumer's observable state, suitable
// for JSON serialization by the debug route.
type Snapshot struct {
	Name                string
	Topic               string
	GroupID             string
	Brokers             []string
	AliveSince          time.Time
	LastFetchAt         time.Time
	LastErrorAt         time.Time
	LastError           string
	RecreateCount       int
	HandlerCount        int
	LastTimeoutAt       time.Time
	ConsecutiveTimeouts int
}

// Snapshot returns a consistent snapshot of the consumer's observable state.
func (c *Consumer) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	brokers := append([]string(nil), c.brokers...)
	return Snapshot{
		Name:                c.name,
		Topic:               c.topic,
		GroupID:             c.groupId,
		Brokers:             brokers,
		AliveSince:          c.aliveSince,
		LastFetchAt:         c.lastFetchAt,
		LastErrorAt:         c.lastErrorAt,
		LastError:           c.lastError,
		RecreateCount:       c.recreateCount,
		HandlerCount:        len(c.handlers),
		LastTimeoutAt:       c.lastTimeoutAt,
		ConsecutiveTimeouts: c.consecutiveTimeouts,
	}
}
```

- [ ] **Step 3.3: Add `recordTimeout` method**

Insert this method immediately after `recordFetch` (manager.go ~line 248). The method belongs to `Consumer` and lives next to its peers:

```go
// recordTimeout marks one deadline expiration; called per tick by runFetchLoop.
// Idle, not an error: lastError / lastErrorAt are untouched.
func (c *Consumer) recordTimeout() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastTimeoutAt = time.Now()
	c.consecutiveTimeouts++
}
```

- [ ] **Step 3.4: Verify the package compiles**

```bash
cd libs/atlas-kafka && go build ./consumer/...
```

Expected: PASS. No new logic is exercised yet, but the package must compile cleanly.

- [ ] **Step 3.5: Run existing tests**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1
```

Expected: PASS. Existing tests don't touch the new fields; behavior is unchanged.

- [ ] **Step 3.6: Commit**

```bash
git add libs/atlas-kafka/consumer/manager.go
git commit -m "feat(kafka-consumer): add timeout state to Consumer and Snapshot

Adds fetchTimeout, maxConsecutiveTimeouts (read-only after construction)
and consecutiveTimeouts, lastTimeoutAt (mutex-guarded observable state)
to the Consumer struct. Extends Snapshot with the two observable fields
and adds the recordTimeout method called per deadline expiration."
```

---

## Task 4: Counter-reset wiring and Config → Consumer plumbing

`recordFetch` resets the consecutive-timeout counter on success. `onReaderCreated(attempt > 0)` resets both the counter and the last-timeout timestamp on every recreate (per design §3.3, this is the second of the two structurally-guaranteed reset sites). `AddConsumer` copies the new Config fields onto the `Consumer`.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go` (recordFetch ~line 243-248, onReaderCreated ~line 233-241, AddConsumer ~line 119-128)

- [ ] **Step 4.1: Modify `recordFetch` to reset the counter**

Replace `recordFetch` (around manager.go:243-248) with:

```go
func (c *Consumer) recordFetch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFetchAt = time.Now()
	c.lastError = ""
	c.consecutiveTimeouts = 0
}
```

- [ ] **Step 4.2: Modify `onReaderCreated` to reset counter and timestamp on recreate**

Replace `onReaderCreated` (around manager.go:233-241) with:

```go
func (c *Consumer) onReaderCreated(attempt int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliveSince = time.Now()
	if attempt > 0 {
		c.recreateCount++
		c.lastError = ""
		c.consecutiveTimeouts = 0
		c.lastTimeoutAt = time.Time{}
	}
}
```

- [ ] **Step 4.3: Wire Config fields through `AddConsumer`**

In `AddConsumer` (around manager.go:119-128), update the `Consumer` literal to copy the two new fields. Replace the `con := &Consumer{...}` block with:

```go
		con := &Consumer{
			name:                   c.name,
			topic:                  c.topic,
			groupId:                c.groupId,
			brokers:                append([]string(nil), c.brokers...),
			readerConfig:           readerConfig,
			rp:                     m.rp,
			handlers:               make(map[string]handler.Handler),
			headerParsers:          c.headerParsers,
			fetchTimeout:           c.fetchTimeout,
			maxConsecutiveTimeouts: c.maxConsecutiveTimeouts,
		}
```

- [ ] **Step 4.4: Verify the package builds and tests pass**

```bash
cd libs/atlas-kafka && go build ./consumer/... && go test ./consumer/... -race -count=1
```

Expected: PASS.

- [ ] **Step 4.5: Commit**

```bash
git add libs/atlas-kafka/consumer/manager.go
git commit -m "feat(kafka-consumer): reset timeout counter on success and recreate

recordFetch zeroes consecutiveTimeouts on every successful fetch.
onReaderCreated zeroes counter + lastTimeoutAt on every recreate
(attempt > 0). AddConsumer copies fetchTimeout and
maxConsecutiveTimeouts from Config onto the Consumer. These are the
two structurally-guaranteed counter reset sites; runFetchLoop relies
on them rather than resetting explicitly."
```

---

## Task 5: Add the `errFetchWedged` sentinel

Tiny standalone task. The sentinel is used only by `runFetchLoop` (Task 6) but is added separately so the diff in Task 6 stays focused on the loop rewrite.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go` (immediately below the imports, before `KafkaReader` interface)

- [ ] **Step 5.1: Add the sentinel**

Insert immediately after the `import (...)` block (before `type KafkaReader interface`) in `libs/atlas-kafka/consumer/manager.go`:

```go
// errFetchWedged is returned from runFetchLoop when FetchMessage has hit
// its deadline maxConsecutiveTimeouts times in a row without a successful
// fetch in between. The outer start loop treats it identically to any
// other recreate-eligible error: close reader, backoff, rebuild.
var errFetchWedged = errors.New("consumer fetch wedged: exceeded consecutive timeouts")
```

- [ ] **Step 5.2: Verify build**

```bash
cd libs/atlas-kafka && go build ./consumer/...
```

Expected: PASS. The variable is package-private and unreferenced for now; Go does not warn on that.

- [ ] **Step 5.3: Commit**

```bash
git add libs/atlas-kafka/consumer/manager.go
git commit -m "feat(kafka-consumer): add errFetchWedged sentinel

Unexported sentinel returned from runFetchLoop after
maxConsecutiveTimeouts consecutive deadline expirations. The outer
start loop's recordError(err) writes the sentinel's message into
lastError, surfacing 'consumer fetch wedged: exceeded consecutive
timeouts' on /api/debug/consumers without a new code path."
```

---

## Task 6: Test 1 + rewrite `runFetchLoop` to tick on a single deadline

This is where the state machine lands. TDD shape: write the test that proves "one deadline expiration ticks but does not recreate," watch it fail against the current implementation (current loop has no per-call deadline), rewrite `runFetchLoop` to drive the loop by `context.WithTimeout` and the tick-and-escalate state machine. Drop the inner `retry.Try` block and the `retry` package import. The same rewrite makes Test 2 and Test 3 (Tasks 7-8) pass; we still write each test explicitly so each behavior has its own assertion.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go` (rewrite `runFetchLoop`, lines 333-374; drop `retry` import line 13)
- Modify: `libs/atlas-kafka/consumer/manager_test.go` (append Test 1)

- [ ] **Step 6.1: Add Test 1 — `TestFetchTimeoutTicksWithoutRecreate`**

Append to `libs/atlas-kafka/consumer/manager_test.go`:

```go
func TestFetchTimeoutTicksWithoutRecreate(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	// Empty scriptedReader always blocks on ctx — every FetchMessage call
	// returns DeadlineExceeded when the per-call deadline fires.
	r1 := &scriptedReader{}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "tick-consumer", "tick-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(50*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	// Wait long enough for at least one deadline to fire (50ms) but well
	// short of three (150ms).
	time.Sleep(75 * time.Millisecond)

	snaps := cm.Consumers()
	if len(snaps) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(snaps))
	}
	s := snaps[0].Snapshot()

	if s.ConsecutiveTimeouts < 1 {
		t.Fatalf("expected ConsecutiveTimeouts >= 1 after a tick, got %d", s.ConsecutiveTimeouts)
	}
	if s.RecreateCount != 0 {
		t.Fatalf("expected RecreateCount == 0 after a single tick, got %d", s.RecreateCount)
	}
	if s.LastError != "" {
		t.Fatalf("expected LastError empty (idle is not an error), got %q", s.LastError)
	}
	if s.LastTimeoutAt.IsZero() {
		t.Fatal("expected LastTimeoutAt to be set after a tick")
	}

	cancel()
	wg.Wait()

	if r1.Closes() != 1 {
		t.Fatalf("expected reader closed exactly once on ctx-cancel, got %d", r1.Closes())
	}
}
```

- [ ] **Step 6.2: Run the test to verify it fails**

```bash
cd libs/atlas-kafka && go test ./consumer/ -run TestFetchTimeoutTicksWithoutRecreate -count=1 -race
```

Expected: FAIL — current `runFetchLoop` has no per-call deadline, so `FetchMessage` blocks indefinitely on the empty `scriptedReader`. `ConsecutiveTimeouts` stays at 0, the assertion `>= 1` fires.

- [ ] **Step 6.3: Rewrite `runFetchLoop`**

Replace the existing `runFetchLoop` (manager.go:333-374) with the full state machine:

```go
// runFetchLoop blocks the caller until the supplied reader errors out or
// the parent ctx is canceled. Each iteration runs FetchMessage under a
// per-call deadline (c.fetchTimeout). A deadline expiration is treated as
// idle: the consumer ticks back to the top of the loop with a fresh
// deadline, the same reader, and the same partition assignment. After
// c.maxConsecutiveTimeouts consecutive deadline expirations without a
// successful fetch in between, the loop returns errFetchWedged so the
// outer start loop closes the reader and recreates it. A successful
// fetch resets the counter via recordFetch.
func (c *Consumer) runFetchLoop(l logrus.FieldLogger, ctx context.Context, reader KafkaReader) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fetchCtx, cancelFetch := context.WithTimeout(ctx, c.fetchTimeout)
		msg, err := reader.FetchMessage(fetchCtx)
		cancelFetch()

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return err
			}
			if errors.Is(err, context.DeadlineExceeded) {
				c.recordTimeout()
				snapshot := c.Snapshot()
				if snapshot.ConsecutiveTimeouts >= c.maxConsecutiveTimeouts {
					l.Warnf("FetchMessage wedged: %d consecutive timeouts on topic [%s] (group [%s]); forcing reader recreate.",
						snapshot.ConsecutiveTimeouts, c.topic, c.groupId)
					return errFetchWedged
				}
				l.Debugf("FetchMessage deadline expired (consecutive=%d/%d); ticking.",
					snapshot.ConsecutiveTimeouts, c.maxConsecutiveTimeouts)
				continue
			}
			return err
		}

		c.recordFetch()
		l.Debugf("Message received %s.", string(msg.Value))
		if c.processMessage(l, ctx, msg) {
			if cerr := reader.CommitMessages(ctx, msg); cerr != nil {
				l.WithError(cerr).Warnf("Could not commit message offset, it may be redelivered.")
			}
		}
	}
}
```

- [ ] **Step 6.4: Drop the now-unused `retry` import**

In `libs/atlas-kafka/consumer/manager.go`, remove this line from the import block (was line 13):

```go
	"github.com/Chronicle20/atlas/libs/atlas-retry"
```

(`retry.Try` and `retry.DefaultConfig` are no longer referenced in the package.)

- [ ] **Step 6.5: Run Test 1 to verify it passes**

```bash
cd libs/atlas-kafka && go test ./consumer/ -run TestFetchTimeoutTicksWithoutRecreate -count=1 -race
```

Expected: PASS.

- [ ] **Step 6.6: Run the full consumer test suite — existing tests must still pass**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1
```

Expected: PASS for all existing tests. Per design §9.3, no existing test exceeds 1s and the 5-minute default deadline never fires — none should need scaffolding changes. If any existing test newly fails, audit it against the test-fake fidelity fix (Task 1) and the state-machine semantics; do NOT add `SetFetchTimeout` decorators reflexively.

- [ ] **Step 6.7: Verify the build outside the consumer package is clean**

```bash
cd libs/atlas-kafka && go build ./... && go vet ./...
```

Expected: PASS. The `retry` package is still used by `producer/producer.go`, so its presence in the workspace is not affected.

- [ ] **Step 6.8: Commit**

```bash
git add libs/atlas-kafka/consumer/manager.go libs/atlas-kafka/consumer/manager_test.go
git commit -m "feat(kafka-consumer): add per-call FetchMessage deadline with tick-and-escalate

Rewrites runFetchLoop to drive each iteration under context.WithTimeout(
ctx, c.fetchTimeout). DeadlineExceeded with a live parent ctx is treated
as idle: the consumer ticks back to the loop with a fresh deadline (same
reader, same partition assignment). After maxConsecutiveTimeouts ticks
without a successful fetch in between, the loop returns errFetchWedged
to the outer recreate-with-backoff path.

Drops the inner retry.Try block (PRD 4.3): the per-call deadline is the
correct retry primitive; the outer start loop is the correct recreate
mechanism. Drops the now-unused retry package import.

This addresses the 2026-04-30 incident where atlas-maps and
atlas-monsters were observed alive ~18h with lastFetchAt zero,
recreateCount 0, lastError empty — the failure shape that task-016's
error-driven recreate could not detect."
```

---

## Task 7: Test 2 — escalate to wedge after `maxConsecutiveTimeouts`

Verifies the full escalation path: empty r1 wedges, sentinel triggers outer recreate, r2 delivers a message. Includes the goroutine-leak guard from risks.md R2.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager_test.go`

- [ ] **Step 7.1: Append Test 2**

```go
func TestFetchTimeoutEscalatesAfterMaxToWedge(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	// r1: empty — every FetchMessage hits the deadline. After 3 ticks
	// runFetchLoop returns errFetchWedged, the outer loop closes r1 and
	// requests r2.
	// r2: delivers one message. Handler invocation is the signal that the
	// recreate path completed.
	r1 := &scriptedReader{}
	r2 := &scriptedReader{script: []scriptedFetch{{msg: kafka.Message{Value: []byte("after-wedge")}}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	// Goroutine-leak guard (risks R2): capture before the consumer starts.
	goroutinesBefore := runtime.NumGoroutine()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "wedge-consumer", "wedge-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(50*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	handlerDone := make(chan struct{})
	_, _ = cm.RegisterHandler("wedge-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		close(handlerDone)
		return true, nil
	})

	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was never invoked on recreated reader after wedge")
	}

	if r1.Closes() != 1 {
		t.Fatalf("expected r1 closed exactly once after wedge, got %d", r1.Closes())
	}

	snaps := cm.Consumers()
	if len(snaps) != 1 {
		t.Fatalf("expected 1 consumer, got %d", len(snaps))
	}
	s := snaps[0].Snapshot()

	if s.RecreateCount < 1 {
		t.Fatalf("expected RecreateCount >= 1 after wedge recreate, got %d", s.RecreateCount)
	}
	if s.LastError != "consumer fetch wedged: exceeded consecutive timeouts" {
		t.Fatalf("expected wedge sentinel in LastError, got %q", s.LastError)
	}
	// Counter must be reset by onReaderCreated for the new reader.
	if s.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected ConsecutiveTimeouts reset to 0 on new reader, got %d", s.ConsecutiveTimeouts)
	}

	// Goroutine-leak guard: settle then compare. If FetchMessage on r1 did
	// not honor ctx cancellation, leaked goroutines accumulate here.
	time.Sleep(50 * time.Millisecond)
	goroutinesAfter := runtime.NumGoroutine()
	if delta := goroutinesAfter - goroutinesBefore; delta > 5 {
		t.Fatalf("goroutine leak suspected: before=%d after=%d delta=%d (>5)",
			goroutinesBefore, goroutinesAfter, delta)
	}
}
```

- [ ] **Step 7.2: Add the `runtime` import to the test file**

If `runtime` is not already imported in `libs/atlas-kafka/consumer/manager_test.go`, add it to the import block:

```go
	"runtime"
```

(Place it alphabetically between `io` and `sync`. If goimports is installed, `goimports -w libs/atlas-kafka/consumer/manager_test.go` does this automatically.)

- [ ] **Step 7.3: Run Test 2 to verify it passes**

```bash
cd libs/atlas-kafka && go test ./consumer/ -run TestFetchTimeoutEscalatesAfterMaxToWedge -count=1 -race
```

Expected: PASS. The `scriptedReader` honors ctx cancellation (the Task 1 fidelity fix returns `ctx.Err()`), so the leak guard should comfortably stay under the threshold of 5.

- [ ] **Step 7.4: Run the full consumer suite**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1
```

Expected: PASS.

- [ ] **Step 7.5: Commit**

```bash
git add libs/atlas-kafka/consumer/manager_test.go
git commit -m "test(kafka-consumer): assert wedge escalation triggers reader recreate

After 3 deadline expirations on an empty reader, runFetchLoop returns
errFetchWedged. The outer loop closes the wedged reader, recreates,
and the handler fires on the second reader. Asserts RecreateCount,
sentinel string in LastError, counter reset on new reader, and
includes a runtime.NumGoroutine() guard for risks.md R2 (kafka-go
ctx-cancel honored)."
```

---

## Task 8: Test 3 — counter resets on successful fetch

Alternating timeout/success pattern. Counter must never reach `maxConsecutiveTimeouts`; no recreate.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager_test.go`

- [ ] **Step 8.1: Add a custom alternating reader and the test**

Append to `libs/atlas-kafka/consumer/manager_test.go`:

```go
// alternatingReader returns DeadlineExceeded on odd-numbered FetchMessage
// calls (1st, 3rd, 5th, ...) and a scripted message on even-numbered calls.
// Used to exercise the timeout-success-timeout-success cycle that should
// keep consecutiveTimeouts pinned at 0 across many iterations.
type alternatingReader struct {
	mu        sync.Mutex
	calls     int
	committed []kafka.Message
	closes    int
}

func (r *alternatingReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	r.mu.Lock()
	r.calls++
	n := r.calls
	r.mu.Unlock()

	if n%2 == 1 {
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	return kafka.Message{Value: []byte("ok")}, nil
}

func (r *alternatingReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.committed = append(r.committed, msgs...)
	return nil
}

func (r *alternatingReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes++
	return nil
}

func (r *alternatingReader) Closes() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closes
}

func (r *alternatingReader) Committed() []kafka.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]kafka.Message, len(r.committed))
	copy(out, r.committed)
	return out
}

func TestFetchTimeoutResetsOnSuccessfulFetch(t *testing.T) {
	consumer.ResetInstance()

	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r := &alternatingReader{}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r))

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "reset-consumer", "reset-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(50*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	handlerInvocations := atomic.Int32{}
	gotThree := make(chan struct{})
	_, _ = cm.RegisterHandler("reset-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		if handlerInvocations.Add(1) == 3 {
			close(gotThree)
		}
		return true, nil
	})

	// 3 successes interleaved with 3 timeouts ≈ 200ms wall-clock.
	select {
	case <-gotThree:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected 3 handler invocations, got %d", handlerInvocations.Load())
	}

	snaps := cm.Consumers()
	s := snaps[0].Snapshot()

	if s.RecreateCount != 0 {
		t.Fatalf("expected RecreateCount == 0 (counter resets between successes), got %d", s.RecreateCount)
	}
	if s.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected ConsecutiveTimeouts == 0 after a success, got %d", s.ConsecutiveTimeouts)
	}
	if r.Closes() != 0 {
		t.Fatalf("expected reader to remain open across timeout/success cycles, got Closes=%d", r.Closes())
	}
}
```

- [ ] **Step 8.2: Run Test 3 to verify it passes**

```bash
cd libs/atlas-kafka && go test ./consumer/ -run TestFetchTimeoutResetsOnSuccessfulFetch -count=1 -race
```

Expected: PASS.

- [ ] **Step 8.3: Run the full suite**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1
```

Expected: PASS.

- [ ] **Step 8.4: Commit**

```bash
git add libs/atlas-kafka/consumer/manager_test.go
git commit -m "test(kafka-consumer): assert counter resets across timeout/success cycles

alternatingReader returns DeadlineExceeded on odd calls and a scripted
message on even calls. After 3 successes interleaved with 3 timeouts,
RecreateCount stays 0, ConsecutiveTimeouts is 0, and the reader is
never closed — the counter reset in recordFetch holds."
```

---

## Task 9: Expose `consecutiveTimeouts` and `lastTimeoutAt` on the debug route

Add the two attributes to the JSON:API serializer in `debug.go` and to the in-test struct in `debug_test.go`. Extend `TestDebugHandler_PopulatedConsumer` to assert the new attributes serialize correctly with their zero values on a freshly-fetched consumer.

**Files:**
- Modify: `libs/atlas-kafka/consumer/debug.go`
- Modify: `libs/atlas-kafka/consumer/debug_test.go`

- [ ] **Step 9.1: Update `debugAttributes` in `debug.go`**

Replace `libs/atlas-kafka/consumer/debug.go:55-66` with:

```go
type debugAttributes struct {
	Name                string    `json:"name"`
	Topic               string    `json:"topic"`
	GroupID             string    `json:"groupId"`
	Brokers             []string  `json:"brokers"`
	AliveSince          time.Time `json:"aliveSince"`
	LastFetchAt         time.Time `json:"lastFetchAt"`
	LastErrorAt         time.Time `json:"lastErrorAt"`
	LastError           string    `json:"lastError"`
	RecreateCount       int       `json:"recreateCount"`
	HandlerCount        int       `json:"handlerCount"`
	LastTimeoutAt       time.Time `json:"lastTimeoutAt"`
	ConsecutiveTimeouts int       `json:"consecutiveTimeouts"`
}
```

- [ ] **Step 9.2: Update `snapshotToAttributes` in `debug.go`**

Replace `libs/atlas-kafka/consumer/debug.go:68-81` with:

```go
func snapshotToAttributes(s Snapshot) debugAttributes {
	return debugAttributes{
		Name:                s.Name,
		Topic:               s.Topic,
		GroupID:             s.GroupID,
		Brokers:             s.Brokers,
		AliveSince:          s.AliveSince,
		LastFetchAt:         s.LastFetchAt,
		LastErrorAt:         s.LastErrorAt,
		LastError:           s.LastError,
		RecreateCount:       s.RecreateCount,
		HandlerCount:        s.HandlerCount,
		LastTimeoutAt:       s.LastTimeoutAt,
		ConsecutiveTimeouts: s.ConsecutiveTimeouts,
	}
}
```

- [ ] **Step 9.3: Mirror the new fields in the test-side `debugAttributes` struct**

Replace `libs/atlas-kafka/consumer/debug_test.go:29-40` with:

```go
type debugAttributes struct {
	Name                string    `json:"name"`
	Topic               string    `json:"topic"`
	GroupID             string    `json:"groupId"`
	Brokers             []string  `json:"brokers"`
	AliveSince          time.Time `json:"aliveSince"`
	LastFetchAt         time.Time `json:"lastFetchAt"`
	LastErrorAt         time.Time `json:"lastErrorAt"`
	LastError           string    `json:"lastError"`
	RecreateCount       int       `json:"recreateCount"`
	HandlerCount        int       `json:"handlerCount"`
	LastTimeoutAt       time.Time `json:"lastTimeoutAt"`
	ConsecutiveTimeouts int       `json:"consecutiveTimeouts"`
}
```

- [ ] **Step 9.4: Extend `TestDebugHandler_PopulatedConsumer` with assertions on the new fields**

Append before the closing brace of `TestDebugHandler_PopulatedConsumer` (after the existing `if a.LastFetchAt.IsZero() { ... }` assertion at debug_test.go:165-167):

```go
	if a.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected consecutiveTimeouts=0 after a successful fetch, got %d", a.ConsecutiveTimeouts)
	}
	if !a.LastTimeoutAt.IsZero() {
		t.Fatalf("expected lastTimeoutAt zero on a consumer that has never timed out, got %v", a.LastTimeoutAt)
	}
```

- [ ] **Step 9.5: Run debug tests to verify**

```bash
cd libs/atlas-kafka && go test ./consumer/ -run TestDebugHandler -count=1 -race
```

Expected: PASS.

- [ ] **Step 9.6: Run the full consumer suite**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1
```

Expected: PASS for all tests (TDD tests + existing tests + new tests).

- [ ] **Step 9.7: Commit**

```bash
git add libs/atlas-kafka/consumer/debug.go libs/atlas-kafka/consumer/debug_test.go
git commit -m "feat(kafka-consumer): expose consecutiveTimeouts on /api/debug/consumers

Adds consecutiveTimeouts and lastTimeoutAt to the JSON:API attributes
block. Pre-escalation operator signature: consecutiveTimeouts > 0 with
no recent lastFetchAt is the unique 'wedge in progress' state that was
missing during the 2026-04-30 incident. Post-recovery: lastError carries
the wedge sentinel and recreateCount is non-zero (existing behavior)."
```

---

## Task 10: Library-level verification

Final clean-room verification that `libs/atlas-kafka` builds, vets, and tests pass under the race detector.

- [ ] **Step 10.1: Run vet on the library**

```bash
cd libs/atlas-kafka && go vet ./...
```

Expected: no output (PASS). If the `retry` import was successfully dropped from `manager.go`, vet will not flag an unused import.

- [ ] **Step 10.2: Run all consumer tests under `-race -count=1`**

```bash
cd libs/atlas-kafka && go test ./consumer/... -race -count=1 -timeout 60s
```

Expected: PASS, no race warnings.

- [ ] **Step 10.3: Run the full library test suite**

```bash
cd libs/atlas-kafka && go test ./... -race -count=1 -timeout 120s
```

Expected: PASS.

- [ ] **Step 10.4: Run vet across the workspace**

```bash
cd <repo-root> && go vet ./libs/atlas-kafka/... && go vet ./services/...
```

Expected: no output from either invocation. The new exported decorators (`SetFetchTimeout`, `SetMaxConsecutiveTimeouts`) are unused in services today; the `//goland:noinspection GoUnusedExportedFunction` annotations on each preempt vet's unused-export concern.

If any service-side vet complaint appears, do NOT modify the service to silence it — investigate root cause. The library change is intentionally service-transparent.

---

## Task 11: Docker build verification for primary affected services

CLAUDE.md mandates Docker build verification for shared-library changes. PRD §10 names atlas-maps, atlas-monsters, and atlas-channel as the minimum set.

- [ ] **Step 11.1: Build atlas-maps**

```bash
cd <repo-root> && docker build -f services/atlas-maps/Dockerfile -t atlas-maps:task-039 .
```

Expected: build succeeds. The Dockerfile copies `libs/atlas-kafka` and references it via go.work replace — no Dockerfile changes are needed.

- [ ] **Step 11.2: Build atlas-monsters**

```bash
cd <repo-root> && docker build -f services/atlas-monsters/Dockerfile -t atlas-monsters:task-039 .
```

Expected: build succeeds.

- [ ] **Step 11.3: Build atlas-channel**

```bash
cd <repo-root> && docker build -f services/atlas-channel/Dockerfile -t atlas-channel:task-039 .
```

Expected: build succeeds.

- [ ] **Step 11.4: Verify no committed changes are pending and the branch is clean**

```bash
git status
git log --oneline main..HEAD
```

Expected: working tree clean; `git log` shows the task-039 commits in order:
1. `test(kafka-consumer): return ctx.Err() from fake readers`
2. `feat(kafka-consumer): add fetchTimeout and maxConsecutiveTimeouts to Config`
3. `feat(kafka-consumer): add timeout state to Consumer and Snapshot`
4. `feat(kafka-consumer): reset timeout counter on success and recreate`
5. `feat(kafka-consumer): add errFetchWedged sentinel`
6. `feat(kafka-consumer): add per-call FetchMessage deadline with tick-and-escalate`
7. `test(kafka-consumer): assert wedge escalation triggers reader recreate`
8. `test(kafka-consumer): assert counter resets across timeout/success cycles`
9. `feat(kafka-consumer): expose consecutiveTimeouts on /api/debug/consumers`

---

## Acceptance criteria mapping (PRD §10)

For verification by the audit phase. Each PRD criterion → which task(s) implement it:

| PRD criterion | Implemented by |
|---|---|
| Behavioral: 1 timeout → tick → no recreate | Task 6 (Test 1 + state machine) |
| Behavioral: max timeouts → `errFetchWedged` → recreate | Task 7 (Test 2) + Task 6 (state machine) |
| Behavioral: success between timeouts resets counter | Task 4 (`recordFetch`) + Task 8 (Test 3) |
| Behavioral: parent ctx cancel exits cleanly within one tick | Task 6 (state machine ctx-cancel branch) + existing `TestContextCancelDoesNotRecreate` |
| Behavioral: non-deadline error returns directly | Task 6 (state machine "other error" branch) + existing `TestRecreatesReaderOnEOF` / `TestRetryExhaustionRecreatesReader` (still pass) |
| Behavioral: active topics never observe deadline | Default 5m vs ms-scale message arrival; verifiable post-deploy on `/api/debug/consumers` |
| Observability: `consecutiveTimeouts` + `lastTimeoutAt` on `/api/debug/consumers` | Task 9 |
| Observability: counter resets on success/recreate, increments on tick | Tasks 4 + 6 + Tests 1, 2, 3 |
| Observability: post-recreate `lastError` contains the sentinel | Task 5 + outer loop's existing `recordError(err)` (Test 2 asserts) |
| Observability: one Warn log per wedge | Task 6 (Warn line) |
| Configuration: `SetFetchTimeout` overrides default | Task 2 (`TestFetchTimeoutDefaultsAndOverride`) |
| Configuration: `SetMaxConsecutiveTimeouts` overrides default | Task 2 |
| Configuration: defaults yield 5m × 3 | Task 2 (`NewConfig` defaults) |
| Configuration: decorators stack | Existing decorator pattern (`model.Decorate`); Task 2 covers structurally |
| Non-regression: existing tests pass | Tasks 1 (fake fix) + 6 + 10 |
| Non-regression: 49 services build | Task 10 (workspace `go build`) + Task 11 (Docker for primary 3) |
| Non-regression: Docker build for atlas-maps, atlas-monsters, atlas-channel | Task 11 |
| Non-regression: steady-state behavior unchanged | Implicit: 5m default never fires under ms-scale traffic; covered by existing tests still passing |
| Non-regression: `go vet` clean | Task 10 |
| Tests: tick test | Task 6 (Test 1) |
| Tests: wedge test | Task 7 (Test 2) |
| Tests: reset test | Task 8 (Test 3) |
| Tests: fake extension documented | Task 1 commit message |
| Tests: `debug_test.go` extended for new attribute keys | Task 9 |
| Build: `libs/atlas-kafka/consumer` builds | Task 10 |
| Build: `go build ./...` from monorepo root | Task 10.4 |
| Build: Docker for atlas-maps, atlas-monsters, atlas-channel | Task 11 |
