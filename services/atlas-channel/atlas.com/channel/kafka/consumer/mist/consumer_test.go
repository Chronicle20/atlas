package mist

import (
	mist2 "atlas-channel/kafka/message/mist"
	protectionmist "atlas-channel/mist"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

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
	// The duration is NOT carried on this packet. nElemAttr is not a time
	// field, and skillDelay is a delay-before-drawing — putting the duration in
	// either one hides or mis-renders the mist client-side.
	if lastCreated.ElemAttr() != 0 {
		t.Fatalf("AffectedAreaCreated.ElemAttr: want 0 (not a duration), got %d", lastCreated.ElemAttr())
	}
	if lastCreated.SkillDelay() != 0 {
		t.Fatalf("AffectedAreaCreated.SkillDelay: want 0 (draw immediately), got %d", lastCreated.SkillDelay())
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

// TestMistCreated_SkillDelayIsNeverDerivedFromDuration is the regression guard
// for the defect this branch shipped and then fixed: skillDelay was briefly set
// to Duration/100 on the belief that it was the mist's client-side lifetime.
// It is not — the client computes tStart = get_update_time() + 100*skillDelay
// and refuses to DRAW the mist until then (v83 CAffectedAreaPool::Update
// @0x431214), so a duration-derived skillDelay hides the mist for its entire
// duration and it is removed at almost the same instant it first appears.
// skillDelay must stay 0 no matter how long the mist lives.
func TestMistCreated_SkillDelayIsNeverDerivedFromDuration(t *testing.T) {
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

	if lastCreated.SkillDelay() != 0 {
		t.Fatalf("AffectedAreaCreated.SkillDelay: want 0 for any duration (it is a draw delay, not a lifetime), got %d", lastCreated.SkillDelay())
	}
	if lastCreated.ElemAttr() != 0 {
		t.Fatalf("AffectedAreaCreated.ElemAttr: want 0 for any duration, got %d", lastCreated.ElemAttr())
	}
}

// TestMistCreated_UsesEventRenderValues asserts the broadcast packet takes
// skillDelay and nElemAttr from the event rather than a channel-local constant
// (task-200 FR-2.4) -- and that the resulting values are still 0, so the wire
// bytes of every already-verified SPAWN_MIST fixture are unchanged (FR-5.5).
func TestMistCreated_UsesEventRenderValues(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, createdCalls, lastCreated, _, _ := withRecordingBroadcasters(t)
	defer restore()

	handleMistCreated(sc, nil)(logrus.New(), ctx, mist2.Event[mist2.CreatedBody]{
		Tenant: tm.Id(), WorldId: 0, ChannelId: 1, MapId: 100000000,
		Instance: uuid.Nil, MistId: uuid.New(), Type: mist2.EventTypeCreated,
		Body: mist2.CreatedBody{
			OwnerType: "CHARACTER", OwnerId: 1001,
			SourceSkillId: 2111003, SourceSkillLevel: 1,
			Type:    0,
			OriginX: 500, OriginY: 300,
			LtX: -110, LtY: -82, RbX: 110, RbY: 83,
			Duration:   4000,
			ElemAttr:   0,
			SkillDelay: 0,
		},
	})

	if *createdCalls != 1 {
		t.Fatalf("createdCalls = %d, want 1", *createdCalls)
	}
	if lastCreated.SkillDelay() != 0 {
		t.Fatalf("SkillDelay() = %d, want 0 (non-zero hides the mist)", lastCreated.SkillDelay())
	}
	if lastCreated.ElemAttr() != 0 {
		t.Fatalf("ElemAttr() = %d, want 0", lastCreated.ElemAttr())
	}
	if lastCreated.NType() != 0 {
		t.Fatalf("NType() = %d, want 0 (3 is the area-buff-ITEM arm)", lastCreated.NType())
	}
	if lastCreated.Phase() != 0 {
		t.Fatalf("Phase() = %d, want 0", lastCreated.Phase())
	}
	if lastCreated.OwnerId() != 1001 {
		t.Fatalf("OwnerId() = %d, want the casting character id 1001", lastCreated.OwnerId())
	}
}

// TestMistCreated_NonZeroRenderValuesPropagate proves the values genuinely
// come from the event and are not still hard-coded to 0 -- a test that only
// ever asserts 0 would pass against the unchanged code.
func TestMistCreated_NonZeroRenderValuesPropagate(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, lastCreated, _, _ := withRecordingBroadcasters(t)
	defer restore()

	handleMistCreated(sc, nil)(logrus.New(), ctx, mist2.Event[mist2.CreatedBody]{
		Tenant: tm.Id(), WorldId: 0, ChannelId: 1, MapId: 100000000,
		Instance: uuid.Nil, MistId: uuid.New(), Type: mist2.EventTypeCreated,
		Body: mist2.CreatedBody{ElemAttr: 7, SkillDelay: 3},
	})

	if lastCreated.ElemAttr() != 7 {
		t.Fatalf("ElemAttr() = %d, want 7 (value must come from the event)", lastCreated.ElemAttr())
	}
	if lastCreated.SkillDelay() != 3 {
		t.Fatalf("SkillDelay() = %d, want 3 (value must come from the event)", lastCreated.SkillDelay())
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

// A PROTECTION mist must land in the channel's registry with its ABSOLUTE
// rect (origin + offsets) so the damage path can test a character's position
// against it directly.
func TestHandleMistCreated_RegistersProtectionMistWithAbsoluteRect(t *testing.T) {
	reg := protectionmist.NewTestProtectionRegistry()
	orig := protectionRegistry
	t.Cleanup(func() { protectionRegistry = orig })
	protectionRegistry = reg

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, _, _, _ := withRecordingBroadcasters(t)
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
			OwnerType:  "CHARACTER",
			OwnerId:    1001,
			EffectKind: mist2.EffectKindProtection,
			Type:       2,
			OriginX:    500, OriginY: 300,
			LtX: -110, LtY: -82, RbX: 110, RbY: 83,
			Duration: 31000,
		},
	})

	f := field.NewBuilder(sc.WorldId(), sc.ChannelId(), 100000000).Build()
	// origin 500,300 + lt(-110,-82)..rb(110,83) => (390,218)..(610,383)
	require.Len(t, reg.Covering(tm, f, 500, 300, time.Now()), 1)
	require.Len(t, reg.Covering(tm, f, 390, 218, time.Now()), 1)
	require.Empty(t, reg.Covering(tm, f, 389, 300, time.Now()))
	require.Equal(t, uint32(1001), reg.Covering(tm, f, 500, 300, time.Now())[0].OwnerId())
}

// Non-protection mists must NOT enter the registry -- a Poison Mist that
// shielded its caster would be a silent invulnerability.
func TestHandleMistCreated_IgnoresNonProtectionMists(t *testing.T) {
	reg := protectionmist.NewTestProtectionRegistry()
	orig := protectionRegistry
	t.Cleanup(func() { protectionRegistry = orig })
	protectionRegistry = reg

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, _, _, _ := withRecordingBroadcasters(t)
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
			OwnerType:  "CHARACTER",
			OwnerId:    1001,
			EffectKind: mist2.EffectKindDamageOverTime,
			OriginX:    500, OriginY: 300,
			LtX: -110, LtY: -82, RbX: 110, RbY: 83,
			Duration: 40000,
		},
	})

	f := field.NewBuilder(sc.WorldId(), sc.ChannelId(), 100000000).Build()
	require.Empty(t, reg.Covering(tm, f, 500, 300, time.Now()))
}

// FR-4.3: protection ends on expiry AND on cancellation.
func TestHandleMistDestroyed_EvictsTheProtection(t *testing.T) {
	reg := protectionmist.NewTestProtectionRegistry()
	orig := protectionRegistry
	t.Cleanup(func() { protectionRegistry = orig })
	protectionRegistry = reg

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, _, _, _, _ := withRecordingBroadcasters(t)
	defer restore()

	f := field.NewBuilder(sc.WorldId(), sc.ChannelId(), 100000000).Build()
	mistId := uuid.New()
	reg.Add(tm, protectionmist.NewProtectionBuilder(mistId, f).
		SetOwnerId(1001).SetRect(390, 218, 610, 383).
		SetExpiresAt(time.Now().Add(time.Minute)).Build())

	h := handleMistDestroyed(sc, nil)
	h(logrus.New(), ctx, mist2.Event[mist2.DestroyedBody]{
		Tenant:    tm.Id(),
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		MapId:     100000000,
		MistId:    mistId,
		Type:      mist2.EventTypeDestroyed,
		Body:      mist2.DestroyedBody{Reason: mist2.ReasonCancelled},
	})

	require.Empty(t, reg.Covering(tm, f, 500, 300, time.Now()))
}
