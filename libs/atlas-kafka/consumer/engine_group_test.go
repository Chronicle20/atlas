package consumer

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
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

	// grp.Close() (fakeGroup, mirroring kafka-go's Generation.close()) blocks
	// until gen's Start'd runPartition goroutine has actually exited, so
	// wg.Wait() below is already sufficient synchronization — no separate
	// gen.wait() is needed (see TestGroupCloseWaitsForPartitionGoroutine,
	// which pins that contract directly).
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

	// Same reasoning as TestAssignedGenerationStartsOnePartitionLoop: Close()
	// already blocks on active's Start'd goroutine, so wg.Wait() alone is a
	// sufficient barrier.
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

// TestGroupCloseWaitsForPartitionGoroutine pins the fake's join-on-Close
// contract (task-209 review finding: fakeGroup.Close must mirror kafka-go's
// Generation.close(), which blocks until every Start'd goroutine has
// returned — consumergroup.go:344-360, 685-690). startGroupEngine's shutdown
// safety depends on that block: its unconditional grp.Close() is what
// guarantees a generation's runPartition goroutine cannot still be running
// (and mutating shared state) when startGroupEngine's wg.Done() fires. This
// test fails if either half of that chain regresses: a fakeGroup.Close()
// that returns early, or a startGroupEngine that stops calling/awaiting it.
//
// The handler is held open on a channel the test controls; the pass/fail
// gate is the ordering this produces (the engine cannot finish until the
// handler is released, and the handler must have actually returned by the
// time it does), not elapsed time. The 150ms window only proves the block is
// real rather than a lucky scheduling race — a regressed fakeGroup.Close()
// returns essentially instantly, so it would trip that window every run.
func TestGroupCloseWaitsForPartitionGoroutine(t *testing.T) {
	l, _ := silentLogger()

	release := make(chan struct{})
	handlerStarted := make(chan struct{})
	var startedOnce sync.Once
	var mu sync.Mutex
	var handlerReturned bool

	gen := newFakeGeneration(1, map[string][]kafka.PartitionAssignment{
		"t": {{ID: 0, Offset: kafka.FirstOffset}},
	})
	grp := newFakeGroup(gen)

	c := newGroupConsumer("t", grp)
	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0, Value: []byte("m")}}}
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }
	c.handlers = map[string]handler.Handler{
		"stuck": func(_ logrus.FieldLogger, _ context.Context, _ kafka.Message) (bool, error) {
			startedOnce.Do(func() { close(handlerStarted) })
			<-release
			mu.Lock()
			handlerReturned = true
			mu.Unlock()
			return true, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	// Wait until the handler has actually been entered (not merely until the
	// message was fetched) — this is what guarantees runPartitionFetchLoop's
	// wg.Add(1) has already happened, so the upcoming cancel() is guaranteed
	// to leave a genuinely outstanding handler behind for quiesce to wait on.
	select {
	case <-handlerStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("handler was never dispatched")
	}

	cancel()

	engineDone := make(chan struct{})
	routine.Go(l, context.Background(), func(_ context.Context) {
		wg.Wait()
		close(engineDone)
	})

	select {
	case <-engineDone:
		t.Fatal("startGroupEngine finished (wg.Done fired) before the stuck handler was released; grp.Close() did not block on the generation's partition goroutine")
	case <-time.After(150 * time.Millisecond):
	}

	close(release)

	select {
	case <-engineDone:
	case <-time.After(3 * time.Second):
		t.Fatal("startGroupEngine never finished after the handler was released")
	}

	mu.Lock()
	defer mu.Unlock()
	if !handlerReturned {
		t.Fatal("engine finished without the handler having actually returned")
	}
}

// TestGateDoesNotJoinUntilTopicExists is the core of the fix: a consumer whose
// topic does not exist must stay a NON-MEMBER. A non-member cannot destabilise
// the group, which is what keeps the two healthy members of the group working
// (design §2.3).
func TestGateDoesNotJoinUntilTopicExists(t *testing.T) {
	l, hook := silentLogger()

	var mu sync.Mutex
	var gpCalls int
	gen := newFakeGeneration(4, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)

	counter := newFakePartitionCounter(counterResult{0, ErrTopicNotFound})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce
	c.gp = func(kafka.ConsumerGroupConfig) (Group, error) {
		mu.Lock()
		gpCalls++
		mu.Unlock()
		return grp, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return counter.callCount() >= 1 }, "the gate never consulted the partition-count seam")
	waitFor(t, func() bool { return hasLogContaining(hook, "does not exist or has no partitions") }, "the gate never warned about the missing topic")

	mu.Lock()
	joined := gpCalls
	mu.Unlock()
	if joined != 0 {
		t.Fatalf("GroupProducer called %d times while the topic was missing, want 0", joined)
	}
	if got := c.Snapshot().TopicMissingObservations; got < 1 {
		t.Fatalf("TopicMissingObservations = %d while waiting, want >= 1", got)
	}

	counter.set(1, nil)
	waitFor(t, func() bool { return c.Snapshot().GenerationID == 4 }, "the consumer never joined after the topic appeared")

	if e := logEntryContaining(hook, "does not exist or has no partitions"); e == nil || e.Level != logrus.WarnLevel {
		t.Fatalf("missing-topic line logged at %v, want WarnLevel", e)
	}

	cancel()
	wg.Wait()
}

// TestGroupEngineRecoversWhenTopicAppears is THE incident test: no pod
// restart, no operator action — the consumer waits the topic out and then
// consumes. Written against startGroupEngine (the whole engine), because
// recovery now lives in the gate rather than in runGenerations.
func TestGroupEngineRecoversWhenTopicAppears(t *testing.T) {
	l, _ := silentLogger()

	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0, Value: []byte("m")}}}
	gen := newFakeGeneration(11, map[string][]kafka.PartitionAssignment{
		"EVENT_TOPIC_TEST": {{ID: 0, Offset: kafka.FirstOffset}},
	})
	grp := newFakeGroup(gen)

	counter := newFakePartitionCounter(counterResult{0, ErrTopicNotFound}, counterResult{1, nil})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return len(gen.commits()) > 0 }, "the consumer never consumed after the topic appeared")

	s := c.Snapshot()
	if s.GenerationID != 11 {
		t.Fatalf("GenerationID = %d, want 11", s.GenerationID)
	}
	if len(s.AssignedPartitions) != 1 || s.AssignedPartitions[0] != 0 {
		t.Fatalf("AssignedPartitions = %v, want [0]", s.AssignedPartitions)
	}
	if s.TopicMissingObservations != 1 {
		t.Fatalf("TopicMissingObservations = %d, want exactly 1 (one distinct observation before recovery)", s.TopicMissingObservations)
	}
	if !s.LastAssignmentAt.After(s.LastTopicMissingAt) {
		t.Fatal("LastAssignmentAt is not after LastTopicMissingAt; the supersession read would still say 'missing' after recovery")
	}

	cancel()
	wg.Wait()
}

// TestGateExitsOnContextCancel: the gate runs after wg.Add(1), so a gate that
// does not select on ctx.Done() hangs shutdown forever.
func TestGateExitsOnContextCancel(t *testing.T) {
	l, _ := silentLogger()

	var mu sync.Mutex
	var gpCalls int
	grp := newFakeGroup()

	counter := newFakePartitionCounter(counterResult{0, ErrTopicNotFound})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce
	c.gp = func(kafka.ConsumerGroupConfig) (Group, error) {
		mu.Lock()
		gpCalls++
		mu.Unlock()
		return grp, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return counter.callCount() >= 1 }, "the gate never consulted the partition-count seam")
	cancel()

	done := make(chan struct{})
	routine.Go(l, context.Background(), func(_ context.Context) {
		wg.Wait()
		close(done)
	})
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("startGroupEngine never returned after cancel; the gate does not select on ctx.Done()")
	}

	mu.Lock()
	defer mu.Unlock()
	if gpCalls != 0 {
		t.Fatalf("GroupProducer called %d times, want 0 — the gate must never join while the topic is missing", gpCalls)
	}
}

// TestNilPartitionCountProducerSkipsGate pins the "nil seam is inert" default
// that keeps every struct-literal Consumer in this suite unchanged.
func TestNilPartitionCountProducerSkipsGate(t *testing.T) {
	l, _ := silentLogger()

	gen := newFakeGeneration(4, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	if c.pcp != nil {
		t.Fatal("newGroupConsumer must leave pcp nil; the inert default is what this test pins")
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return c.Snapshot().GenerationID == 4 }, "a nil pcp blocked the join; the gate must be inert without a seam")

	cancel()
	wg.Wait()
}

// TestGateJoinsImmediatelyOnIndeterminateLookup is FR-2.5 at the gate: a
// broker outage must not hold a consumer out of its group.
func TestGateJoinsImmediatelyOnIndeterminateLookup(t *testing.T) {
	l, hook := silentLogger()

	gen := newFakeGeneration(4, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)
	counter := newFakePartitionCounter(counterResult{0, context.DeadlineExceeded})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return c.Snapshot().GenerationID == 4 }, "an indeterminate lookup held the consumer out of its group; FR-2.5 forbids it")

	if hasLogContaining(hook, "does not exist or has no partitions") {
		t.Fatal("an indeterminate lookup was reported as a missing topic")
	}
	if got := c.Snapshot().TopicMissingObservations; got != 0 {
		t.Fatalf("TopicMissingObservations = %d after an indeterminate lookup, want 0", got)
	}

	cancel()
	wg.Wait()
}

// TestEmptyAssignmentWarnsWhenTopicMissing is FR-2.3: an empty assignment
// caused by a MISSING topic is not healthy-idle and must say so. It also pins
// design §3.3's "return, don't continue": errTopicMissing is a deliberate
// withdrawal, so it must not reach recordError or the Error log.
func TestEmptyAssignmentWarnsWhenTopicMissing(t *testing.T) {
	l, hook := silentLogger()

	gen := newFakeGeneration(4, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)
	counter := newFakePartitionCounter(counterResult{1, nil}, counterResult{0, ErrTopicNotFound})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool {
		return hasLogContaining(hook, "because the topic does not exist or has no partitions")
	}, "the empty assignment was never classified as a missing topic")

	e := logEntryContaining(hook, "because the topic does not exist or has no partitions")
	if e == nil || e.Level != logrus.WarnLevel {
		t.Fatalf("missing-topic classification logged at %v, want WarnLevel", e)
	}
	if !strings.Contains(e.Message, "leaving the group until it appears") {
		t.Fatalf("warn does not say the consumer is leaving the group: %q", e.Message)
	}
	if !strings.Contains(e.Message, "EVENT_TOPIC_TEST") || !strings.Contains(e.Message, "Test Service") {
		t.Fatalf("warn must name the topic and the group: %q", e.Message)
	}
	if hasLogContaining(hook, "exited; rejoining after backoff") {
		t.Fatal("errTopicMissing was treated as an error; it is a deliberate withdrawal")
	}
	if got := c.Snapshot().LastError; got != "" {
		t.Fatalf("LastError = %q after a deliberate withdrawal, want empty", got)
	}
	if got := c.Snapshot().TopicMissingObservations; got < 1 {
		t.Fatalf("TopicMissingObservations = %d, want >= 1", got)
	}

	cancel()
	wg.Wait()
}

// TestEmptyAssignmentStaysHealthyIdleWhenPartitionsExist is FR-2.4 preserved
// exactly: with 237 single-partition topics and replicas: 2, one member of
// every group holds nothing permanently. It must stay debug, must not tick,
// must not recreate, and must PARK in Next rather than rejoin.
func TestEmptyAssignmentStaysHealthyIdleWhenPartitionsExist(t *testing.T) {
	l, hook := silentLogger()
	l.SetLevel(logrus.DebugLevel)

	gen := newFakeGeneration(4, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)
	counter := newFakePartitionCounter(counterResult{1, nil})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return c.Snapshot().GenerationID == 4 }, "generation was never observed")
	waitFor(t, func() bool {
		return hasLogContaining(hook, "holds no partition assignment in generation 4; healthy-idle.")
	}, "the held-elsewhere path lost its healthy-idle debug line")

	time.Sleep(300 * time.Millisecond)

	s := c.Snapshot()
	if s.RecreateCount != 0 {
		t.Fatalf("RecreateCount = %d while healthy-idle, want 0", s.RecreateCount)
	}
	if s.IdleTicks != 0 {
		t.Fatalf("IdleTicks = %d while healthy-idle, want 0", s.IdleTicks)
	}
	if s.TopicMissingObservations != 0 {
		t.Fatalf("TopicMissingObservations = %d while the topic exists, want 0", s.TopicMissingObservations)
	}
	if hasLogContaining(hook, "because the topic does not exist or has no partitions") {
		t.Fatal("a held-elsewhere assignment was misreported as a missing topic")
	}
	if n := grp.nextCalls(); n > 2 {
		t.Fatalf("Next called %d times while parked; the healthy-idle branch rejoined instead of parking", n)
	}

	cancel()
	wg.Wait()
}

// TestEmptyAssignmentIndeterminateLookupIsHealthyIdle is FR-2.5 at the
// classification: a failed lookup falls through to today's behaviour, and in
// particular does NOT leave the group.
func TestEmptyAssignmentIndeterminateLookupIsHealthyIdle(t *testing.T) {
	l, hook := silentLogger()
	l.SetLevel(logrus.DebugLevel)

	gen := newFakeGeneration(4, map[string][]kafka.PartitionAssignment{})
	grp := newFakeGroup(gen)
	counter := newFakePartitionCounter(counterResult{0, context.DeadlineExceeded})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool {
		return hasLogContaining(hook, "holds no partition assignment in generation 4; healthy-idle.")
	}, "an indeterminate lookup did not fall through to healthy-idle")

	time.Sleep(300 * time.Millisecond)

	if hasLogContaining(hook, "because the topic does not exist or has no partitions") {
		t.Fatal("an indeterminate lookup was reported as a missing topic")
	}
	if got := c.Snapshot().TopicMissingObservations; got != 0 {
		t.Fatalf("TopicMissingObservations = %d after an indeterminate lookup, want 0", got)
	}
	if n := grp.nextCalls(); n > 2 {
		t.Fatalf("Next called %d times; an indeterminate lookup must park, not leave the group", n)
	}

	cancel()
	wg.Wait()
}

// TestSteadyPartitionCountCausesNoRejoin: a steady partition count adds no
// rejoin of OUR making. kafka-go's own partition watcher is upstream-owned and
// broker-bound, so it is not under test here; what this pins is that the
// classification's per-generation metadata read never turns into churn.
func TestSteadyPartitionCountCausesNoRejoin(t *testing.T) {
	l, _ := silentLogger()

	rd := &scriptedPartitionReader{msgs: []kafka.Message{{Offset: 0, Value: []byte("m")}}}
	gen := newFakeGeneration(6, map[string][]kafka.PartitionAssignment{
		"EVENT_TOPIC_TEST": {{ID: 0, Offset: kafka.FirstOffset}},
	})
	grp := newFakeGroup(gen)
	counter := newFakePartitionCounter(counterResult{1, nil})
	c := newGroupConsumer("EVENT_TOPIC_TEST", grp)
	c.pcp = counter.produce
	c.prp = func(kafka.ReaderConfig, int64) KafkaReader { return rd }

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	routine.Go(l, ctx, func(_ context.Context) { c.startGroupEngine(l, ctx, wg) })

	waitFor(t, func() bool { return len(gen.commits()) > 0 }, "assigned partition never delivered")
	time.Sleep(300 * time.Millisecond)

	if got := c.Snapshot().RecreateCount; got != 0 {
		t.Fatalf("RecreateCount = %d on a steady partition count, want 0", got)
	}
	if n := grp.nextCalls(); n > 2 {
		t.Fatalf("Next called %d times on a steady partition count; the engine is rejoining", n)
	}

	cancel()
	wg.Wait()
}
