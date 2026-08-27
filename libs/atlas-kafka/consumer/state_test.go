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

// TestRecordTopicMissingCountsAndStamps pins FR-3.1/FR-3.3: the pair is a
// monotonic counter plus a timestamp, mirroring idleTicks/lastIdleTickAt.
func TestRecordTopicMissingCountsAndStamps(t *testing.T) {
	c := newTestConsumer()

	if s := c.Snapshot(); s.TopicMissingObservations != 0 || !s.LastTopicMissingAt.IsZero() {
		t.Fatalf("fresh consumer: observations = %d, lastAt = %v, want 0 and zero time", s.TopicMissingObservations, s.LastTopicMissingAt)
	}

	c.recordTopicMissing()
	s := c.Snapshot()
	if s.TopicMissingObservations != 1 {
		t.Fatalf("TopicMissingObservations = %d, want 1", s.TopicMissingObservations)
	}
	if s.LastTopicMissingAt.IsZero() {
		t.Fatal("LastTopicMissingAt is zero after an observation")
	}

	c.recordTopicMissing()
	c.recordTopicMissing()
	if got := c.Snapshot().TopicMissingObservations; got != 3 {
		t.Fatalf("TopicMissingObservations = %d after three observations, want 3", got)
	}
}

// TestSnapshotTopicMissingSupersededByAssignment pins the FR-3.3 supersession
// rule: the counter is NOT cleared on recovery — it is the post-mortem signal
// this incident lacked. "Currently healthy" is read as lastAssignmentAt being
// after lastTopicMissingAt.
func TestSnapshotTopicMissingSupersededByAssignment(t *testing.T) {
	c := newTestConsumer()
	c.recordTopicMissing()
	c.recordTopicMissing()

	time.Sleep(2 * time.Millisecond)
	c.onAssignment(7, []int{0})

	s := c.Snapshot()
	if s.TopicMissingObservations != 2 {
		t.Fatalf("TopicMissingObservations = %d after recovery, want the history preserved (2)", s.TopicMissingObservations)
	}
	if !s.LastAssignmentAt.After(s.LastTopicMissingAt) {
		t.Fatalf("LastAssignmentAt (%v) is not after LastTopicMissingAt (%v); the supersession read breaks", s.LastAssignmentAt, s.LastTopicMissingAt)
	}
	if len(s.AssignedPartitions) != 1 || s.AssignedPartitions[0] != 0 {
		t.Fatalf("AssignedPartitions = %v, want [0]", s.AssignedPartitions)
	}
}
