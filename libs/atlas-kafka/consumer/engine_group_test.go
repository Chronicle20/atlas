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
	// The assigned partition's runPartition goroutine is spawned via
	// gen.Start and is not tracked by wg (mirrors production: kafka-go owns
	// generation-scoped goroutines, not the caller). Wait for it explicitly
	// so it cannot outlive the test and race a later test's mutation of the
	// package-level drainTimeout var.
	gen.wait()
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
	// See the equivalent wait in TestAssignedGenerationStartsOnePartitionLoop:
	// the active generation's runPartition goroutine is not tracked by wg.
	active.wait()
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
