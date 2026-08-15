package service

import (
	"context"
	"encoding/json"
	"io"
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

	c.SetEndOffsets(map[int]int64{0: 3})
	if c.CaughtUpNow() {
		t.Fatal("caught up immediately after SetEndOffsets, before any messages consumed")
	}

	c.Observe(0, 1)
	if c.CaughtUpNow() {
		t.Fatal("caught up before reaching the snapshotted end offset")
	}

	c.Observe(0, 2) // end=3 means offsets 0,1,2 exist; caught up at consumed==end-1
	if !c.CaughtUpNow() {
		t.Fatal("not caught up after consuming through the snapshotted end offset")
	}
}

func TestEnvCaughtUpWaitCaughtUpUnblocksOnFlip(t *testing.T) {
	c := newEnvCaughtUp()
	c.SetEndOffsets(map[int]int64{0: 1})

	done := make(chan error, 1)
	go func() { done <- c.WaitCaughtUp(context.Background()) }()

	select {
	case <-done:
		t.Fatal("WaitCaughtUp returned before the gate flipped")
	case <-time.After(20 * time.Millisecond):
	}

	c.Observe(0, 0)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WaitCaughtUp: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitCaughtUp did not unblock after the gate flipped")
	}
}
