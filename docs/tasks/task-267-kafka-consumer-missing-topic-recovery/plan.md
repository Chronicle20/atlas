# Kafka Consumer Recovery From a Late-Appearing Topic — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A consumer whose topic does not yet exist must never join a consumer group deaf; it waits out the topic's creation as a non-member and starts consuming when the topic appears, with no pod restart and no rebalance storm.

**Architecture:** Three cooperating parts in `libs/atlas-kafka/consumer`, plus a one-line logging fix in `atlas-character-factory`. (1) A `PartitionCountProducer` seam answers "does this topic have partitions right now?" without a broker in tests. (2) A pre-join readiness gate in `startGroupEngine` polls that seam and refuses to join while the topic is definitively absent. (3) A three-way classification of the empty-assignment branch in `runGenerations` distinguishes healthy-idle from topic-missing and, on topic-missing, leaves the group back to the gate. `WatchPartitionChanges: true` is enabled on top of all three and is load-bearing only for the partition-count-change case on an existing topic.

**Tech Stack:** Go 1.27, `github.com/segmentio/kafka-go v0.4.51`, `github.com/sirupsen/logrus` (+ `hooks/test` for log assertions), standard `testing`.

**Spec:** `docs/tasks/task-267-kafka-consumer-missing-topic-recovery/design.md` (PRD: `prd.md`, incident: `incident.md`)

## Global Constraints

- **Two module roots.** Tasks 1–5 build and test from `libs/atlas-kafka`. Task 6 builds and tests from `services/atlas-character-factory/atlas.com/character-factory`. Every `go build ./...` / `go test ./...` in this plan runs from the module root named in the task's `### Files` block.
- **Nil seam means inert.** `Consumer.pcp == nil` means *gate disabled, classification indeterminate*. Every existing test in `engine_group_test.go`, `idle_stuck_test.go`, `dwell_integration_test.go`, `state_test.go` and `group_test.go` builds a `Consumer` as a struct literal, so `pcp` is nil there and their behaviour must not change. Do not edit those existing tests except where a task explicitly says so.
- **Only a definite negative triggers a leave.** `ErrTopicNotFound` — an explicit `UnknownTopicOrPartition`, or a *successful* metadata exchange with zero partitions / no such topic entry. Every other error (transport, timeout, any other topic error code) is **indeterminate** and must fall through to today's behaviour (FR-2.5). Never treat an error as zero.
- **No readiness-probe or health-endpoint change** (FR-3.4). The gate runs inside the per-consumer goroutine launched by `routine.Go` in `AddConsumer` (`libs/atlas-kafka/consumer/manager.go:199`) and must never block HTTP server start.
- **`wg.Add(1)` stays at the top of `startGroupEngine`, before the gate.** The gate therefore MUST `select` on `ctx.Done()` on every wait, or shutdown hangs.
- **The `reader` (legacy) engine is frozen.** It never builds a `ConsumerGroupConfig`; nothing in this plan touches `engine_reader.go`.
- **Existing wording is contract.** The healthy-idle debug line's text — `"Consumer for topic [%s] (group [%s]) holds no partition assignment in generation %d; healthy-idle."` — must survive byte-for-byte on the held-elsewhere path (FR-2.4).
- **Snapshot fields are superseded, not cleared** (FR-3.3). `topicMissingObservations` is monotonic. A consumer is currently healthy iff `assignedPartitions` is non-empty, or `lastAssignmentAt` is after `lastTopicMissingAt`.
- **Task 5 lands last of the library tasks.** `WatchPartitionChanges: true` is only safe once Tasks 3 and 4 keep the member out of the group whenever the topic is absent (design §2.3, §2.4).

---

### Task 1: The partition-count seam

**Files:**
- `libs/atlas-kafka/consumer/group.go` — add `ErrTopicNotFound`, `topicMetadataTimeout`, `PartitionCountProducer`, `ConfigPartitionCountProducer`, `partitionCountFromMetadata`, `defaultPartitionCountProducer`
- `libs/atlas-kafka/consumer/manager.go` — `Manager.pcp` field, default wiring in `GetManager`, `Consumer.pcp` field, `pcp: m.pcp` in `AddConsumer`
- `libs/atlas-kafka/consumer/group_test.go` — new mapping table test; extend two existing tests
- `libs/atlas-kafka/go.mod` — read-only; confirms `kafka-go v0.4.51`

Module root: `libs/atlas-kafka`.

Patterns to copy: `libs/atlas-kafka/consumer/group.go:33-54` (producer type + `Config*` + default, exactly the shape `GroupProducer`/`PartitionReaderProducer` already use) and `libs/atlas-kafka/consumer/manager.go:190-191` (how `gp`/`prp` flow `Manager` → `Consumer`).

**Interfaces:**
- Produces:
  - `var ErrTopicNotFound error`
  - `const topicMetadataTimeout = 5 * time.Second`
  - `type PartitionCountProducer func(ctx context.Context, brokers []string, topic string) (int, error)`
  - `func ConfigPartitionCountProducer(pcp PartitionCountProducer) ManagerConfig`
  - `func partitionCountFromMetadata(topic string, res *kafka.MetadataResponse) (int, error)`
  - `func defaultPartitionCountProducer(ctx context.Context, brokers []string, topic string) (int, error)`
  - `Manager.pcp`, `Consumer.pcp` (both `PartitionCountProducer`)

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-kafka/consumer/group_test.go`. The mapping test is a pure
function test over a hand-built `*kafka.MetadataResponse` — no broker, no
network. `kafka.Topic` has fields `Name string`, `Partitions []Partition`,
`Error error` (kafka-go `kafka.go:14-30`); `kafka.UnknownTopicOrPartition` is
`kafka.Error(3)` (kafka-go `error.go:18`).

| case | metadata response | want count | want err |
|---|---|---|---|
| topic present with two partitions | `Topics: [{Name:"t", Partitions:[{ID:0},{ID:1}]}]` | `2` | `nil` |
| topic present with one partition | `Topics: [{Name:"t", Partitions:[{ID:0}]}]` | `1` | `nil` |
| topic entry carries UnknownTopicOrPartition | `Topics: [{Name:"t", Error: kafka.UnknownTopicOrPartition}]` | `0` | `ErrTopicNotFound` |
| topic present, zero partitions, no error | `Topics: [{Name:"t", Partitions: nil}]` | `0` | `ErrTopicNotFound` |
| topic absent from the response entirely | `Topics: [{Name:"other", Partitions:[{ID:0}]}]` | `0` | `ErrTopicNotFound` |
| empty response | `Topics: nil` | `0` | `ErrTopicNotFound` |
| some other topic error | `Topics: [{Name:"t", Error: kafka.LeaderNotAvailable}]` | `0` | `kafka.LeaderNotAvailable` (NOT `ErrTopicNotFound`) |

```go
// TestPartitionCountFromMetadata pins the FR-2.5 boundary: only a DEFINITE
// negative maps to ErrTopicNotFound. Every other topic error stays itself, so
// the callers treat it as indeterminate and fall through to today's behaviour.
func TestPartitionCountFromMetadata(t *testing.T) {
	tests := []struct {
		name      string
		res       *kafka.MetadataResponse
		wantCount int
		wantErr   error
	}{
		{"two partitions", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Partitions: []kafka.Partition{{ID: 0}, {ID: 1}}}}}, 2, nil},
		{"one partition", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Partitions: []kafka.Partition{{ID: 0}}}}}, 1, nil},
		{"unknown topic", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Error: kafka.UnknownTopicOrPartition}}}, 0, ErrTopicNotFound},
		{"zero partitions", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t"}}}, 0, ErrTopicNotFound},
		{"topic absent", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "other", Partitions: []kafka.Partition{{ID: 0}}}}}, 0, ErrTopicNotFound},
		{"empty response", &kafka.MetadataResponse{}, 0, ErrTopicNotFound},
		{"other topic error is indeterminate", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Error: kafka.LeaderNotAvailable}}}, 0, kafka.LeaderNotAvailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := partitionCountFromMetadata("t", tt.res)
			if got != tt.wantCount {
				t.Fatalf("count = %d, want %d", got, tt.wantCount)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && tt.wantErr != ErrTopicNotFound && errors.Is(err, ErrTopicNotFound) {
				t.Fatalf("indeterminate error %v was collapsed into ErrTopicNotFound", err)
			}
		})
	}
}
```

Also extend the two existing tests in the same file:

- `TestManagerDefaultProducersArePresent` (`libs/atlas-kafka/consumer/group_test.go:98-108`) — add, after the `m.prp` check:

```go
	if m.pcp == nil {
		t.Fatal("Manager.pcp is nil; GetManager must install a default PartitionCountProducer")
	}
```

- `TestConfigProducersOverrideDefaults` (`libs/atlas-kafka/consumer/group_test.go:111-124`) — add a third seam: declare `pcpCalled bool`, add
  `ConfigPartitionCountProducer(func(context.Context, []string, string) (int, error) { pcpCalled = true; return 1, nil })`
  to the `GetManager` call, call `_, _ = m.pcp(context.Background(), nil, "t")`, and include `pcpCalled` in the final assertion. This requires adding `"context"` and `"errors"` to the file's import block.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestPartitionCountFromMetadata|TestManagerDefaultProducersArePresent|TestConfigProducersOverrideDefaults' -v`
Expected: FAIL to compile — `undefined: partitionCountFromMetadata`, `undefined: ErrTopicNotFound`, `m.pcp undefined`, `undefined: ConfigPartitionCountProducer`.

- [ ] **Step 3: Add the seam to `group.go`**

Add `"errors"` and `"time"` to `group.go`'s import block (it currently imports
only `"context"` and `kafka`). Insert after `ConfigPartitionReaderProducer`
(`libs/atlas-kafka/consumer/group.go:54`):

```go
// ErrTopicNotFound reports that the broker knows no partitions for the topic:
// either UNKNOWN_TOPIC_OR_PARTITION, or a successful metadata response whose
// topic entry carries zero partitions. It is a DEFINITE negative — every other
// error is indeterminate (FR-2.5), because acting on a broker blip by leaving
// a group is worse than the deafness this task fixes.
var ErrTopicNotFound = errors.New("topic not found or has no partitions")

// topicMetadataTimeout bounds a single partition-count lookup, and doubles as
// the metadata client's own Timeout. 5s is the same order as kafka-go's
// default PartitionWatchInterval; the lookup is off the message path, so the
// worst case is a delayed join, never a stalled fetch.
const topicMetadataTimeout = 5 * time.Second

// PartitionCountProducer reports the current partition count for a topic.
// Shaped as a producer function, like GroupProducer and
// PartitionReaderProducer, so both the pre-join gate and the empty-assignment
// classification can be scripted without a broker (FR-2.2). Group stays a pure
// subset of *kafka.ConsumerGroup: the gate needs this lookup BEFORE a Group
// exists at all.
type PartitionCountProducer func(ctx context.Context, brokers []string, topic string) (int, error)

//goland:noinspection GoUnusedExportedFunction
func ConfigPartitionCountProducer(pcp PartitionCountProducer) ManagerConfig {
	return func(m *Manager) {
		m.pcp = pcp
	}
}

// defaultPartitionCountProducer asks the cluster for one topic's metadata.
//
// kafka.Client.Metadata, NOT kafka.Conn.ReadPartitions: ReadPartitions sends
// topicMetadataRequestV6{... AllowAutoTopicCreation: true} (kafka-go
// conn.go:984-986), and a consumer must never create a topic as a side effect
// of asking whether it exists. Client.Metadata builds metadataAPI.Request with
// TopicNames only (kafka-go metadata.go:40-44) and returns the per-topic error
// as a structured field rather than collapsing the response into one error —
// which is what lets partitionCountFromMetadata separate "no such topic" from
// "broker unreachable".
func defaultPartitionCountProducer(ctx context.Context, brokers []string, topic string) (int, error) {
	client := &kafka.Client{Addr: kafka.TCP(brokers...), Timeout: topicMetadataTimeout}
	res, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{topic}})
	if err != nil {
		return 0, err
	}
	return partitionCountFromMetadata(topic, res)
}

// partitionCountFromMetadata is the pure mapping half of
// defaultPartitionCountProducer, split out so the FR-2.5 boundary is testable
// without a broker.
func partitionCountFromMetadata(topic string, res *kafka.MetadataResponse) (int, error) {
	if res == nil {
		return 0, ErrTopicNotFound
	}
	for _, t := range res.Topics {
		if t.Name != topic {
			continue
		}
		if t.Error != nil {
			if errors.Is(t.Error, kafka.UnknownTopicOrPartition) {
				return 0, ErrTopicNotFound
			}
			return 0, t.Error
		}
		if len(t.Partitions) == 0 {
			return 0, ErrTopicNotFound
		}
		return len(t.Partitions), nil
	}
	return 0, ErrTopicNotFound
}
```

- [ ] **Step 4: Wire the seam through `manager.go`**

Three edits, all mirroring how `gp`/`prp` already flow:

1. `Manager` struct (`libs/atlas-kafka/consumer/manager.go:82-89`) — add `pcp PartitionCountProducer` after `prp`.
2. `GetManager` (`libs/atlas-kafka/consumer/manager.go:110-112`) — add `pcp: defaultPartitionCountProducer,` after the `prp:` line.
3. `Consumer` struct (`libs/atlas-kafka/consumer/manager.go:252-254`) — add `pcp PartitionCountProducer` after `prp`, with the comment:

```go
	// pcp answers "does this topic have partitions right now?". A nil pcp
	// means gate disabled, classification indeterminate — which is what keeps
	// every struct-literal Consumer in the test suite behaviourally unchanged.
	pcp PartitionCountProducer
```

4. `AddConsumer`'s `&Consumer{...}` literal (`libs/atlas-kafka/consumer/manager.go:190-191`) — add `pcp: m.pcp,` after `prp: m.prp,`.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestPartitionCountFromMetadata|TestManagerDefaultProducersArePresent|TestConfigProducersOverrideDefaults' -v`
Expected: PASS

- [ ] **Step 6: Run the whole package**

Run: `cd libs/atlas-kafka && go build ./... && go test ./consumer/`
Expected: PASS, no behaviour change to any existing test.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-kafka/consumer/group.go libs/atlas-kafka/consumer/manager.go libs/atlas-kafka/consumer/group_test.go
git commit -m "feat(atlas-kafka): add PartitionCountProducer seam for topic-readiness lookups"
```

---

### Task 2: Topic-missing observability state

**Files:**
- `libs/atlas-kafka/consumer/manager.go` — `topicMissingWarnInterval`, `Consumer.topicMissingObservations` / `lastTopicMissingAt`, `recordTopicMissing`, two `Snapshot` fields, their population in `Snapshot()`
- `libs/atlas-kafka/consumer/debug.go` — two `debugAttributes` fields and their mapping in `snapshotToAttributes`
- `libs/atlas-kafka/consumer/state_test.go` — new state test
- `libs/atlas-kafka/consumer/debug_test.go` — extend the external-package attribute struct and its assertions

Module root: `libs/atlas-kafka`.

Patterns to copy: `libs/atlas-kafka/consumer/manager.go:506-513` (`recordIdleTick` — the counter+timestamp recorder shape) and `libs/atlas-kafka/consumer/debug.go:69-71,100-103` (the `idleTicks`/`lastIdleTickAt` attribute pair, lowerCamelCase JSON tags).

**Interfaces:**
- Consumes: nothing from Task 1.
- Produces:
  - `const topicMissingWarnInterval = 1 * time.Minute`
  - `func (c *Consumer) recordTopicMissing()`
  - `Snapshot.TopicMissingObservations int`, `Snapshot.LastTopicMissingAt time.Time`
  - JSON attribute names `topicMissingObservations`, `lastTopicMissingAt`

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-kafka/consumer/state_test.go` (package `consumer`, so it
can reach the unexported recorder). `newTestConsumer` is at
`libs/atlas-kafka/consumer/state_test.go:8-15`.

| case | action | expect |
|---|---|---|
| fresh consumer | none | `TopicMissingObservations == 0`, `LastTopicMissingAt.IsZero()` |
| one observation | `recordTopicMissing()` × 1 | `TopicMissingObservations == 1`, `LastTopicMissingAt` non-zero |
| three observations | `recordTopicMissing()` × 3 | `TopicMissingObservations == 3` (monotonic) |
| superseded by assignment | 2 × `recordTopicMissing()`, then `onAssignment(7, []int{0})` | count still `2`, `LastAssignmentAt.After(LastTopicMissingAt)`, `AssignedPartitions == [0]` |

```go
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
```

In `libs/atlas-kafka/consumer/debug_test.go` (package `consumer_test`), add the
two fields to its local `debugAttributes` struct (`debug_test.go:29-42`) so the
JSON names are pinned by decoding:

```go
	TopicMissingObservations int       `json:"topicMissingObservations"`
	LastTopicMissingAt       time.Time `json:"lastTopicMissingAt"`
```

and add to `TestDebugHandler_PopulatedConsumer`, after the existing
`a.LastTimeoutAt` assertion (`debug_test.go:177-179`):

```go
	if a.TopicMissingObservations != 0 {
		t.Fatalf("expected topicMissingObservations=0 on a consumer whose topic exists, got %d", a.TopicMissingObservations)
	}
	if !a.LastTopicMissingAt.IsZero() {
		t.Fatalf("expected lastTopicMissingAt zero on a consumer that never saw a missing topic, got %v", a.LastTopicMissingAt)
	}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestRecordTopicMissing|TestSnapshotTopicMissingSuperseded|TestDebugHandler_PopulatedConsumer' -v`
Expected: FAIL to compile — `c.recordTopicMissing undefined`, `s.TopicMissingObservations undefined`.

- [ ] **Step 3: Add the state, the recorder and the snapshot fields**

In `libs/atlas-kafka/consumer/manager.go`:

1. Next to `legacyPartition` (`manager.go:301-304`), add:

```go
// topicMissingWarnInterval throttles the pre-join gate's repeated warn. A
// ten-minute topic outage produces ten warns per consumer instead of six
// hundred; the per-poll line stays at Debug.
const topicMissingWarnInterval = 1 * time.Minute
```

2. In the `Consumer` struct's "Observable state — protected by mu" block
   (`manager.go:270-274`), after `recreateCount int`:

```go
	// topicMissingObservations counts DISTINCT observations that this
	// consumer's topic does not exist or has no partitions — the pre-join
	// gate's warn and the empty-assignment classification, not every poll.
	// Monotonic: it is SUPERSEDED by a later assignment, never cleared
	// (FR-3.3), because after the pod recovers this counter plus
	// lastTopicMissingAt is the only surviving evidence of the outage.
	topicMissingObservations int
	lastTopicMissingAt       time.Time
```

3. Next to `recordIdleTick` (`manager.go:506-513`), add:

```go
// recordTopicMissing marks one observation that the topic does not exist or
// has no partitions. Mirrors recordIdleTick: increment, stamp, touch nothing
// else — this is not an error state, it is a wait state.
func (c *Consumer) recordTopicMissing() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topicMissingObservations++
	c.lastTopicMissingAt = time.Now()
}
```

4. In `Snapshot` (`manager.go:394-399`), after `LastAssignmentAt time.Time`:

```go
	// TopicMissingObservations counts distinct observations that this
	// consumer's topic does not exist or has no partitions. It is monotonic:
	// a consumer is CURRENTLY healthy iff AssignedPartitions is non-empty, or
	// LastAssignmentAt is after LastTopicMissingAt.
	TopicMissingObservations int
	LastTopicMissingAt       time.Time
```

5. In the returned `Snapshot{...}` literal (`manager.go:460-462`), after
   `LastAssignmentAt: c.lastAssignmentAt,`:

```go
		TopicMissingObservations: c.topicMissingObservations,
		LastTopicMissingAt:       c.lastTopicMissingAt,
```

In `libs/atlas-kafka/consumer/debug.go`:

6. In `debugAttributes` (`debug.go:73-76`), after `Engine string \`json:"engine"\``:

```go
	TopicMissingObservations int       `json:"topicMissingObservations"`
	LastTopicMissingAt       time.Time `json:"lastTopicMissingAt"`
```

7. In `snapshotToAttributes` (`debug.go:105-108`), after `Engine: s.Engine,`:

```go
		TopicMissingObservations: s.TopicMissingObservations,
		LastTopicMissingAt:       s.LastTopicMissingAt,
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestRecordTopicMissing|TestSnapshotTopicMissingSuperseded|TestDebugHandler' -v`
Expected: PASS

- [ ] **Step 5: Run the whole package**

Run: `cd libs/atlas-kafka && go build ./... && go test ./consumer/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/consumer/manager.go libs/atlas-kafka/consumer/debug.go libs/atlas-kafka/consumer/state_test.go libs/atlas-kafka/consumer/debug_test.go
git commit -m "feat(atlas-kafka): expose topicMissingObservations/lastTopicMissingAt on the consumer snapshot"
```

---

### Task 3: The pre-join topic-readiness gate

**Files:**
- `libs/atlas-kafka/consumer/engine_group.go` — new `awaitTopic`; call it in `startGroupEngine` immediately before `c.gp(...)`
- `libs/atlas-kafka/consumer/fakegroup_test.go` — new `fakePartitionCounter` helper and a `logEntryContaining` helper
- `libs/atlas-kafka/consumer/engine_group_test.go` — four new tests

Module root: `libs/atlas-kafka`.

Patterns to copy: `libs/atlas-kafka/consumer/engine_group.go:45-52` (the existing `backoff.next()` + `select { <-ctx.Done() / <-time.After }` + `recordBackoff` block — the gate reuses it verbatim in shape) and `libs/atlas-kafka/consumer/engine_group_test.go:35-80` (`TestZeroAssignmentIsHealthyIdle` — the `routine.Go`/`waitFor`/`cancel`/`wg.Wait()` engine-test scaffold).

**Interfaces:**
- Consumes: `PartitionCountProducer`, `ErrTopicNotFound`, `topicMetadataTimeout`, `Consumer.pcp` (Task 1); `recordTopicMissing`, `topicMissingWarnInterval` (Task 2).
- Produces:
  - `func (c *Consumer) awaitTopic(l logrus.FieldLogger, ctx context.Context) bool` — `false` means the parent context was cancelled, i.e. shut down.
  - test helpers `type counterResult struct { count int; err error }`, `func newFakePartitionCounter(results ...counterResult) *fakePartitionCounter`, `(*fakePartitionCounter).produce`, `(*fakePartitionCounter).set`, `(*fakePartitionCounter).callCount`, `func logEntryContaining(hook *test.Hook, sub string) *logrus.Entry`

- [ ] **Step 1: Write the test helpers**

Append to `libs/atlas-kafka/consumer/fakegroup_test.go` (it already imports
`context`, `sync`, `strings`, `logrus`, `hooks/test`):

```go
// counterResult is one scripted answer from a fakePartitionCounter.
type counterResult struct {
	count int
	err   error
}

// fakePartitionCounter is a scriptable PartitionCountProducer. It pops queued
// results in order and then repeats the last one forever, so a test can drive
// "missing, then present" without listing every poll the backoff makes.
type fakePartitionCounter struct {
	mu    sync.Mutex
	queue []counterResult
	last  counterResult
	calls int
}

func newFakePartitionCounter(results ...counterResult) *fakePartitionCounter {
	f := &fakePartitionCounter{queue: append([]counterResult(nil), results...), last: counterResult{0, ErrTopicNotFound}}
	if len(results) > 0 {
		f.last = results[len(results)-1]
	}
	return f
}

// produce satisfies PartitionCountProducer.
func (f *fakePartitionCounter) produce(_ context.Context, _ []string, _ string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if len(f.queue) == 0 {
		return f.last.count, f.last.err
	}
	r := f.queue[0]
	f.queue = f.queue[1:]
	return r.count, r.err
}

// set discards the remaining script and answers with (count, err) from now on.
func (f *fakePartitionCounter) set(count int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queue = nil
	f.last = counterResult{count, err}
}

func (f *fakePartitionCounter) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// logEntryContaining returns the first captured entry whose message contains
// sub, or nil. Callers assert on entry.Level, which hasLogContaining cannot.
func logEntryContaining(hook *test.Hook, sub string) *logrus.Entry {
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, sub) {
			return e
		}
	}
	return nil
}
```

- [ ] **Step 2: Write the failing gate tests**

Append to `libs/atlas-kafka/consumer/engine_group_test.go`.

| test | counter script | expect |
|---|---|---|
| `TestGateDoesNotJoinUntilTopicExists` | always `{0, ErrTopicNotFound}`, then `set(1, nil)` after the warn is observed | `GroupProducer` not called while missing; called and generation observed after `set` |
| `TestGroupEngineRecoversWhenTopicAppears` | `{0, ErrTopicNotFound}` then `{1, nil}` | a fetch reaches the handler (commit recorded) with **no** consumer restart; `TopicMissingObservations >= 1` |
| `TestGateExitsOnContextCancel` | always `{0, ErrTopicNotFound}` | `wg.Wait()` returns after `cancel()`; `GroupProducer` never called |
| `TestNilPartitionCountProducerSkipsGate` | `c.pcp = nil` | generation 4 observed; counter never consulted |
| `TestGateJoinsImmediatelyOnIndeterminateLookup` | always `{0, context.DeadlineExceeded}` | generation 4 observed; `TopicMissingObservations == 0` |

```go
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
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestGate|TestGroupEngineRecovers|TestNilPartitionCountProducer' -v`
Expected: FAIL — the gate does not exist, so `TestGateDoesNotJoinUntilTopicExists` fails on `GroupProducer called 1 times while the topic was missing`, and `TestGroupEngineRecoversWhenTopicAppears`/`TestGateExitsOnContextCancel` fail their `gpCalls != 0` / observation assertions.

- [ ] **Step 4: Implement `awaitTopic` and call it**

In `libs/atlas-kafka/consumer/engine_group.go`, add after `startGroupEngine`:

```go
// awaitTopic blocks until this consumer's topic definitely has at least one
// partition, or the lookup is indeterminate, or ctx is cancelled. It returns
// false ONLY for cancellation.
//
// This is the cold-start half of task-267. kafka-go's leader forgives a
// missing topic when computing assignments (consumergroup.go:1010-1049) and
// hands the member an EMPTY assignment; its partition watcher is supposed to
// rebalance when the topic appears, but the watcher's startup read returns on
// any error including UnknownTopicOrPartition (consumergroup.go:512-518) — the
// very error its own ticker branch tolerates. So a member that joins for a
// missing topic is permanently deaf. The remedy is to not be a member: a
// non-member cannot go deaf, and cannot destabilise the group for the members
// that ARE working.
//
// FR-2.5: only ErrTopicNotFound — a DEFINITE negative — parks here. Any other
// error is indeterminate and joins immediately, exactly as today, because a
// broker outage must never hold a consumer out of its group.
func (c *Consumer) awaitTopic(l logrus.FieldLogger, ctx context.Context) bool {
	if c.pcp == nil {
		return true
	}

	backoff := newFetchBackoff()
	waiting := false
	var lastWarn time.Time

	for {
		if ctx.Err() != nil {
			return false
		}

		lctx, cancel := context.WithTimeout(ctx, topicMetadataTimeout)
		count, err := c.pcp(lctx, c.brokers, c.topic)
		cancel()

		switch {
		case err != nil && !errors.Is(err, ErrTopicNotFound):
			l.WithError(err).Debugf("Partition-count lookup for topic [%s] was indeterminate; joining group [%s] anyway.", c.topic, c.groupId)
			return true
		case err == nil && count >= 1:
			if waiting {
				l.Infof("Topic [%s] now has %d partition(s); joining group [%s].", c.topic, count, c.groupId)
			}
			return true
		}

		now := time.Now()
		if !waiting || now.Sub(lastWarn) >= topicMissingWarnInterval {
			l.Warnf("Topic [%s] does not exist or has no partitions; consumer will not join group [%s] until it appears.", c.topic, c.groupId)
			lastWarn = now
			c.recordTopicMissing()
		} else {
			l.Debugf("Topic [%s] is still absent; re-polling before joining group [%s].", c.topic, c.groupId)
		}
		waiting = true

		wait := backoff.next()
		select {
		case <-ctx.Done():
			l.Infof("Topic consumer stopped while waiting for topic [%s] to appear.", c.topic)
			return false
		case <-time.After(wait):
			c.recordBackoff(wait)
		}
	}
}
```

Then insert the call in `startGroupEngine`, between the `ctx.Err()` check and
the `c.gp(...)` call (currently `libs/atlas-kafka/consumer/engine_group.go:28-30`):

```go
		// Never join a group for a topic that does not exist (task-267). This
		// runs inside the per-consumer goroutine launched by AddConsumer, and
		// AFTER the wg.Add(1) above — hence awaitTopic's mandatory ctx select.
		if !c.awaitTopic(l, ctx) {
			return
		}

		grp, err := c.gp(c.groupConfig())
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestGate|TestGroupEngineRecovers|TestNilPartitionCountProducer' -v`
Expected: PASS

- [ ] **Step 6: Run the whole package with the race detector**

Run: `cd libs/atlas-kafka && go build ./... && go test -race ./consumer/`
Expected: PASS, no data race. Every existing test still passes (they all leave `pcp` nil).

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-kafka/consumer/engine_group.go libs/atlas-kafka/consumer/fakegroup_test.go libs/atlas-kafka/consumer/engine_group_test.go
git commit -m "feat(atlas-kafka): gate consumer-group join on topic readiness"
```

---

### Task 4: Zero-partition classification in `runGenerations`

**Files:**
- `libs/atlas-kafka/consumer/engine_group.go` — `errTopicMissing`, `emptyAssignmentClass`, `classifyEmptyAssignment`, the three-way empty-assignment branch, the sentinel arm in `startGroupEngine`
- `libs/atlas-kafka/consumer/engine_group_test.go` — four new tests

Module root: `libs/atlas-kafka`.

Patterns to copy: `libs/atlas-kafka/consumer/engine_group.go:73-86` (the branch being split — its debug wording and `continue` must survive verbatim on the held-elsewhere path) and `libs/atlas-kafka/consumer/manager.go:566-587` (`handleFetchDeadline` — the "classify, then branch on a sentinel error" shape).

**Interfaces:**
- Consumes: `PartitionCountProducer`, `ErrTopicNotFound`, `topicMetadataTimeout`, `Consumer.pcp` (Task 1); `recordTopicMissing` (Task 2); `awaitTopic` (Task 3).
- Produces:
  - `var errTopicMissing error` (unexported sentinel)
  - `type emptyAssignmentClass int` with `emptyHealthyIdle`, `emptyBecauseTopicMissing`, `emptyIndeterminate`
  - `func (c *Consumer) classifyEmptyAssignment(ctx context.Context) emptyAssignmentClass`

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-kafka/consumer/engine_group_test.go`.

| test | counter script | generation | expect |
|---|---|---|---|
| `TestEmptyAssignmentWarnsWhenTopicMissing` | `{1,nil}` (gate passes), then `{0,ErrTopicNotFound}` forever | one empty-assignment generation | Warn containing `"because the topic does not exist or has no partitions"` and `"leaving the group until it appears"`; `LastError == ""`; **no** `"exited; rejoining after backoff"` log; `TopicMissingObservations >= 1` |
| `TestEmptyAssignmentStaysHealthyIdleWhenPartitionsExist` | always `{1, nil}` | one empty-assignment generation | debug line `"holds no partition assignment in generation 4; healthy-idle."`; `RecreateCount == 0`; `IdleTicks == 0`; `grp.nextCalls() <= 2` (parked in `Next`, not rejoined); no missing-topic warn |
| `TestEmptyAssignmentIndeterminateLookupIsHealthyIdle` | always `{0, context.DeadlineExceeded}` | one empty-assignment generation | same debug line; `grp.nextCalls() <= 2`; no missing-topic warn; `TopicMissingObservations == 0` |
| `TestSteadyPartitionCountCausesNoRejoin` | always `{1, nil}` | one generation holding partition 0 | after 300 ms: `RecreateCount == 0`, `grp.nextCalls() <= 2` — our engine adds no rejoin |

```go
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
```

`strings` must be added to `engine_group_test.go`'s import block.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestEmptyAssignment|TestSteadyPartitionCount' -v`
Expected: FAIL — `TestEmptyAssignmentWarnsWhenTopicMissing` times out in `waitFor` with `the empty assignment was never classified as a missing topic`, because the branch is still unconditionally healthy-idle.

- [ ] **Step 3: Implement the classification**

In `libs/atlas-kafka/consumer/engine_group.go`, add above `runGenerations`:

```go
// errTopicMissing is a deliberate withdrawal from the group, not a failure:
// runGenerations returns it when an empty assignment is caused by the topic
// not existing. startGroupEngine handles it as a sentinel — no recordError, no
// Error log — and falls back into awaitTopic, which parks as a NON-member.
//
// Returning rather than continuing is what BOUNDS the storm. With
// WatchPartitionChanges enabled, a topic that disappears under a joined member
// makes kafka-go end each generation the instant the watcher's startup read
// fails, and ConsumerGroup.run rejoins with NO backoff (consumergroup.go:713-770
// takes the `err == nil: continue` arm, and JoinGroupBackoff applies only on
// the error arm). That loop is paced by our Next calls, so leaving costs one
// extra rejoin cycle instead of an unbounded rebalance storm.
var errTopicMissing = errors.New("topic missing; leaving the consumer group until it appears")

// emptyAssignmentClass explains WHY this member holds no partitions.
type emptyAssignmentClass int

const (
	// emptyHealthyIdle: the topic has partitions, another member holds them.
	// Expected and permanent on every single-partition topic with replicas: 2.
	emptyHealthyIdle emptyAssignmentClass = iota
	// emptyBecauseTopicMissing: the topic definitively has no partitions.
	emptyBecauseTopicMissing
	// emptyIndeterminate: the lookup could not answer. Treated exactly like
	// emptyHealthyIdle (FR-2.5) — never as zero.
	emptyIndeterminate
)

// classifyEmptyAssignment issues at most ONE partition-count lookup, bounded by
// topicMetadataTimeout. FR-2.6's "at most once per generation" is structural:
// the empty-assignment branch is reached once per generation and then either
// parks in Next or returns. The polling lives in awaitTopic, which is a
// non-member state.
func (c *Consumer) classifyEmptyAssignment(ctx context.Context) emptyAssignmentClass {
	if c.pcp == nil {
		return emptyIndeterminate
	}
	lctx, cancel := context.WithTimeout(ctx, topicMetadataTimeout)
	defer cancel()
	count, err := c.pcp(lctx, c.brokers, c.topic)
	switch {
	case errors.Is(err, ErrTopicNotFound):
		return emptyBecauseTopicMissing
	case err != nil:
		return emptyIndeterminate
	case count >= 1:
		return emptyHealthyIdle
	default:
		return emptyBecauseTopicMissing
	}
}
```

Replace the empty-assignment branch (`libs/atlas-kafka/consumer/engine_group.go:73-86`) with:

```go
		if len(parts) == 0 {
			switch c.classifyEmptyAssignment(ctx) {
			case emptyBecauseTopicMissing:
				c.recordTopicMissing()
				l.Warnf("Consumer for topic [%s] (group [%s]) holds no partition assignment in "+
					"generation %d because the topic does not exist or has no partitions; "+
					"leaving the group until it appears.", c.topic, c.groupId, gen.ID())
				return errTopicMissing
			default:
				// HEALTHY-IDLE (FR-2.1/2.2/2.4/2.5), and the indeterminate
				// case too. Every atlas-main topic has a single partition
				// while services run replicas: 2, so exactly one member of
				// every group lands here permanently. It holds nothing, so it
				// must not tick, must not warn, and must not recreate — the
				// recreate is what rejoins the group and rebalances every
				// OTHER topic in it. Heartbeats are generation-scoped and
				// started unconditionally by kafka-go (consumergroup.go:840),
				// so doing nothing here keeps this member alive and eligible
				// for the next assignment. Debug, not Warn: this state is
				// expected (FR-4.3).
				l.Debugf("Consumer for topic [%s] (group [%s]) holds no partition assignment in generation %d; healthy-idle.",
					c.topic, c.groupId, gen.ID())
				continue
			}
		}
```

Add the sentinel arm in `startGroupEngine`, immediately after the existing
`ctx.Err() != nil || errors.Is(err, context.Canceled)` block
(`libs/atlas-kafka/consumer/engine_group.go:38-41`) and BEFORE `c.recordError(err)`:

```go
		if errors.Is(err, errTopicMissing) {
			// Already warned in runGenerations, and not a failure: no
			// recordError, no Error log. Back off anyway — a flapping lookup
			// must not turn the leave/rejoin cycle into a spin — then fall
			// through to the gate, which parks as a non-member.
			wait := backoff.next()
			select {
			case <-ctx.Done():
				l.Infof("Topic consumer stopped during backoff.")
				return
			case <-time.After(wait):
				c.recordBackoff(wait)
			}
			continue
		}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run 'TestEmptyAssignment|TestSteadyPartitionCount' -v`
Expected: PASS

- [ ] **Step 5: Run the whole package with the race detector**

Run: `cd libs/atlas-kafka && go build ./... && go test -race ./consumer/`
Expected: PASS. In particular `TestZeroAssignmentIsHealthyIdle` (`engine_group_test.go:35`) still passes unchanged — its `Consumer` has a nil `pcp`, so it classifies indeterminate and takes the same `continue`.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/consumer/engine_group.go libs/atlas-kafka/consumer/engine_group_test.go
git commit -m "feat(atlas-kafka): classify zero-partition assignments and leave the group when the topic is gone"
```

---

### Task 5: Enable `WatchPartitionChanges` and rewrite the docs

**Files:**
- `libs/atlas-kafka/consumer/group.go` — flip `WatchPartitionChanges` to `true` in `groupConfig()` and rewrite its comment
- `libs/atlas-kafka/consumer/group_test.go` — invert the assertion in `TestGroupConfigMirrorsTodaysTopology` and rewrite its comment
- `libs/atlas-kafka/README.md` — rewrite the "Partition-count changes" section and document the two new snapshot fields

Module root: `libs/atlas-kafka`.

Patterns to copy: `libs/atlas-kafka/README.md:25-28` (the existing `recreateCount` caveat paragraph — the new snapshot-field note reads the same way).

**Interfaces:**
- Consumes: `awaitTopic` (Task 3), `errTopicMissing` / `classifyEmptyAssignment` (Task 4). This task is only safe once both have landed.
- Produces: nothing new.

- [ ] **Step 1: Invert the failing assertion**

In `libs/atlas-kafka/consumer/group_test.go`, replace the comment and check at
lines 37-43 with:

```go
	// Deliberate divergence from the legacy engine (task-267 FR-1.1/FR-1.2).
	// The `reader` engine inherits kafka-go's ReaderConfig default of false and
	// cannot be changed without touching the frozen rollback path; the
	// consumergroup engine enables the watch as one of THREE parts of the
	// task-267 fix. It is only safe doing so because awaitTopic and
	// classifyEmptyAssignment keep this member out of the group whenever the
	// topic is absent — on its own, the watch's startup read fails with
	// UnknownTopicOrPartition, ends the generation, and rejoins with no
	// backoff (design §2.1-2.3).
	if !cfg.WatchPartitionChanges {
		t.Fatal("WatchPartitionChanges = false, want true (task-267 FR-1.1)")
	}
```

The existing `cfg.Validate()` check on the next line (FR-1.3) stays.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run TestGroupConfigMirrorsTodaysTopology -v`
Expected: FAIL with `WatchPartitionChanges = false, want true (task-267 FR-1.1)`.

- [ ] **Step 3: Flip the flag**

In `libs/atlas-kafka/consumer/group.go`, replace `groupConfig()`'s doc comment
(lines 105-108) and the `WatchPartitionChanges` field (line 115):

```go
// groupConfig builds the consumer-group config for this Consumer. ID, Brokers,
// Topics and StartOffset mirror 1:1 what kafka-go derives internally from
// today's ReaderConfig (reader.go:717-733), so broker-visible group topology is
// unchanged and a rollback to the reader engine needs no group-state migration
// (FR-1.4).
//
// WatchPartitionChanges is the one deliberate divergence (task-267 FR-1.1). It
// covers a partition-count change (1 -> N) on a topic that EXISTS.
// PartitionWatchInterval is left at kafka-go's default (5s;
// consumergroup.go:203-205) per FR-1.5.
//
// It is safe here only because awaitTopic and classifyEmptyAssignment keep this
// member out of the group whenever the topic is absent. Enabled alone, against
// a missing topic, kafka-go's watcher startup read returns
// UnknownTopicOrPartition (consumergroup.go:512-518), the generation ends, and
// ConsumerGroup.run rejoins with no backoff — a group-wide rebalance storm that
// also strips the HEALTHY members of their assignments (design §2.3). Do not
// enable this flag anywhere the two guards are not also present.
func (c *Consumer) groupConfig() kafka.ConsumerGroupConfig {
	return kafka.ConsumerGroupConfig{
		ID:                    c.groupId,
		Brokers:               append([]string(nil), c.brokers...),
		Topics:                []string{c.topic},
		StartOffset:           c.startOffset,
		WatchPartitionChanges: true,
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd libs/atlas-kafka && go test ./consumer/ -run TestGroupConfig -v`
Expected: PASS, including `cfg.Validate()` (FR-1.3).

- [ ] **Step 5: Rewrite the README**

In `libs/atlas-kafka/README.md`, replace the whole "Partition-count changes"
section (lines 30-39) with:

```markdown
### Missing topics and partition-count changes

The `consumergroup` engine **will not join a consumer group for a topic that
does not exist.** Before each join it asks the cluster for the topic's
partition count; while the answer is a definite "no such topic, or no
partitions" it waits — logging a warn on entry and then once a minute — and
joins as soon as the topic appears. If the lookup cannot answer (broker
unreachable, timeout, any other error) the consumer joins immediately, exactly
as it always did: a broker blip must never hold a consumer out of its group.

This exists because of the 2026-08-26 `atlas-pr-1449` incident. kafka-go's
group leader forgives a missing topic when computing assignments and hands the
member an empty assignment, expecting its own partition watcher to trigger a
rebalance once the topic appears. The watcher does not deliver: its startup
read returns on *any* error including `UnknownTopicOrPartition`
(`consumergroup.go:512-518`) — the very error its ticker branch tolerates
(`consumergroup.go:527-529`) — so the goroutine gives up and the member stays
permanently deaf. Twenty-two consumers in one pod raced `atlas-kafka-precreate`
and lost; they were silently deaf for the process's lifetime. That upstream
asymmetry is the reason the pre-join wait exists; a future kafka-go bump that
fixes it should revisit the gate.

The same logic covers the runtime case. When a generation gives this member no
partitions, it asks once whether the topic still has any. If it definitively
does not, the consumer logs a warn and **leaves the group** until it comes
back, rather than holding a doomed membership. If it does — the normal case,
since every atlas-main topic has one partition and services run `replicas: 2` —
that is healthy-idle, logged at debug, and the member parks awaiting the next
rebalance.

`WatchPartitionChanges` is `true` on the `consumergroup` engine, so a topic
repartitioned from 1 to N partitions is picked up within
`PartitionWatchInterval` (kafka-go default, 5s) with no restart. The legacy
`reader` engine leaves it at kafka-go's `false`. **Do not enable this flag
without the two guards above**: against a missing topic, the watcher's failing
startup read ends each generation and kafka-go rejoins with no backoff, which
is a group-wide rebalance storm, strictly worse than the silent deafness it was
meant to fix.

`GET /api/debug/consumers` reports `topicMissingObservations` and
`lastTopicMissingAt`. The counter is monotonic and is **superseded, not
cleared**: after recovery it keeps the history, which is exactly the
post-mortem evidence the incident lacked. A consumer is currently healthy iff
`assignedPartitions` is non-empty, or `lastAssignmentAt` is after
`lastTopicMissingAt` — the same read an operator already performs for
`idleTicks`.
```

- [ ] **Step 6: Run the whole package**

Run: `cd libs/atlas-kafka && go build ./... && go test -race ./consumer/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-kafka/consumer/group.go libs/atlas-kafka/consumer/group_test.go libs/atlas-kafka/README.md
git commit -m "feat(atlas-kafka): enable WatchPartitionChanges on the consumergroup engine"
```

---

### Task 6: Log the preset-creation failure in atlas-character-factory

**Files:**
- `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go` — one log line in `handleCreateFromPreset`
- `services/atlas-character-factory/atlas.com/character-factory/factory/resource_test.go` — new handler log test
- `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go` — read-only; defines `ErrInvalidPresetId` and friends at lines 27-31

Module root: `services/atlas-character-factory/atlas.com/character-factory`.

Patterns to copy: `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:110` (the sibling seed handler's identical log call) and `services/atlas-character-factory/atlas.com/character-factory/factory/resource_test.go:33-41` (`postPreset` — the `ParseInput` round-trip scaffold this test reuses with a hooked logger).

Independent of Tasks 1–5; carried on the same branch.

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: the log message string `"Error creating character from preset."`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-character-factory/atlas.com/character-factory/factory/resource_test.go`.
Add `"github.com/sirupsen/logrus/hooks/test"` to its import block (`logrus`,
`strings`, `net/http`, `net/http/httptest`, `jsonapi` and `server` are already
imported).

`presetId: "not-a-valid-uuid"` makes `processor.CreateFromPreset` return
`ErrInvalidPresetId` (`processor.go:274`) without reaching the tenant registry
or any downstream service — the same path
`TestHandleCreateFromPreset_InvalidPresetIdFormat` (`resource_test.go:66-72`)
already uses.

| assertion | expected |
|---|---|
| status code | `400` (`categorizePresetError(ErrInvalidPresetId)`, unchanged — FR-4.3) |
| an entry with message `"Error creating character from preset."` exists | yes |
| that entry's `Level` | `logrus.ErrorLevel` |
| that entry's `Data["error"]` | non-nil (the error is attached via `WithError`) |
| no entry with message `"Error creating character from seed."` | correct — the two handlers must be separable by a log search (FR-4.2) |

```go
// TestHandleCreateFromPreset_LogsErrorWithPresetMessage covers FR-4.1/4.2/4.3:
// the preset handler swallowed its error entirely, so a 500 from
// /api/factory/characters/from-preset produced no server-side log at all. The
// message must differ from the seed handler's so a log search separates them,
// and the mapped status code must not change.
func TestHandleCreateFromPreset_LogsErrorWithPresetMessage(t *testing.T) {
	ctx, _ := createMockContext(t)
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	dep := server.NewHandlerDependency(l, ctx)
	hc := server.NewHandlerContext(jsonapi.ServerInformation(stubServerInfo{}))

	body := `{"data":{"type":"preset-create","attributes":{"presetId":"not-a-valid-uuid","accountId":1,"worldId":0,"name":"TestChar"}}}`
	req := httptest.NewRequest(http.MethodPost, "/characters/from-preset", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/vnd.api+json")
	rr := httptest.NewRecorder()
	server.ParseInput[PresetCreateRestModel](&dep, &hc, handleCreateFromPreset)(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; categorizePresetError mapping must be unchanged", rr.Code)
	}

	var found *logrus.Entry
	for _, e := range hook.AllEntries() {
		if e.Message == "Error creating character from preset." {
			found = e
		}
		if e.Message == "Error creating character from seed." {
			t.Fatalf("preset handler logged the seed handler's message; FR-4.2 requires them to differ")
		}
	}
	if found == nil {
		t.Fatal("handleCreateFromPreset failed without logging anything")
	}
	if found.Level != logrus.ErrorLevel {
		t.Fatalf("logged at %v, want ErrorLevel", found.Level)
	}
	if found.Data[logrus.ErrorKey] == nil {
		t.Fatal("the error itself was not attached; use WithError(err)")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go test ./factory/ -run TestHandleCreateFromPreset_LogsErrorWithPresetMessage -v`
Expected: FAIL with `handleCreateFromPreset failed without logging anything`.

- [ ] **Step 3: Add the log line**

In `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go`,
inside `handleCreateFromPreset`'s error branch (lines 61-65), add the log call
before `w.WriteHeader(statusCode)`:

```go
		if err != nil {
			d.Logger().WithError(err).Error("Error creating character from preset.")
			statusCode := categorizePresetError(err)
			w.WriteHeader(statusCode)
			return
		}
```

`categorizePresetError` and the status mapping are untouched (FR-4.3).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go test ./factory/ -run 'TestHandleCreateFromPreset|TestCategorizePresetError' -v`
Expected: PASS — including the four pre-existing tests, whose status-code assertions must be unchanged.

- [ ] **Step 5: Build and test the service**

Run: `cd services/atlas-character-factory/atlas.com/character-factory && go build ./... && go test ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character-factory/atlas.com/character-factory/factory/resource.go services/atlas-character-factory/atlas.com/character-factory/factory/resource_test.go
git commit -m "fix(atlas-character-factory): log preset-creation failures"
```

---

### Task 7: Repo-wide verification

**Files:**
- `tools/verify.sh` — read-only; the gate

Module root: repository root.

No source changes. Every service in the monorepo consumes `libs/atlas-kafka`, so
the library change must be proven not to break any of them.

- [ ] **Step 1: Run the quick gate**

Run: `tools/verify.sh --quick`
Expected: exit 0. If it fails, fix the failure in the owning task's files and re-run; do not proceed.

- [ ] **Step 2: Run the full gate**

Run: `tools/verify.sh`
Expected: exit 0. This is the flagless run — it includes the bake and `-race`, and per CLAUDE.md only the flagless run counts as "done".

- [ ] **Step 3: Confirm no service outside atlas-character-factory changed**

Run: `git diff --name-only e880af7e4..HEAD -- services/ | sort -u`
Expected: only paths under `services/atlas-character-factory/`.

- [ ] **Step 4: Code review before PR**

Dispatch `backend-guidelines-reviewer` over the changed Go packages
(`libs/atlas-kafka/consumer`, `services/atlas-character-factory/.../factory`)
and `plan-adherence-reviewer` over this plan. Both are gates; a green
`verify.sh` cannot see a cross-service seam defect.
