# Kafka Consumer Fetch-Wedge Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the multi-second-to-minute Kafka consume dwell by removing the idle-wedge reader-recreate churn in `libs/atlas-kafka/consumer`, detecting genuine stalls by reader progress instead of message silence, and proving the fix with a deterministic testcontainers harness.

**Architecture:** Design Approach A (`docs/tasks/task-136-consumer-fetch-wedge/design.md` §3): keep one reader per topic / one group per service; reinterpret an expired fetch deadline as an **idle tick** (never a recreate) when `Reader.Stats()` deltas show the reader is still making fetch attempts, and reserve `errFetchWedged` + recreate for **no-progress** ticks. Raise default `maxWait` 50ms→10s (kafka-go's own default; `MinBytes=1` means zero delivery-latency cost) and lower default `fetchTimeout` 5m→1m (now a cheap liveness tick). An integration harness reproduces the pre-fix dwell (H1 rebalance churn) and pins post-fix latency bounds; `findings.md` records the attribution.

**Tech Stack:** Go 1.25, `segmentio/kafka-go v0.4.51`, `testcontainers-go/modules/kafka v0.43.0` (`confluentinc/cp-kafka:7.6.0`), testify, logrus test hooks.

## Global Constraints

- Latency target: under the harness's modeled fan-out, p99 publish→handler latency **< 1s** (PRD §8).
- The `Config` decorator API (`SetMaxWait`, `SetFetchTimeout`, `SetMaxConsecutiveTimeouts`, `SetMaxInFlight`, `SetStartOffset`, `SetHeaderParsers`) keeps its exact signatures; per-consumer overrides keep working (PRD §4.4).
- At-least-once delivery, serial in-order commit, and the parallel loop's prefix-commit cursor are unchanged (PRD §4.5).
- `Snapshot` / debug-route changes are **additive only** (PRD §5).
- No broker manifest / `deploy/k8s` changes (PRD §2 non-goal).
- Only `libs/atlas-kafka` source changes; no per-service code changes (PRD §2).
- The lib owns the reader's `Stats()` delta stream — no other code may call `Stats()` on a lib-owned reader (design R4).
- Existing unit tests in `libs/atlas-kafka/consumer` must pass **unmodified** (mock readers don't implement `Stats()`, so they take the legacy counting path by design).
- Integration tests carry `//go:build integration` (they are NOT run by CI or plain `go test`; verified: no workflow passes `-tags integration`). Between Task 2 and Task 6, integration scenarios S2/S4 are expected-red — that IS the pre-fix reproduction. Unit suites stay green at every commit.
- No `// TODO` / stubs in landed commits. Committed files use repo-relative paths only.
- Verification bar (CLAUDE.md): `go test -race ./...`, `go vet ./...` clean in `libs/atlas-kafka`; `tools/redis-key-guard.sh` clean from worktree root; `docker buildx bake all-go-services` (shared-lib change touches every Go service image).

---

### Task 1: Phase-timing instrumentation on Consumer/Snapshot

Attribution counters (design §4.2) so the harness can attribute dwell to a phase: time in `FetchMessage` (last/max), reader-create→first-successful-fetch (join/assignment cost), cumulative recreate-backoff time, handler-dispatch duration (last/max).

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go`
- Modify: `libs/atlas-kafka/consumer/debug.go`
- Create: `libs/atlas-kafka/consumer/timing_test.go`

**Interfaces:**
- Consumes: existing `Consumer`, `Snapshot`, `scriptedReader`/`readerFactory` test helpers (`manager_test.go:419-501`).
- Produces: `Snapshot` fields `LastFetchDuration, MaxFetchDuration, TimeToFirstFetch, LastHandlerDuration, MaxHandlerDuration, TotalBackoff` (all `time.Duration`); test helper `snapshotForTopic(t *testing.T, cm *consumer.Manager, topic string) consumer.Snapshot` in `timing_test.go` (package `consumer_test`) — Tasks 2, 4, 6 reuse both.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-kafka/consumer/timing_test.go`:

```go
package consumer_test

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"
)

// snapshotForTopic returns the Snapshot of the registered consumer for topic,
// failing the test if none exists. Shared by timing and idle/stuck tests.
func snapshotForTopic(t *testing.T, cm *consumer.Manager, topic string) consumer.Snapshot {
	t.Helper()
	for _, c := range cm.Consumers() {
		s := c.Snapshot()
		if s.Topic == topic {
			return s
		}
	}
	t.Fatalf("no consumer registered for topic %s", topic)
	return consumer.Snapshot{}
}

// TestSnapshotPhaseTimings drives one recreate (io.EOF) followed by one
// handled message and asserts every phase-timing field is populated:
// TotalBackoff from the recreate wait, TimeToFirstFetch on the second
// reader, LastFetchDuration from the successful fetch, and handler
// durations from a deliberately slow handler.
func TestSnapshotPhaseTimings(t *testing.T) {
	consumer.ResetInstance()
	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &scriptedReader{script: []scriptedFetch{{err: io.EOF}}}
	r2 := &scriptedReader{script: []scriptedFetch{{msg: kafka.Message{Value: []byte("timed")}}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "timing-consumer", "timing-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(c)

	handled := make(chan struct{})
	_, _ = cm.RegisterHandler("timing-topic", func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
		time.Sleep(30 * time.Millisecond)
		close(handled)
		return true, nil
	})

	select {
	case <-handled:
	case <-time.After(5 * time.Second):
		t.Fatal("message was never handled after recreate")
	}

	// Handler duration is recorded after processMessage returns; poll
	// briefly for the snapshot to reflect it.
	var s consumer.Snapshot
	deadline := time.Now().Add(2 * time.Second)
	for {
		s = snapshotForTopic(t, cm, "timing-topic")
		if s.MaxHandlerDuration > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if s.TotalBackoff < 500*time.Millisecond {
		t.Fatalf("expected TotalBackoff >= 500ms (one recreate backoff), got %v", s.TotalBackoff)
	}
	if s.TimeToFirstFetch <= 0 {
		t.Fatalf("expected TimeToFirstFetch > 0 after a successful fetch, got %v", s.TimeToFirstFetch)
	}
	if s.LastFetchDuration <= 0 {
		t.Fatalf("expected LastFetchDuration > 0, got %v", s.LastFetchDuration)
	}
	if s.MaxFetchDuration < s.LastFetchDuration {
		t.Fatalf("expected MaxFetchDuration >= LastFetchDuration, got %v < %v", s.MaxFetchDuration, s.LastFetchDuration)
	}
	if s.MaxHandlerDuration < 30*time.Millisecond {
		t.Fatalf("expected MaxHandlerDuration >= 30ms (sleeping handler), got %v", s.MaxHandlerDuration)
	}
	if s.LastHandlerDuration < 30*time.Millisecond {
		t.Fatalf("expected LastHandlerDuration >= 30ms, got %v", s.LastHandlerDuration)
	}

	cancel()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run TestSnapshotPhaseTimings -v`
Expected: FAIL to compile — `s.TotalBackoff undefined (type consumer.Snapshot has no field or method TotalBackoff)` (and the other new fields).

- [ ] **Step 3: Implement the instrumentation**

In `libs/atlas-kafka/consumer/manager.go`:

3a. Add fields to the `Consumer` struct's observable-state block (after `lastTimeoutAt time.Time`):

```go
	// Phase-timing attribution — protected by mu. Durations are monotonic
	// deltas around existing call sites; they exist so a dwell can be
	// attributed to a phase (fetch wait, group join, recreate backoff,
	// handler dispatch) via Snapshot without a profiler.
	readerCreatedAt    time.Time
	awaitingFirstFetch bool
	timeToFirstFetch   time.Duration
	lastFetchDuration  time.Duration
	maxFetchDuration   time.Duration
	lastHandlerDuration time.Duration
	maxHandlerDuration  time.Duration
	totalBackoff        time.Duration
```

3b. Add fields to `Snapshot` (after `ConsecutiveTimeouts int`):

```go
	TimeToFirstFetch    time.Duration
	LastFetchDuration   time.Duration
	MaxFetchDuration    time.Duration
	LastHandlerDuration time.Duration
	MaxHandlerDuration  time.Duration
	TotalBackoff        time.Duration
```

and populate them in `(c *Consumer) Snapshot()`:

```go
		TimeToFirstFetch:    c.timeToFirstFetch,
		LastFetchDuration:   c.lastFetchDuration,
		MaxFetchDuration:    c.maxFetchDuration,
		LastHandlerDuration: c.lastHandlerDuration,
		MaxHandlerDuration:  c.maxHandlerDuration,
		TotalBackoff:        c.totalBackoff,
```

3c. Stamp reader creation in `onReaderCreated` (inside the existing lock, before the `if attempt > 0` branch):

```go
	c.readerCreatedAt = time.Now()
	c.awaitingFirstFetch = true
```

3d. Extend `recordFetch` to capture join/assignment cost:

```go
func (c *Consumer) recordFetch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.lastFetchAt = now
	c.lastError = ""
	c.consecutiveTimeouts = 0
	if c.awaitingFirstFetch {
		c.timeToFirstFetch = now.Sub(c.readerCreatedAt)
		c.awaitingFirstFetch = false
	}
}
```

3e. Add three recorders (near `recordError`):

```go
func (c *Consumer) recordFetchDuration(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFetchDuration = d
	if d > c.maxFetchDuration {
		c.maxFetchDuration = d
	}
}

func (c *Consumer) recordHandlerDuration(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastHandlerDuration = d
	if d > c.maxHandlerDuration {
		c.maxHandlerDuration = d
	}
}

func (c *Consumer) recordBackoff(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalBackoff += d
}
```

3f. In `runFetchLoopSerial`, wrap the fetch and the handler dispatch:

```go
		fetchCtx, cancelFetch := context.WithTimeout(ctx, c.fetchTimeout)
		fetchStart := time.Now()
		msg, err := reader.FetchMessage(fetchCtx)
		cancelFetch()
		c.recordFetchDuration(time.Since(fetchStart))
```

and:

```go
		c.recordFetch()
		l.Debugf("Message received %s.", string(msg.Value))
		handlerStart := time.Now()
		ok := c.processMessage(l, ctx, msg)
		c.recordHandlerDuration(time.Since(handlerStart))
		if ok {
			if cerr := reader.CommitMessages(ctx, msg); cerr != nil {
				l.WithError(cerr).Warnf("Could not commit message offset, it may be redelivered.")
			}
		}
```

3g. In `runFetchLoopParallel`, apply the same two wraps: `fetchStart := time.Now()` before `reader.FetchMessage(fetchCtx)` with `c.recordFetchDuration(time.Since(fetchStart))` after `cancelFetch()`, and inside the dispatch goroutine:

```go
		go func(p *pending) {
			defer func() { <-sem }()
			handlerStart := time.Now()
			ok := c.processMessage(l, ctx, p.msg)
			c.recordHandlerDuration(time.Since(handlerStart))
			p.ok.Store(ok)
			p.done.Store(true)
			advanceCommit()
		}(pm)
```

3h. In `start`, capture the backoff wait:

```go
		c.recordError(err)
		l.WithError(err).Errorf("Fetcher exited; recreating reader after backoff.")
		wait := backoff.next()
		select {
		case <-ctx.Done():
			l.Infof("Topic consumer stopped during backoff.")
			return
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
```

3i. In `libs/atlas-kafka/consumer/debug.go`, add to `debugAttributes` (JSON durations serialize as integer nanoseconds — suffix the keys accordingly):

```go
	TimeToFirstFetchNs    time.Duration `json:"timeToFirstFetchNs"`
	LastFetchDurationNs   time.Duration `json:"lastFetchDurationNs"`
	MaxFetchDurationNs    time.Duration `json:"maxFetchDurationNs"`
	LastHandlerDurationNs time.Duration `json:"lastHandlerDurationNs"`
	MaxHandlerDurationNs  time.Duration `json:"maxHandlerDurationNs"`
	TotalBackoffNs        time.Duration `json:"totalBackoffNs"`
```

and map them in `snapshotToAttributes`:

```go
		TimeToFirstFetchNs:    s.TimeToFirstFetch,
		LastFetchDurationNs:   s.LastFetchDuration,
		MaxFetchDurationNs:    s.MaxFetchDuration,
		LastHandlerDurationNs: s.LastHandlerDuration,
		MaxHandlerDurationNs:  s.MaxHandlerDuration,
		TotalBackoffNs:        s.TotalBackoff,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-kafka && go test -race ./consumer/ -v`
Expected: `TestSnapshotPhaseTimings` PASS and every pre-existing test PASS (the existing `debug_test.go` mirror struct ignores unknown JSON fields, so it needs no change).

- [ ] **Step 5: Vet and commit**

```bash
cd libs/atlas-kafka && go vet ./...
cd ../..
git add libs/atlas-kafka/consumer/manager.go libs/atlas-kafka/consumer/debug.go libs/atlas-kafka/consumer/timing_test.go
git commit -m "feat(atlas-kafka): phase-timing attribution on consumer snapshots"
```

---

### Task 2: Dwell reproduction & attribution harness (integration)

The testcontainers harness (design §4.1) modeling the live fan-out: one group with 15 idle + 1 active topic (atlas-saga-orchestrator's shape), a second group for coordinator sharing. Scenarios S1–S5. **S2 and S4 assert post-fix bounds and are expected to FAIL until Task 4 lands — that failure is the deterministic pre-fix reproduction Task 3 captures.**

**Files:**
- Create: `libs/atlas-kafka/consumer/dwell_integration_test.go`

**Interfaces:**
- Consumes: Task 1's `Snapshot` phase-timing fields; existing testcontainers pattern (`offsets_test.go`); `consumer.ConfigReaderProducer`, `consumer.GetManager`, `consumer.ResetInstance`, `Manager.Consumers()`.
- Produces: test functions `TestDwellS1_SteadyStateLatency`, `TestDwellS2_IdleTickChurn`, `TestDwellS3_ForcedRecreateBounded`, `TestDwellS4_TickControl`, `TestDwellS5_MaxWaitIdleFetchRate`; helpers `startDwellKafka`, `createDwellTopics`, `latencyRecorder`, `publishStamped`, `dumpSnapshots` (Task 6 edits S2 to add an `IdleTicks` assertion once that field exists).

- [ ] **Step 1: Write the harness**

Create `libs/atlas-kafka/consumer/dwell_integration_test.go`:

```go
//go:build integration

package consumer_test

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

// The dwell harness models the live conditions behind the task-136 incident:
// a single broker, one consumer group with many idle topics plus one active
// topic (atlas-saga-orchestrator registers 15 consumers under one GroupID),
// and a second group sharing the coordinator. Each scenario measures
// publish→handler latency end-to-end: the publisher stamps send-time (unix
// nanos) into the message value and the handler records time.Since(stamp).
//
// S2/S4 assert POST-fix bounds (no self-recreate on idle, p99 < 1s). On
// pre-fix code they fail with the reproduced dwell — that failure is the
// baseline capture for findings.md, per the design (§4.1).

const dwellActiveTopic = "dwell.active"

func startDwellKafka(t *testing.T) []string {
	t.Helper()
	ctx := context.Background()
	kc, err := tckafka.Run(ctx, "confluentinc/cp-kafka:7.6.0", tckafka.WithClusterID("atlas-dwell"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = kc.Terminate(context.Background()) })
	brokers, err := kc.Brokers(ctx)
	require.NoError(t, err)
	return brokers
}

func createDwellTopics(t *testing.T, brokers []string, topics []string) {
	t.Helper()
	conn, err := (&kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}).
		DialContext(context.Background(), "tcp", brokers[0])
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	cfgs := make([]kafka.TopicConfig, 0, len(topics))
	for _, tp := range topics {
		cfgs = append(cfgs, kafka.TopicConfig{Topic: tp, NumPartitions: 1, ReplicationFactor: 1})
	}
	require.NoError(t, conn.CreateTopics(cfgs...))
}

func idleTopics(prefix string, n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, fmt.Sprintf("%s.idle.%d", prefix, i))
	}
	return out
}

type latencyRecorder struct {
	mu        sync.Mutex
	latencies []time.Duration
}

func (r *latencyRecorder) record(msg kafka.Message) {
	ns, err := strconv.ParseInt(string(msg.Value), 10, 64)
	if err != nil {
		return
	}
	d := time.Since(time.Unix(0, ns))
	r.mu.Lock()
	r.latencies = append(r.latencies, d)
	r.mu.Unlock()
}

func (r *latencyRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.latencies)
}

func (r *latencyRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.latencies = nil
}

func (r *latencyRecorder) sorted() []time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := append([]time.Duration(nil), r.latencies...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s
}

func (r *latencyRecorder) p99() time.Duration {
	s := r.sorted()
	if len(s) == 0 {
		return 0
	}
	idx := int(float64(len(s))*0.99) - 1
	if idx < 0 {
		idx = 0
	}
	return s[idx]
}

func (r *latencyRecorder) max() time.Duration {
	s := r.sorted()
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}

// publishStamped writes n messages whose value is the send-time in unix
// nanoseconds, waiting interval between sends. WriteMessages blocks until
// the broker acks (RequireAll), so the stamp precedes broker persistence by
// at most the 10ms batch timeout.
func publishStamped(t *testing.T, brokers []string, topic string, n int, interval time.Duration) {
	t.Helper()
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireAll,
	}
	defer func() { _ = w.Close() }()
	for i := 0; i < n; i++ {
		msg := kafka.Message{
			Key:   []byte(fmt.Sprintf("k%d", i)),
			Value: []byte(strconv.FormatInt(time.Now().UnixNano(), 10)),
		}
		require.NoError(t, w.WriteMessages(context.Background(), msg))
		if interval > 0 {
			time.Sleep(interval)
		}
	}
}

func dumpSnapshots(t *testing.T, cm *consumer.Manager) {
	t.Helper()
	for _, c := range cm.Consumers() {
		t.Logf("snapshot: %+v", c.Snapshot())
	}
}

func totalRecreates(cm *consumer.Manager) int {
	total := 0
	for _, c := range cm.Consumers() {
		total += c.Snapshot().RecreateCount
	}
	return total
}

// dwellSetup boots the manager with the modeled topology and a latency
// recorder on the active topic. idleDecorators apply to the idle-topic
// consumers only (S2 uses them to compress the wedge cadence).
func dwellSetup(t *testing.T, brokers []string, idleCount int, otherCount int,
	rp []consumer.ManagerConfig, idleDecorators ...model.Decorator[consumer.Config],
) (*consumer.Manager, *latencyRecorder, context.CancelFunc, *sync.WaitGroup) {
	t.Helper()
	idle := idleTopics("dwell", idleCount)
	other := idleTopics("other", otherCount)
	all := append(append([]string{}, idle...), dwellActiveTopic)
	createDwellTopics(t, brokers, append(all, other...))

	consumer.ResetInstance()
	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	cm := consumer.GetManager(rp...)
	add := cm.AddConsumer(l, ctx, wg)
	for _, tp := range idle {
		add(consumer.NewConfig(brokers, "dwell-"+tp, tp, "dwell-svc"), idleDecorators...)
	}
	add(consumer.NewConfig(brokers, "dwell-active", dwellActiveTopic, "dwell-svc"))
	for _, tp := range other {
		add(consumer.NewConfig(brokers, "other-"+tp, tp, "other-svc"))
	}

	rec := &latencyRecorder{}
	_, err := cm.RegisterHandler(dwellActiveTopic, func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		rec.record(msg)
		return true, nil
	})
	require.NoError(t, err)

	// Warm-up: group join for ~20 members can take tens of seconds; the
	// first delivery proves assignment settled and is excluded from
	// measurement.
	publishStamped(t, brokers, dwellActiveTopic, 1, 0)
	require.Eventually(t, func() bool { return rec.count() >= 1 },
		120*time.Second, 200*time.Millisecond, "warm-up message never delivered — group join did not settle")
	rec.reset()

	return cm, rec, cancel, wg
}

// S1 — steady state: full modeled fan-out, healthy consumers, production
// defaults. Asserts the PRD §8 target.
func TestDwellS1_SteadyStateLatency(t *testing.T) {
	brokers := startDwellKafka(t)
	cm, rec, cancel, wg := dwellSetup(t, brokers, 15, 4, nil)
	defer func() { cancel(); wg.Wait() }()

	const n = 100
	publishStamped(t, brokers, dwellActiveTopic, n, 100*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= n },
		60*time.Second, 100*time.Millisecond, "not all messages delivered")

	p99 := rec.p99()
	t.Logf("S1: p99=%v max=%v over %d messages", p99, rec.max(), rec.count())
	dumpSnapshots(t, cm)
	require.Less(t, p99, time.Second, "S1: steady-state p99 publish→handler latency must be < 1s (PRD §8)")
}

// S2 — idle-tick churn (H1). Short fetchTimeout + low threshold on the 15
// idle consumers compresses the legacy 3×5m wedge cadence into seconds:
// pre-fix, every idle consumer self-wedges every ~4s, each Close() sends
// LeaveGroup and rebalances the whole group — including the active topic's
// member — reproducing the live dwell. Post-fix these deadlines are idle
// ticks: zero self-recreates, latency stays at the S1 bound.
func TestDwellS2_IdleTickChurn(t *testing.T) {
	brokers := startDwellKafka(t)
	cm, rec, cancel, wg := dwellSetup(t, brokers, 15, 0, nil,
		consumer.SetFetchTimeout(2*time.Second),
		consumer.SetMaxConsecutiveTimeouts(2),
	)
	defer func() { cancel(); wg.Wait() }()

	const n = 30
	publishStamped(t, brokers, dwellActiveTopic, n, time.Second)
	require.Eventually(t, func() bool { return rec.count() >= n },
		120*time.Second, 100*time.Millisecond, "not all messages delivered")

	p99 := rec.p99()
	t.Logf("S2: p99=%v max=%v recreates=%d", p99, rec.max(), totalRecreates(cm))
	dumpSnapshots(t, cm)
	require.Zero(t, totalRecreates(cm), "S2: idle deadline ticks must not recreate readers (design §3-A)")
	require.Less(t, p99, time.Second, "S2: churn-free p99 must be < 1s (PRD §8)")
}

// forceErrReader wraps a real reader; the test arms a one-shot injected
// fetch error to drive the genuine recreate path (close → LeaveGroup →
// backoff → recreate → rejoin) on the active topic mid-stream.
type forceErrReader struct {
	inner consumer.KafkaReader
	arm   *atomic.Bool
}

func (r *forceErrReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if r.arm.CompareAndSwap(true, false) {
		return kafka.Message{}, fmt.Errorf("injected fetch failure")
	}
	return r.inner.FetchMessage(ctx)
}

func (r *forceErrReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return r.inner.CommitMessages(ctx, msgs...)
}

func (r *forceErrReader) Close() error { return r.inner.Close() }

// Stats delegates so idle-vs-stuck classification sees the real reader's
// progress. The anonymous interface avoids depending on the lib's
// StatsProvider name.
func (r *forceErrReader) Stats() kafka.ReaderStats {
	if sp, ok := r.inner.(interface{ Stats() kafka.ReaderStats }); ok {
		return sp.Stats()
	}
	return kafka.ReaderStats{}
}

// S3 — bounded recreate: a genuine failure on the active topic's reader
// forces one recreate mid-stream; delivery must resume with the recreate
// dwell bounded by the join+backoff budget (design §4.1: ≤ 10s).
func TestDwellS3_ForcedRecreateBounded(t *testing.T) {
	brokers := startDwellKafka(t)
	var arm atomic.Bool
	rp := consumer.ConfigReaderProducer(func(cfg kafka.ReaderConfig) consumer.KafkaReader {
		inner := kafka.NewReader(cfg)
		if cfg.Topic == dwellActiveTopic {
			return &forceErrReader{inner: inner, arm: &arm}
		}
		return inner
	})
	cm, rec, cancel, wg := dwellSetup(t, brokers, 5, 0, []consumer.ManagerConfig{rp})
	defer func() { cancel(); wg.Wait() }()

	// Healthy stretch first.
	publishStamped(t, brokers, dwellActiveTopic, 10, 100*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= 10 },
		30*time.Second, 100*time.Millisecond, "pre-recreate messages not delivered")
	rec.reset()

	// Inject the failure, then keep publishing across the recreate window.
	arm.Store(true)
	publishStamped(t, brokers, dwellActiveTopic, 20, 250*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= 20 },
		60*time.Second, 100*time.Millisecond, "messages lost across recreate")

	maxDwell := rec.max()
	active := snapshotForTopic(t, cm, dwellActiveTopic)
	t.Logf("S3: max dwell across recreate=%v recreates=%d timeToFirstFetch=%v totalBackoff=%v",
		maxDwell, active.RecreateCount, active.TimeToFirstFetch, active.TotalBackoff)
	dumpSnapshots(t, cm)
	require.GreaterOrEqual(t, active.RecreateCount, 1, "S3: injected error must have forced a recreate")
	require.LessOrEqual(t, maxDwell, 10*time.Second, "S3: recreate dwell must be bounded (design §4.1)")
}

// S4 — control (H3): deadline ticks at 2s cadence on a small healthy group.
// Post-fix, ticks alone add no dwell and cause no recreates — closing PRD
// §9 Q2 with measurement rather than only the kafka-go source citation.
func TestDwellS4_TickControl(t *testing.T) {
	brokers := startDwellKafka(t)
	cm, rec, cancel, wg := dwellSetup(t, brokers, 2, 0, nil,
		consumer.SetFetchTimeout(2*time.Second),
	)
	defer func() { cancel(); wg.Wait() }()

	const n = 30
	publishStamped(t, brokers, dwellActiveTopic, n, 500*time.Millisecond)
	require.Eventually(t, func() bool { return rec.count() >= n },
		60*time.Second, 100*time.Millisecond, "not all messages delivered")

	p99 := rec.p99()
	t.Logf("S4: p99=%v max=%v recreates=%d", p99, rec.max(), totalRecreates(cm))
	dumpSnapshots(t, cm)
	require.Zero(t, totalRecreates(cm), "S4: ticks alone must not recreate")
	require.Less(t, p99, time.Second, "S4: ticks alone must add no dwell")
}

// S5 — MaxWait A/B (H2): an idle group reader at maxWait=50ms vs 10s. With
// MinBytes=1 the broker answers immediately when data exists, so MaxWait
// only bounds the empty long-poll — the 50ms setting buys no latency and
// multiplies idle fetch traffic. Raw kafka.Readers are used (not the lib)
// so the test may consume Stats() deltas itself; findings.md extrapolates
// the per-reader rates to the live ~481 partitions.
func TestDwellS5_MaxWaitIdleFetchRate(t *testing.T) {
	brokers := startDwellKafka(t)
	createDwellTopics(t, brokers, []string{"s5.idle.a", "s5.idle.b"})

	measure := func(topic string, maxWait time.Duration) int64 {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: "s5-" + topic,
			MaxWait: maxWait,
		})
		defer func() { _ = r.Close() }()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// Drive the reader; FetchMessage blocks the full window on an idle
		// topic while the background loop long-polls the broker.
		_, _ = r.FetchMessage(ctx)
		return r.Stats().Fetches
	}

	fast := measure("s5.idle.a", 50*time.Millisecond)
	slow := measure("s5.idle.b", 10*time.Second)
	t.Logf("S5: idle fetch attempts in 30s — maxWait=50ms: %d; maxWait=10s: %d", fast, slow)
	require.Greater(t, fast, slow, "S5: shorter MaxWait must issue more idle fetch requests")
}
```

- [ ] **Step 2: Verify the harness compiles and the unit suite is untouched**

Run: `cd libs/atlas-kafka && go vet -tags integration ./consumer/ && go test -race ./consumer/`
Expected: vet clean (integration file compiles); unit tests all PASS (harness is excluded without the tag).

- [ ] **Step 3: Smoke one cheap scenario end-to-end**

Run: `cd libs/atlas-kafka && go test -tags integration -run TestDwellS5_MaxWaitIdleFetchRate -v -timeout 10m ./consumer/`
Expected: PASS with a logged line like `S5: idle fetch attempts in 30s — maxWait=50ms: <hundreds>; maxWait=10s: <single digits>`. (Requires local Docker. If `fast` is not ≫ `slow`, stop and investigate before proceeding — H2's premise would be wrong.)

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-kafka/consumer/dwell_integration_test.go
git commit -m "test(atlas-kafka): dwell reproduction and attribution harness (integration)"
```

---

### Task 3: Pre-fix baseline capture → findings.md draft

Run the harness against the current (pre-fix) loop to capture the reproduced dwell with phase attribution. **S2 is expected to FAIL here — its failure output IS the baseline evidence.** This task runs before any fix lands (design R1: reproduce, then fix).

**Files:**
- Create: `docs/tasks/task-136-consumer-fetch-wedge/findings.md`

**Interfaces:**
- Consumes: Task 2's scenario tests and their `t.Logf` output (latency percentiles, `totalRecreates`, snapshot dumps).
- Produces: `findings.md` with a completed "Pre-fix baseline" section; Task 6 fills the post-fix sections.

- [ ] **Step 1: Run the pre-fix baseline scenarios**

Run: `cd libs/atlas-kafka && go test -tags integration -run 'TestDwellS1_|TestDwellS2_|TestDwellS4_|TestDwellS5_' -v -timeout 45m ./consumer/ 2>&1 | tee /tmp/task-136-prefix-baseline.log`

Expected outcomes (record the actual numbers, whatever they are):
- S1 PASS (steady state without ticks firing shows no dwell — the pre-fix bug needs the wedge cadence, which 5m defaults don't reach in-test).
- S2 **FAIL** on `totalRecreates == 0` (pre-fix idle consumers self-wedge and recreate) and likely on the p99 bound, with logged p99/max dwell and per-consumer snapshots showing `RecreateCount > 0` and `TotalBackoff > 0`.
- S4 **FAIL** on `totalRecreates == 0` (same mechanism, small scale).
- S5 PASS with the two fetch-rate numbers.

If S2 passes pre-fix (dwell does not reproduce), STOP — do not proceed to Task 4. Per design R1 the investigation must widen (coordinator contention, second-group interaction) before committing to the fix. Report the observed numbers and escalate.

- [ ] **Step 2: Write the findings draft**

Create `docs/tasks/task-136-consumer-fetch-wedge/findings.md` with the skeleton below, filling every `<...>` from the Step 1 log (quote actual numbers — no paraphrasing):

```markdown
# task-136 findings — Kafka consumer dwell root cause

## Reproduction environment

Harness: `libs/atlas-kafka/consumer/dwell_integration_test.go` (build tag
`integration`), single-broker testcontainers Kafka (`confluentinc/cp-kafka:7.6.0`),
one group with 15 idle topics + 1 active topic (models atlas-saga-orchestrator's
fan-out) plus a 4-consumer second group. Latency = publish→handler, stamped
in-message.

## Pre-fix baseline (commit <git rev-parse --short HEAD>)

| Scenario | Result | p99 | max | total recreates | notes |
|---|---|---|---|---|---|
| S1 steady state | <PASS/FAIL> | <v> | <v> | <n> | ticks never fire at 5m defaults |
| S2 idle-tick churn | FAIL (expected) | <v> | <v> | <n> | wedge cadence compressed to 2s×2 |
| S4 tick control | <result> | <v> | <v> | <n> | |
| S5 fetch rate 50ms vs 10s | <fast> vs <slow> attempts/30s | — | — | — | |

Phase attribution (S2 snapshot dump, active-topic consumer):
- TimeToFirstFetch: <v> (join/assignment cost per recreate)
- TotalBackoff: <v>
- MaxFetchDuration: <v>
- MaxHandlerDuration: <v> (H4 check — expected negligible)

## Hypothesis verdicts

- **H1 (wedge-recreate churn → group-wide rebalance storms):** <verdict with
  S2 numbers — recreates observed, dwell inflation, attribution>
- **H2 (50ms MaxWait idle-spin):** S5 measured <fast> vs <slow> fetch attempts
  per idle reader per 30s. Extrapolated to ~481 live partitions: <n>/s vs <n>/s.
- **H3 (deadline drops group session — refuted in source):** S4 control:
  <ticks alone added/did not add dwell pre-fix beyond the recreate mechanism>.
- **H4 (head-of-line blocking):** handler dispatch time <v> — <verdict>.

## Client-side vs broker-side split

<From the pre-fix numbers: how much dwell is attributable to client-side
churn (H1) vs broker fetch latency. The harness broker is unloaded, so
dwell reproduced here is client-side by construction; note this.>

## Post-fix results

(Completed in the post-fix run task.)

## Config default changes & rationale

(Completed in the post-fix run task.)

## Follow-up decision (design §7 gate)

(Completed in the post-fix run task.)
```

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-136-consumer-fetch-wedge/findings.md
git commit -m "docs(task-136): pre-fix dwell baseline findings"
```

---

### Task 4: Idle-vs-stuck tick classification (the fix)

Design §4.3: an expired fetch deadline on a reader that is still making fetch attempts is an **idle tick** — debug log, no recreate, resets the escalation counter. Only **no-progress** ticks (zero `Fetches`/`Dials`/`Messages` in the `Stats()` delta) count toward `maxConsecutiveTimeouts` and trigger `errFetchWedged`. Mock readers without `Stats()` keep legacy counting, so every existing unit test passes unmodified.

**Files:**
- Modify: `libs/atlas-kafka/consumer/manager.go`
- Modify: `libs/atlas-kafka/consumer/debug.go`
- Create: `libs/atlas-kafka/consumer/idle_stuck_test.go`

**Interfaces:**
- Consumes: Task 1's `snapshotForTopic` helper; existing `readerFactory`; `kafka.ReaderStats` (`Fetches`, `Dials`, `Messages` are int64 delta counters — `Reader.Stats()` returns deltas since the previous call, kafka-go `reader.go:1089-1096`).
- Produces: exported `StatsProvider` interface; `Snapshot` fields `IdleTicks int`, `LastIdleTickAt time.Time`, `NoProgressTicks int`, `LastNoProgressAt time.Time`. `ConsecutiveTimeouts`/`LastTimeoutAt` keep their names but now count only no-progress ticks.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-kafka/consumer/idle_stuck_test.go`:

```go
package consumer_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"go.opentelemetry.io/otel"
)

// statsStubReader blocks every FetchMessage on ctx (so each call expires the
// per-call deadline) and reports scripted ReaderStats deltas: entries in
// deltas are cycled per Stats() call; an empty slice means a permanent
// zero delta (no progress).
type statsStubReader struct {
	mu     sync.Mutex
	deltas []kafka.ReaderStats
	i      int
	closes int
}

func (r *statsStubReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (r *statsStubReader) CommitMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}

func (r *statsStubReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes++
	return nil
}

func (r *statsStubReader) Closes() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closes
}

func (r *statsStubReader) Stats() kafka.ReaderStats {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.deltas) == 0 {
		return kafka.ReaderStats{}
	}
	d := r.deltas[r.i%len(r.deltas)]
	r.i++
	return d
}

// Compile-time check: the stub must satisfy the new StatsProvider.
var _ consumer.StatsProvider = (*statsStubReader)(nil)

// TestIdleTickNeverWedges: a reader that keeps making fetch attempts
// (Fetches delta > 0) ticks through many deadlines without a warn log,
// without recreating, and with ConsecutiveTimeouts pinned at 0.
func TestIdleTickNeverWedges(t *testing.T) {
	consumer.ResetInstance()
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &statsStubReader{deltas: []kafka.ReaderStats{{Fetches: 3}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "idle-consumer", "idle-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(30*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	// >10 deadline ticks — far past the legacy wedge threshold of 3.
	time.Sleep(400 * time.Millisecond)

	s := snapshotForTopic(t, cm, "idle-topic")
	if s.RecreateCount != 0 {
		t.Fatalf("expected no recreates on an idle-but-healthy reader, got %d", s.RecreateCount)
	}
	if r1.Closes() != 0 {
		t.Fatalf("expected reader never closed while idle-healthy, got %d closes", r1.Closes())
	}
	if s.IdleTicks < 3 {
		t.Fatalf("expected IdleTicks >= 3, got %d", s.IdleTicks)
	}
	if s.LastIdleTickAt.IsZero() {
		t.Fatal("expected LastIdleTickAt to be set")
	}
	if s.ConsecutiveTimeouts != 0 {
		t.Fatalf("expected ConsecutiveTimeouts == 0 (idle ticks reset it), got %d", s.ConsecutiveTimeouts)
	}
	if s.NoProgressTicks != 0 {
		t.Fatalf("expected NoProgressTicks == 0, got %d", s.NoProgressTicks)
	}
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			t.Fatalf("expected no Warn logs for idle ticks, got %q", e.Message)
		}
	}

	cancel()
}

// TestNoProgressTicksEscalateToWedge: a reader whose Stats() delta shows
// zero progress across the threshold count is recreated, with the wedge
// warn naming topic and group.
func TestNoProgressTicksEscalateToWedge(t *testing.T) {
	consumer.ResetInstance()
	l, hook := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &statsStubReader{} // permanent zero delta: stuck
	r2 := &statsStubReader{deltas: []kafka.ReaderStats{{Fetches: 1}}}
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1, r2))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "stuck-consumer", "stuck-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(30*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(3),
	)

	deadline := time.Now().Add(5 * time.Second)
	for r1.Closes() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if r1.Closes() == 0 {
		t.Fatal("stuck reader was never recreated")
	}

	s := snapshotForTopic(t, cm, "stuck-topic")
	if s.RecreateCount < 1 {
		t.Fatalf("expected RecreateCount >= 1 after a genuine stall, got %d", s.RecreateCount)
	}
	if s.NoProgressTicks < 3 {
		t.Fatalf("expected NoProgressTicks >= 3, got %d", s.NoProgressTicks)
	}
	if s.LastNoProgressAt.IsZero() {
		t.Fatal("expected LastNoProgressAt to be set")
	}

	foundWedgeWarn := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel &&
			strings.Contains(e.Message, "FetchMessage wedged") &&
			strings.Contains(e.Message, "stuck-topic") &&
			strings.Contains(e.Message, "test-group") {
			foundWedgeWarn = true
			break
		}
	}
	if !foundWedgeWarn {
		t.Fatalf("expected a Warn log containing 'FetchMessage wedged' with topic+group, got: %v", hook.AllEntries())
	}

	cancel()
}

// TestIdleTickResetsNoProgressCount: alternating no-progress and progress
// ticks never reach the threshold — a transiently quiet reader that
// recovers is not recreated.
func TestIdleTickResetsNoProgressCount(t *testing.T) {
	consumer.ResetInstance()
	l, _ := test.NewNullLogger()
	wg := &sync.WaitGroup{}
	otel.SetTracerProvider(&MockTracerProvider{})

	r1 := &statsStubReader{deltas: []kafka.ReaderStats{{}, {Fetches: 1}}} // stuck, idle, stuck, idle...
	rp := consumer.ConfigReaderProducer(readerFactory(t, r1))

	ctx, cancel := context.WithCancel(context.Background())
	defer wg.Wait()

	cm := consumer.GetManager(rp)
	c := consumer.NewConfig([]string{""}, "flappy-consumer", "flappy-topic", "test-group")
	cm.AddConsumer(l, ctx, wg)(
		c,
		consumer.SetFetchTimeout(30*time.Millisecond),
		consumer.SetMaxConsecutiveTimeouts(2),
	)

	// >12 ticks alternating no-progress/progress: consecutive never hits 2.
	time.Sleep(400 * time.Millisecond)

	s := snapshotForTopic(t, cm, "flappy-topic")
	if s.RecreateCount != 0 {
		t.Fatalf("expected no recreate when progress interleaves, got %d", s.RecreateCount)
	}
	if r1.Closes() != 0 {
		t.Fatalf("expected reader never closed, got %d closes", r1.Closes())
	}
	if s.NoProgressTicks < 2 {
		t.Fatalf("expected cumulative NoProgressTicks >= 2, got %d", s.NoProgressTicks)
	}
	if s.IdleTicks < 2 {
		t.Fatalf("expected cumulative IdleTicks >= 2, got %d", s.IdleTicks)
	}

	cancel()
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestIdleTick|TestNoProgress' -v`
Expected: FAIL to compile — `undefined: consumer.StatsProvider`, `s.IdleTicks undefined`, etc.

- [ ] **Step 3: Implement the classification**

In `libs/atlas-kafka/consumer/manager.go`:

3a. Add the interface and progress predicate (below the `KafkaReader` interface declarations):

```go
// StatsProvider is implemented by readers that can report kafka-go reader
// statistics; *kafka.Reader satisfies it natively. The fetch loop uses
// Stats() deltas to distinguish an idle reader (still issuing fetch
// attempts against the broker) from a stuck one (no progress at all).
//
// OWNERSHIP: kafka-go's Stats() returns counter deltas since the previous
// call. This lib owns the reader's stats stream exclusively — nothing else
// may call Stats() on a lib-owned reader, or both callers see partial
// deltas. External metrics/telemetry must read Consumer.Snapshot() instead.
type StatsProvider interface {
	Stats() kafka.ReaderStats
}

// readerMadeProgress reports whether the reader has done any work since the
// previous deadline tick. Readers that don't expose Stats() (test mocks)
// are conservatively treated as making no progress — legacy behavior, where
// every deadline tick counts toward the wedge threshold.
func readerMadeProgress(reader KafkaReader) bool {
	sp, ok := reader.(StatsProvider)
	if !ok {
		return false
	}
	s := sp.Stats()
	return s.Fetches > 0 || s.Dials > 0 || s.Messages > 0
}
```

3b. Add observable fields to `Consumer` (in the observable-state block, after `lastTimeoutAt time.Time`):

```go
	idleTicks        int
	lastIdleTickAt   time.Time
	noProgressTicks  int
	lastNoProgressAt time.Time
```

3c. Replace `recordTimeout` with the two classified recorders (`idleTicks`/`noProgressTicks` are cumulative and deliberately NOT reset by `onReaderCreated` — only the consecutive counter is):

```go
// recordIdleTick marks one deadline expiration on a reader that is still
// making fetch attempts. Idle is healthy: it resets the no-progress
// escalation counter and touches no error state.
func (c *Consumer) recordIdleTick() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.idleTicks++
	c.lastIdleTickAt = time.Now()
	c.consecutiveTimeouts = 0
}

// recordNoProgressTick marks one deadline expiration with zero reader
// progress — a stall suspect. Returns the new consecutive count so callers
// can branch on the threshold without a second mutex acquisition.
func (c *Consumer) recordNoProgressTick() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.lastTimeoutAt = now
	c.lastNoProgressAt = now
	c.noProgressTicks++
	c.consecutiveTimeouts++
	return c.consecutiveTimeouts
}
```

3d. Add the shared deadline handler (used by both loops so they stay semantically aligned):

```go
// handleFetchDeadline classifies one expired fetch deadline: an idle tick
// (reader made progress — normal on a no-traffic topic) or a no-progress
// tick (stall suspect). Returns errFetchWedged once consecutive no-progress
// ticks reach the threshold, nil otherwise.
func (c *Consumer) handleFetchDeadline(l logrus.FieldLogger, reader KafkaReader) error {
	if readerMadeProgress(reader) {
		c.recordIdleTick()
		l.Debugf("Fetch deadline expired on idle topic [%s]; reader healthy, continuing.", c.topic)
		return nil
	}
	consecutive := c.recordNoProgressTick()
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

3e. In `runFetchLoopSerial`, replace the `DeadlineExceeded` branch body:

```go
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, reader); werr != nil {
					return werr
				}
				continue
			}
```

3f. In `runFetchLoopParallel`, replace its `DeadlineExceeded` branch body (keeping the `advanceCommit()` call):

```go
			if errors.Is(err, context.DeadlineExceeded) {
				if werr := c.handleFetchDeadline(l, reader); werr != nil {
					return werr
				}
				// In-flight goroutines may have completed; try to advance.
				advanceCommit()
				continue
			}
```

3g. Update the doc comment on `runFetchLoopSerial` (the old one documents recreate-on-idle): replace the paragraph starting "Each iteration runs FetchMessage under a per-call deadline" with:

```go
// Each iteration runs FetchMessage under a per-call deadline
// (c.fetchTimeout) that acts as a liveness tick. An expiration on a reader
// that is still making fetch attempts (per Stats() deltas) is an idle
// tick — healthy, never a recreate. Only ticks with zero reader progress
// count toward c.maxConsecutiveTimeouts; at the threshold the loop returns
// errFetchWedged so the outer start loop closes and recreates the reader.
// A successful fetch resets the counter via recordFetch.
```

3h. Add the fields to `Snapshot` (after Task 1's timing fields):

```go
	IdleTicks        int
	LastIdleTickAt   time.Time
	NoProgressTicks  int
	LastNoProgressAt time.Time
```

populate them in `(c *Consumer) Snapshot()`:

```go
		IdleTicks:        c.idleTicks,
		LastIdleTickAt:   c.lastIdleTickAt,
		NoProgressTicks:  c.noProgressTicks,
		LastNoProgressAt: c.lastNoProgressAt,
```

3i. In `libs/atlas-kafka/consumer/debug.go`, add to `debugAttributes`:

```go
	IdleTicks        int       `json:"idleTicks"`
	LastIdleTickAt   time.Time `json:"lastIdleTickAt"`
	NoProgressTicks  int       `json:"noProgressTicks"`
	LastNoProgressAt time.Time `json:"lastNoProgressAt"`
```

and map them in `snapshotToAttributes`:

```go
		IdleTicks:        s.IdleTicks,
		LastIdleTickAt:   s.LastIdleTickAt,
		NoProgressTicks:  s.NoProgressTicks,
		LastNoProgressAt: s.LastNoProgressAt,
```

- [ ] **Step 4: Run the full unit suite**

Run: `cd libs/atlas-kafka && go test -race ./consumer/ -v`
Expected: the three new tests PASS, and **every pre-existing test passes unmodified** — `scriptedReader`/`ChannelMockReader`/`alternatingReader`/`controlledReader` don't implement `Stats()`, so `TestFetchTimeoutTicksWithoutRecreate` and `TestFetchTimeoutEscalatesAfterMaxToWedge` exercise the legacy counting path, and the new wedge warn still contains the `FetchMessage wedged` / topic / group substrings that `TestFetchTimeoutEscalatesAfterMaxToWedge` greps for. If any existing test fails, the change is wrong — do not "fix" the test; fix the implementation.

- [ ] **Step 5: Vet and commit**

```bash
cd libs/atlas-kafka && go vet ./... && go vet -tags integration ./consumer/
cd ../..
git add libs/atlas-kafka/consumer/manager.go libs/atlas-kafka/consumer/debug.go libs/atlas-kafka/consumer/idle_stuck_test.go
git commit -m "feat(atlas-kafka): classify fetch-deadline ticks as idle vs no-progress; recreate only on genuine stalls"
```

---

### Task 5: Config default changes

Design §4.4: `maxWait` 50ms→10s (kafka-go's own default; `MinBytes=1` means the broker answers immediately when data exists, so the empty-long-poll bound costs zero delivery latency and ~200× less idle broker load), `fetchTimeout` 5m→1m (now a liveness-tick cadence, detecting a genuine stall in ~3m instead of ~15m). `maxConsecutiveTimeouts` stays 3.

**Files:**
- Modify: `libs/atlas-kafka/consumer/config.go`
- Modify: `libs/atlas-kafka/consumer/config_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: new default values; all decorator signatures unchanged.

- [ ] **Step 1: Update the default-value tests first**

In `libs/atlas-kafka/consumer/config_test.go`, in `TestConfig` replace:

```go
	if c.maxWait != time.Millisecond*50 {
		t.Fatalf("Invalid broker max wait.")
	}
```

with:

```go
	if c.maxWait != 10*time.Second {
		t.Fatalf("expected default maxWait=10s (kafka-go default; MinBytes=1 makes it latency-free), got %v", c.maxWait)
	}
```

and in `TestFetchTimeoutDefaultsAndOverride` replace:

```go
	if c.fetchTimeout != 5*time.Minute {
		t.Fatalf("expected default fetchTimeout=5m, got %v", c.fetchTimeout)
	}
```

with:

```go
	if c.fetchTimeout != time.Minute {
		t.Fatalf("expected default fetchTimeout=1m (liveness-tick cadence), got %v", c.fetchTimeout)
	}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestConfig|TestFetchTimeoutDefaultsAndOverride' -v`
Expected: FAIL — `expected default maxWait=10s ... got 50ms` and `expected default fetchTimeout=1m ... got 5m0s`.

- [ ] **Step 3: Change the defaults with documented rationale**

In `libs/atlas-kafka/consumer/config.go`, replace `NewConfig`:

```go
// NewConfig builds a consumer Config with the library defaults.
//
// Default rationale (task-136 — see docs/tasks/task-136-consumer-fetch-wedge/findings.md):
//
//   - maxWait 10s: kafka-go's own default. With MinBytes=1 (the kafka-go
//     default) the broker answers a fetch immediately when data exists;
//     MaxWait only bounds how long the broker parks an EMPTY long-poll, so
//     a large value costs zero delivery latency while cutting idle fetch
//     traffic ~200× vs the previous 50ms.
//   - fetchTimeout 1m: the per-call FetchMessage deadline is a liveness
//     tick, not a recreate trigger. A deadline expiration on a reader that
//     is still making fetch attempts is an idle tick (healthy); only ticks
//     with zero reader progress count toward maxConsecutiveTimeouts.
//   - maxConsecutiveTimeouts 3: consecutive NO-PROGRESS ticks before the
//     reader is declared wedged and recreated (~3m to detection at the
//     default tick interval).
func NewConfig(brokers []string, name string, topic string, groupId string) Config {
	return Config{
		brokers:                brokers,
		name:                   name,
		topic:                  topic,
		groupId:                groupId,
		maxWait:                10 * time.Second,
		startOffset:            kafka.FirstOffset,
		fetchTimeout:           time.Minute,
		maxConsecutiveTimeouts: 3,
	}
}
```

(Keep the `//goland:noinspection GoUnusedExportedFunction` directive above the function.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-kafka && go test -race ./consumer/ -v`
Expected: all PASS (no other unit test asserts the old defaults — every timeout-behavior test sets explicit `SetFetchTimeout`/`SetMaxConsecutiveTimeouts` overrides).

- [ ] **Step 5: Vet and commit**

```bash
cd libs/atlas-kafka && go vet ./...
cd ../..
git add libs/atlas-kafka/consumer/config.go libs/atlas-kafka/consumer/config_test.go
git commit -m "feat(atlas-kafka): default maxWait 50ms->10s, fetchTimeout 5m->1m (task-136 rationale in config.go)"
```

---

### Task 6: Post-fix harness run, IdleTicks assertion, findings completion

Run the full S1–S5 suite green, strengthen S2 with the now-existing `IdleTicks` field (proving ticks actually fired and were classified idle), and complete `findings.md`.

**Files:**
- Modify: `libs/atlas-kafka/consumer/dwell_integration_test.go`
- Modify: `docs/tasks/task-136-consumer-fetch-wedge/findings.md`

**Interfaces:**
- Consumes: Task 4's `Snapshot.IdleTicks`; Task 3's findings draft.
- Produces: completed `findings.md` (acceptance artifact, PRD §10).

- [ ] **Step 1: Strengthen S2 with the idle-tick proof**

In `TestDwellS2_IdleTickChurn` in `libs/atlas-kafka/consumer/dwell_integration_test.go`, after the `require.Zero(t, totalRecreates(cm), ...)` line, add:

```go
	// Prove the deadline ticks actually fired and were classified idle —
	// otherwise a too-long fetchTimeout would make this test vacuous.
	tickedIdle := 0
	for _, c := range cm.Consumers() {
		if c.Snapshot().IdleTicks > 0 {
			tickedIdle++
		}
	}
	require.GreaterOrEqual(t, tickedIdle, 10,
		"S2: expected most idle consumers to have recorded idle ticks (fetchTimeout=2s over a 30s window)")
```

- [ ] **Step 2: Run the full integration suite**

Run: `cd libs/atlas-kafka && go test -tags integration -race -run TestDwell -v -timeout 60m ./consumer/ 2>&1 | tee /tmp/task-136-postfix.log`
Expected: **all five scenarios PASS**, including S2 (zero recreates, p99 < 1s, ≥10 idle-ticking consumers) and S3 (recreate dwell ≤ 10s). If S3's measured max dwell is nowhere near 10s (e.g. consistently < 3s), keep the 10s bound — it is the design's join+backoff budget, not a tuned number; record the measured value in findings. If any scenario FAILS, this is a genuine defect — use superpowers:systematic-debugging on the failing scenario before touching thresholds; do not loosen an assertion to get green.

- [ ] **Step 3: Also re-run the unit suite**

Run: `cd libs/atlas-kafka && go test -race ./... && go vet ./... && go vet -tags integration ./consumer/`
Expected: all PASS, vet clean.

- [ ] **Step 4: Complete findings.md**

Fill the three deferred sections of `docs/tasks/task-136-consumer-fetch-wedge/findings.md` from `/tmp/task-136-postfix.log` (quote actual numbers):

- **Post-fix results:** the S1–S5 table mirroring the pre-fix one (p99/max/recreates per scenario, S2's idle-tick counts, S3's measured recreate dwell vs the 10s bound).
- **Config default changes & rationale:** the §4.4 table (maxWait 50ms→10s, fetchTimeout 5m→1m, maxConsecutiveTimeouts 3 with refined meaning) plus S5's measured fetch-rate reduction as the maxWait justification.
- **Follow-up decision (design §7 gate):** the harness broker is unloaded, so the dwell reproduced and eliminated here is client-side (H1). State the decision rule verbatim: if post-deploy live observation still shows multi-second dwells with wedge logs gone, file the cluster-infra follow-up (multi-broker / per-env topic reduction / design Approach C) citing S5's extrapolated numbers; otherwise the library fix suffices. Record S5's extrapolation: `<fast>` vs `<slow>` fetch attempts/reader/30s × ~481 live partitions.

Also update the `## Hypothesis verdicts` section with final verdicts now that both baseline and post-fix numbers exist (H1 confirmed/refuted by S2 before/after; H2 quantified by S5; H3 closed by S4; H4 closed by MaxHandlerDuration attribution).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-kafka/consumer/dwell_integration_test.go docs/tasks/task-136-consumer-fetch-wedge/findings.md
git commit -m "test(atlas-kafka): post-fix dwell bounds green; findings with root-cause attribution (task-136)"
```

---

### Task 7: Full verification sweep (shared-lib bump discipline)

`libs/atlas-kafka` is vendored by every Go service; CLAUDE.md's build discipline applies even though no service source changed.

**Files:** none (verification only; commit only if fixes are needed).

**Interfaces:**
- Consumes: everything above.
- Produces: a verified branch ready for the code-review phase (review runs via `superpowers:requesting-code-review` before any PR — do not skip).

- [ ] **Step 1: Module-level checks**

Run, from the worktree root:
```bash
cd libs/atlas-kafka && go test -race ./... && go vet ./... && go build ./... && cd ../..
```
Expected: all PASS/clean.

- [ ] **Step 2: Redis key guard**

Run, from the worktree root (no global `GOWORK=off` prefix):
```bash
tools/redis-key-guard.sh
```
Expected: clean (this task adds no redis usage; the guard is a repo invariant).

- [ ] **Step 3: Bake every Go service**

Run, from the worktree root:
```bash
docker buildx bake all-go-services
```
Expected: every image builds. A failure here means a Dockerfile `COPY libs/...` or module wiring issue that `go build` cannot catch — fix and re-bake; do not skip.

- [ ] **Step 4: Confirm branch state**

Run:
```bash
git status --short
git log --oneline main..HEAD
```
Expected: clean tree; the six commits from Tasks 1–6. Report the verification outputs verbatim in the completion summary. Implementation ends here — code review (`superpowers:requesting-code-review`) is the next phase's entry point, followed by PR creation.
