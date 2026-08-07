package mist

import (
	mist2 "atlas-channel/kafka/message/mist"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// withRecordingBroadcasters swaps the package-level broadcast seams for
// recording stubs that capture invocations. Returns a restore func and
// pointers to the captured arguments. Tests use this to assert the
// AffectedAreaCreated/AffectedAreaRemoved wire effect of the mist consumer
// without standing up a REST mock for ForSessionsInMap.
func withRecordingBroadcasters(t *testing.T) (restore func(), createdCalls *int, lastCreated *fieldpkt.AffectedAreaCreated, removedCalls *int, lastRemoved *fieldpkt.AffectedAreaRemoved) {
	t.Helper()
	createdN, removedN := 0, 0
	var capturedCreated fieldpkt.AffectedAreaCreated
	var capturedRemoved fieldpkt.AffectedAreaRemoved
	origCreated := affectedAreaCreatedBroadcaster
	origRemoved := affectedAreaRemovedBroadcaster
	affectedAreaCreatedBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, body fieldpkt.AffectedAreaCreated) {
		createdN++
		capturedCreated = body
	}
	affectedAreaRemovedBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, body fieldpkt.AffectedAreaRemoved) {
		removedN++
		capturedRemoved = body
	}
	return func() {
		affectedAreaCreatedBroadcaster = origCreated
		affectedAreaRemovedBroadcaster = origRemoved
	}, &createdN, &capturedCreated, &removedN, &capturedRemoved
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 1)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// TestMistCreated_BroadcastsAffectedAreaCreated synthesises a MIST_CREATED
// event and asserts the channel consumer translates it into an
// AffectedAreaCreated broadcast carrying the same mist identity and bounds.
func TestMistCreated_BroadcastsAffectedAreaCreated(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, createdCalls, lastCreated, _, _ := withRecordingBroadcasters(t)
	defer restore()

	mistId := uuid.New()
	h := handleMistCreated(sc, nil)
	h(logrus.New(), ctx, mist2.Event[mist2.CreatedBody]{
		Tenant:    tm.Id(),
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MapId:     100000000,
		Instance:  uuid.Nil,
		MistId:    mistId,
		Type:      mist2.EventTypeCreated,
		Body: mist2.CreatedBody{
			OwnerType:        "MONSTER",
			OwnerId:          424242,
			SourceSkillId:    2121006,
			SourceSkillLevel: 20,
			Type:             1,
			OriginX:          100,
			OriginY:          200,
			LtX:              -50,
			LtY:              -60,
			RbX:              50,
			RbY:              60,
			Duration:         8000,
		},
	})

	if *createdCalls != 1 {
		t.Fatalf("MIST_CREATED: want 1 AffectedAreaCreated broadcast, got %d", *createdCalls)
	}
	if got := lastCreated.MistId(); got != mistId {
		t.Fatalf("AffectedAreaCreated.MistId: want %s, got %s", mistId, got)
	}
	if got := lastCreated.OwnerId(); got != 424242 {
		t.Fatalf("AffectedAreaCreated.OwnerId: want 424242, got %d", got)
	}
	if lastCreated.OriginX() != 100 || lastCreated.OriginY() != 200 {
		t.Fatalf("AffectedAreaCreated origin: want (100,200), got (%d,%d)", lastCreated.OriginX(), lastCreated.OriginY())
	}
	if lastCreated.LtX() != -50 || lastCreated.LtY() != -60 || lastCreated.RbX() != 50 || lastCreated.RbY() != 60 {
		t.Fatalf("AffectedAreaCreated bounds wrong: lt (%d,%d) rb (%d,%d)", lastCreated.LtX(), lastCreated.LtY(), lastCreated.RbX(), lastCreated.RbY())
	}
	if lastCreated.TEnd() != 8000 {
		t.Fatalf("AffectedAreaCreated.TEnd: want 8000 (duration ms), got %d", lastCreated.TEnd())
	}
	if lastCreated.Phase() != 80 {
		t.Fatalf("AffectedAreaCreated.Phase: want 80 (8000ms / 100), got %d", lastCreated.Phase())
	}
	if lastCreated.SkillId() != 2121006 {
		t.Fatalf("AffectedAreaCreated.SkillId: want 2121006, got %d", lastCreated.SkillId())
	}
	if lastCreated.SkillLevel() != 20 {
		t.Fatalf("AffectedAreaCreated.SkillLevel: want 20, got %d", lastCreated.SkillLevel())
	}
	if lastCreated.NType() != 1 {
		t.Fatalf("AffectedAreaCreated.NType: want 1, got %d", lastCreated.NType())
	}
}

// TestMistCreated_WrongType_DoesNotBroadcast guards against the handler
// firing for unrelated event types delivered on the same topic.
func TestMistCreated_WrongType_DoesNotBroadcast(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, createdCalls, _, _, _ := withRecordingBroadcasters(t)
	defer restore()

	h := handleMistCreated(sc, nil)
	h(logrus.New(), ctx, mist2.Event[mist2.CreatedBody]{
		Tenant:    tm.Id(),
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MistId:    uuid.New(),
		Type:      mist2.EventTypeDestroyed, // wrong type for created handler
	})

	if *createdCalls != 0 {
		t.Fatalf("wrong-type event: want 0 broadcasts, got %d", *createdCalls)
	}
}

// TestMistPhase covers the duration(ms) -> phase(100ms units) conversion
// performed for the AffectedAreaCreated `phase` field, including the
// int16-overflow clamp, the sub-100ms truncation floor, and the degenerate
// zero/negative cases (task-12: phase must carry the real mist lifetime so
// the client computes the correct expiry instead of `0*100 + now`).
func TestMistPhase(t *testing.T) {
	tests := []struct {
		name       string
		durationMs int64
		want       int16
	}{
		{name: "normal duration", durationMs: 8000, want: 80},
		{name: "exact 100ms boundary", durationMs: 100, want: 1},
		{name: "sub-100ms truncation floors to 1", durationMs: 50, want: 1},
		{name: "zero duration floors to 1", durationMs: 0, want: 1},
		{name: "negative duration floors to 1", durationMs: -5000, want: 1},
		{name: "int16-max duration is exact", durationMs: int64(math.MaxInt16) * 100, want: math.MaxInt16},
		{name: "overflow clamps to int16 max", durationMs: int64(math.MaxInt16)*100 + 100, want: math.MaxInt16},
		{name: "large overflow clamps to int16 max", durationMs: 1_000_000_000, want: math.MaxInt16},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mistPhase(tt.durationMs); got != tt.want {
				t.Fatalf("mistPhase(%d): want %d, got %d", tt.durationMs, tt.want, got)
			}
		})
	}
}

// TestMistCreated_PhaseOverflow_ClampsToInt16Max asserts the consumer-level
// wiring (not just the mistPhase helper in isolation) clamps a duration that
// would overflow the wire int16 phase field, so a very long mist does not
// wrap into a negative phase (which would compute a client-side expiry in
// the past).
func TestMistCreated_PhaseOverflow_ClampsToInt16Max(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, lastCreated, _, _ := withRecordingBroadcasters(t)
	defer restore()

	h := handleMistCreated(sc, nil)
	h(logrus.New(), ctx, mist2.Event[mist2.CreatedBody]{
		Tenant:    tm.Id(),
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MapId:     100000000,
		Instance:  uuid.Nil,
		MistId:    uuid.New(),
		Type:      mist2.EventTypeCreated,
		Body: mist2.CreatedBody{
			Duration: 1_000_000_000, // far beyond int16*100
		},
	})

	if lastCreated.Phase() != math.MaxInt16 {
		t.Fatalf("AffectedAreaCreated.Phase: want %d (int16 max, clamped), got %d", int16(math.MaxInt16), lastCreated.Phase())
	}
	if lastCreated.Phase() < 0 {
		t.Fatalf("AffectedAreaCreated.Phase must never be negative (would compute a past expiry), got %d", lastCreated.Phase())
	}
}

// TestMistDestroyed_BroadcastsAffectedAreaRemoved synthesises a
// MIST_DESTROYED event and asserts the channel consumer broadcasts
// AffectedAreaRemoved carrying the same mist identity.
func TestMistDestroyed_BroadcastsAffectedAreaRemoved(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, _, removedCalls, lastRemoved := withRecordingBroadcasters(t)
	defer restore()

	mistId := uuid.New()
	h := handleMistDestroyed(sc, nil)
	h(logrus.New(), ctx, mist2.Event[mist2.DestroyedBody]{
		Tenant:    tm.Id(),
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MapId:     100000000,
		Instance:  uuid.Nil,
		MistId:    mistId,
		Type:      mist2.EventTypeDestroyed,
		Body: mist2.DestroyedBody{
			Reason: "EXPIRED",
		},
	})

	if *removedCalls != 1 {
		t.Fatalf("MIST_DESTROYED: want 1 AffectedAreaRemoved broadcast, got %d", *removedCalls)
	}
	if got := lastRemoved.MistId(); got != mistId {
		t.Fatalf("AffectedAreaRemoved.MistId: want %s, got %s", mistId, got)
	}
}

// TestMistDestroyed_WrongType_DoesNotBroadcast guards against the
// destroy handler firing for unrelated event types on the same topic.
func TestMistDestroyed_WrongType_DoesNotBroadcast(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, _, removedCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	h := handleMistDestroyed(sc, nil)
	h(logrus.New(), ctx, mist2.Event[mist2.DestroyedBody]{
		Tenant:    tm.Id(),
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MistId:    uuid.New(),
		Type:      mist2.EventTypeCreated, // wrong type for destroyed handler
	})

	if *removedCalls != 0 {
		t.Fatalf("wrong-type event: want 0 broadcasts, got %d", *removedCalls)
	}
}
