package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

func testLogger(t *testing.T) *logrus.Logger {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// mustEnvelope builds the wire bytes atlas-configurations' outbox.NewEnvironmentEnvelope
// produces (Task 19): {schema_version, id, config, emitted_at} where config
// is the env.Record itself.
func mustEnvelope(t *testing.T, rec env.Record) []byte {
	t.Helper()
	bts, err := json.Marshal(struct {
		SchemaVersion int        `json:"schema_version"`
		Id            string     `json:"id"`
		Config        env.Record `json:"config"`
		EmittedAt     string     `json:"emitted_at"`
	}{
		SchemaVersion: 1,
		Id:            string(rec.Name),
		Config:        rec,
		EmittedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("mustEnvelope: %v", err)
	}
	return bts
}

// mustTenantEnvelope builds the wire bytes atlas-configurations' outbox.NewTenantEnvelope
// produces for the tenant-status topic (task-232 R21-1): the same envelope
// shape as mustEnvelope, but Config is the tenant's RestModel — here reduced
// to just the "environment" attribute this projection reads.
func mustTenantEnvelope(t *testing.T, tenantId string, environment string) []byte {
	t.Helper()
	bts, err := json.Marshal(struct {
		SchemaVersion int    `json:"schema_version"`
		Id            string `json:"id"`
		Config        struct {
			Environment string `json:"environment"`
		} `json:"config"`
		EmittedAt string `json:"emitted_at"`
	}{
		SchemaVersion: 1,
		Id:            tenantId,
		Config: struct {
			Environment string `json:"environment"`
		}{Environment: environment},
		EmittedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("mustTenantEnvelope: %v", err)
	}
	return bts
}

func TestEnvironmentProjectionAppliesRecords(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp()}

	payload := mustEnvelope(t, env.Record{
		Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-character": "atlas-pr-123"},
		Phase:     env.PhaseActive,
	})
	if _, err := s.handle(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{
			Topic: "t", Partition: 0, Offset: 0,
			Key: []byte("environment:pr-123"), Value: payload,
		}); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if !reg.IsActive(env.Id("pr-123")) {
		t.Fatal("record not applied")
	}
}

func TestEnvironmentProjectionAppliesTombstones(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseActive,
	})
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp()}

	if _, err := s.handle(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{Topic: "t", Key: []byte("environment:pr-123"), Value: nil}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if reg.IsActive(env.Id("pr-123")) {
		t.Fatal("tombstone not applied")
	}
}

func TestEnvironmentProjectionIgnoresAnUnreadableSchema(t *testing.T) {
	// Forward-compatible, matching the configuration projection: a schema we
	// cannot read is acknowledged, not retried.
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp()}

	cont, err := s.handle(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{
			Topic: "t", Key: []byte("environment:x"),
			Value: []byte(`{"schema_version":999}`),
		})
	if err != nil || !cont {
		t.Fatalf("got (cont=%v, err=%v), want (true, nil)", cont, err)
	}
}

// TestTenantProjectionAppliesEnvironment pins task-232 R21-1/FR-7.3: a
// tenant-status message projects its "environment" attribute into the
// registry's tenant map.
func TestTenantProjectionAppliesEnvironment(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp(), tenantTopic: "tenant-topic"}

	payload := mustTenantEnvelope(t, "t-1", "pr-123")
	if _, err := s.handleTenant(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{
			Topic: "tenant-topic", Partition: 0, Offset: 0,
			Key: []byte("tenant:t-1"), Value: payload,
		}); err != nil {
		t.Fatalf("handleTenant: %v", err)
	}

	got, ok := reg.EnvironmentOfTenant("t-1")
	if !ok || got != env.Id("pr-123") {
		t.Fatalf("got (%q, %v), want (\"pr-123\", true)", got, ok)
	}
}

// TestTenantProjectionAppliesTombstones pins the tombstone half: a
// "tenant:<uuid>" tombstone removes the tenant's projected environment.
func TestTenantProjectionAppliesTombstones(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.ApplyTenant("t-1", env.Id("pr-123"))
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp(), tenantTopic: "tenant-topic"}

	if _, err := s.handleTenant(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{Topic: "tenant-topic", Key: []byte("tenant:t-1"), Value: nil}); err != nil {
		t.Fatalf("handleTenant: %v", err)
	}

	if _, ok := reg.EnvironmentOfTenant("t-1"); ok {
		t.Fatal("tenant tombstone not applied")
	}
}

// TestTenantProjectionIgnoresAnUnmatchedTombstoneKey mirrors the
// environment-topic equivalent: a tombstone whose key doesn't carry the
// "tenant:" prefix must not remove anything.
func TestTenantProjectionIgnoresAnUnmatchedTombstoneKey(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.ApplyTenant("t-1", env.Id("pr-123"))
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp(), tenantTopic: "tenant-topic"}

	if _, err := s.handleTenant(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{Topic: "tenant-topic", Key: []byte("t-1"), Value: nil}); err != nil {
		t.Fatalf("handleTenant: %v", err)
	}

	got, ok := reg.EnvironmentOfTenant("t-1")
	if !ok || got != env.Id("pr-123") {
		t.Fatal("tenant wrongly removed by a malformed tombstone key")
	}
}

// TestEnvSubscriberStartSkipsTenantConsumerWhenTopicUnset pins task-232
// R21-3: an unset tenant topic must not be fatal, and must not contribute
// offsets to the catch-up gate. With only the environment topic's (empty)
// snapshot registered, the gate must still be able to flip ready.
func TestEnvSubscriberStartSkipsTenantConsumerWhenTopicUnset(t *testing.T) {
	c := newEnvCaughtUp()
	s := &envSubscriber{
		registry:    env.NewMapRegistry(env.Id("main"), time.Now),
		caughtUp:    c,
		topic:       "env-topic",
		tenantTopic: "", // unset
		readEndOffsets: func(context.Context, []string, string) (map[int]int64, error) {
			return map[int]int64{}, nil
		},
	}

	if err := s.Start(context.Background(), testLogger(t), &sync.WaitGroup{}, "group"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !c.CaughtUpNow() {
		t.Fatal("gate must flip ready from the environment topic's empty snapshot alone when the tenant topic is unset")
	}
}

func TestBootstrapWithoutTheOptionLeavesTheLegacyRegistry(t *testing.T) {
	// FR-1.8: a service that has not yet been migrated keeps today's
	// behaviour exactly.
	if !env.CurrentRegistry().IsOwner(env.Id(""), "atlas-anything") {
		t.Fatal("default registry is not the legacy no-op")
	}
}

// --- Tests proving the fixtures above actually discriminate ---

// TestEnvironmentProjectionDoesNotApplyAnUnmatchedTombstoneKey guards
// against silently dropping tombstones by decoding an id from a key that
// doesn't carry the "environment:" prefix atlas-configurations writes.
func TestEnvironmentProjectionDoesNotApplyAnUnmatchedTombstoneKey(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseActive,
	})
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp()}

	if _, err := s.handle(testLogger(t))(testLogger(t), context.Background(),
		kafka.Message{Topic: "t", Key: []byte("pr-123"), Value: nil}); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if !reg.IsActive(env.Id("pr-123")) {
		t.Fatal("record wrongly removed by a malformed tombstone key")
	}
}

func TestEnvCaughtUpNotReadyUntilEndOffsetsCleared(t *testing.T) {
	c := newEnvCaughtUp()
	if c.CaughtUpNow() {
		t.Fatal("caught up before any end offsets were even set")
	}

	c.SetEndOffsets("env-topic", map[int]int64{0: 3})
	if c.CaughtUpNow() {
		t.Fatal("caught up immediately after SetEndOffsets, before any messages consumed")
	}

	c.Observe("env-topic", 0, 1)
	if c.CaughtUpNow() {
		t.Fatal("caught up before reaching the snapshotted end offset")
	}

	c.Observe("env-topic", 0, 2) // end=3 means offsets 0,1,2 exist; caught up at consumed==end-1
	if !c.CaughtUpNow() {
		t.Fatal("not caught up after consuming through the snapshotted end offset")
	}
}

func TestEnvCaughtUpWaitCaughtUpUnblocksOnFlip(t *testing.T) {
	c := newEnvCaughtUp()
	c.SetEndOffsets("env-topic", map[int]int64{0: 1})

	done := make(chan error, 1)
	go func() { done <- c.WaitCaughtUp(context.Background()) }()

	select {
	case <-done:
		t.Fatal("WaitCaughtUp returned before the gate flipped")
	case <-time.After(20 * time.Millisecond):
	}

	c.Observe("env-topic", 0, 0)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WaitCaughtUp: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitCaughtUp did not unblock after the gate flipped")
	}
}

// TestEnvCaughtUpKeyedByTopicNotPartitionAlone pins task-232 R21-2: the
// gate is keyed by (topic, partition), not partition alone. Two topics each
// register a non-trivial end offset on partition 0; consuming only topic
// A's partition 0 must leave the gate NOT caught up, even though topic B's
// partition 0 end offset happens to be satisfiable by the same numbers. A
// partition-only key would wrongly flip the gate here.
func TestEnvCaughtUpKeyedByTopicNotPartitionAlone(t *testing.T) {
	c := newEnvCaughtUp()
	c.SetEndOffsets("topic-a", map[int]int64{0: 3})
	c.SetEndOffsets("topic-b", map[int]int64{0: 3})

	// Fully consume topic-a's partition 0 only.
	c.Observe("topic-a", 0, 2)
	if c.CaughtUpNow() {
		t.Fatal("caught up after consuming only one of two registered topics")
	}

	// Now also consume topic-b's partition 0; only now should the gate flip.
	c.Observe("topic-b", 0, 2)
	if !c.CaughtUpNow() {
		t.Fatal("not caught up after consuming both registered topics through their end offsets")
	}
}

// --- Fix round 1: an end-offset resolution failure that is NOT "topic
// missing" must fail closed, never flip the gate ready with an empty
// registry (task-20-report.md, review finding "readiness gate has a path
// that reports Ready with an empty registry"). ---

// TestResolveStartOffsetsTreatsMissingTopicAsAnEmptySnapshot is the
// legitimate case: the topic has not been created yet (no environment has
// ever been published). kafka.UnknownTopicOrPartition must resolve to an
// empty, non-error snapshot so the gate is free to flip immediately.
func TestResolveStartOffsetsTreatsMissingTopicAsAnEmptySnapshot(t *testing.T) {
	offsets, err := resolveStartOffsets(context.Background(), []string{"broker:9092"}, "t",
		func(context.Context, []string, string) (map[int]int64, error) {
			return nil, kafka.UnknownTopicOrPartition
		},
		testLogger(t))
	if err != nil {
		t.Fatalf("missing topic must not error: %v", err)
	}
	if offsets == nil || len(offsets) != 0 {
		t.Fatalf("missing topic must resolve to an empty (non-nil) snapshot, got %v", offsets)
	}
}

// TestResolveStartOffsetsFailsClosedOnAnUnrelatedError is the outage case:
// any error OTHER than kafka.UnknownTopicOrPartition (e.g. a dial failure)
// must propagate rather than collapse to an empty snapshot.
func TestResolveStartOffsetsFailsClosedOnAnUnrelatedError(t *testing.T) {
	outage := errors.New("dial tcp 10.0.0.1:9092: connect: connection refused")
	_, err := resolveStartOffsets(context.Background(), []string{"broker:9092"}, "t",
		func(context.Context, []string, string) (map[int]int64, error) { return nil, outage },
		testLogger(t))
	if err == nil {
		t.Fatal("a broker-outage-shaped error must propagate, not collapse to an empty snapshot")
	}
	if !errors.Is(err, outage) {
		t.Fatalf("propagated error must wrap the original: %v", err)
	}
}

// TestEnvSubscriberStartFailsClosedWhenEndOffsetsUnresolvable is the
// assertion that would have caught the original bug: when end-offset
// resolution fails for a reason other than a missing topic, Start must
// return an error and the caught-up gate must NOT report ready. Start
// returns before touching consumer.GetManager(), so this runs with no live
// Kafka broker.
func TestEnvSubscriberStartFailsClosedWhenEndOffsetsUnresolvable(t *testing.T) {
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	c := newEnvCaughtUp()
	s := &envSubscriber{
		registry: reg,
		caughtUp: c,
		topic:    "t",
		readEndOffsets: func(context.Context, []string, string) (map[int]int64, error) {
			return nil, errors.New("dial tcp: connection refused")
		},
	}

	err := s.Start(context.Background(), testLogger(t), &sync.WaitGroup{}, "group")
	if err == nil {
		t.Fatal("Start must fail when end offsets cannot be resolved for a non-missing-topic reason")
	}
	if c.CaughtUpNow() {
		t.Fatal("gate must not report ready when end offsets could not be resolved")
	}
}
