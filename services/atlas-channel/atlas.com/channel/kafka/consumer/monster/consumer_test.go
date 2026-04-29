package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/monster"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// withRecordingBroadcasters swaps the package-level broadcast seams for
// recording stubs that count invocations. Returns a restore func and
// pointers to the call counters. Tests use this to assert the
// MonsterStatSet/MonsterStatReset wire effect of the venom collapse
// gate without standing up a REST mock for ForSessionsInMap.
func withRecordingBroadcasters(t *testing.T) (restore func(), setCalls *int, resetCalls *int) {
	t.Helper()
	setN, resetN := 0, 0
	origSet := monsterStatSetBroadcaster
	origReset := monsterStatResetBroadcaster
	monsterStatSetBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, _ field.Model, _ uint32, _ *packetmodel.MonsterTemporaryStat) {
		setN++
	}
	monsterStatResetBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, _ field.Model, _ uint32, _ *packetmodel.MonsterTemporaryStat) {
		resetN++
	}
	return func() {
		monsterStatSetBroadcaster = origSet
		monsterStatResetBroadcaster = origReset
	}, &setN, &resetN
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
	return server.Register(tm, ch, "127.0.0.1", 8484)
}

func TestHandleNextSkillDecided_PutsIntoInbox(t *testing.T) {
	monster.InitNextSkillInbox()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	sc := newTestServer(t, tm)
	h := handleStatusEventNextSkillDecided(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEventNextSkillDecidedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  42,
		Type:      monster2.EventStatusNextSkillDecided,
		Body: monster2.StatusEventNextSkillDecidedBody{
			SkillId:     100,
			SkillLevel:  1,
			DecidedAtMs: 12345,
		},
	})

	d, ok := monster.GetNextSkillInbox().TakeAndClear(tm, 42)
	if !ok || d.SkillId != 100 {
		t.Fatalf("expected inbox to have decision; got ok=%v skill=%d", ok, d.SkillId)
	}
}

func TestHandleNextSkillDecided_WrongType_DoesNotPut(t *testing.T) {
	monster.InitNextSkillInbox()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	sc := newTestServer(t, tm)
	h := handleStatusEventNextSkillDecided(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEventNextSkillDecidedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  99,
		Type:      "WRONG_TYPE",
		Body: monster2.StatusEventNextSkillDecidedBody{
			SkillId: 100,
		},
	})

	_, ok := monster.GetNextSkillInbox().TakeAndClear(tm, 99)
	if ok {
		t.Fatalf("expected no entry for wrong event type")
	}
}

// TestHandleStatusEffectApplied_PopulatesStatusMirror verifies that a
// STATUS_APPLIED event carrying a PHYSICAL reflect window is mirrored
// into the in-process StatusMirror so that GetReflect returns the
// reflect info. This is the regression target for Task 11 — guards
// against the wire body / mirror body fields drifting apart and
// against the handler skipping the mirror call. Uses synthetic per-
// test uniqueIds for singleton isolation since the mirror is process-
// wide and self-initialised via sync.Once.
func TestHandleStatusEffectApplied_PopulatesStatusMirror(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	uniqueId := uint32(424242)
	defer monster.GetStatusMirror().OnMonsterGone(tm, uniqueId)

	h := handleStatusEffectApplied(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEffectAppliedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  uniqueId,
		Type:      monster2.EventStatusEffectApplied,
		Body: monster2.StatusEffectAppliedBody{
			EffectId:          uuid.NewString(),
			SourceType:        "CHARACTER",
			SourceCharacterId: 99,
			SourceSkillId:     1311006,
			SourceSkillLevel:  1,
			Statuses:          map[string]int32{"WEAPON_REFLECT": 1},
			Duration:          60000,
			ReflectKind:       "PHYSICAL",
			ReflectPercent:    40,
			ReflectLtX:        -150,
			ReflectLtY:        -150,
			ReflectRbX:        150,
			ReflectRbY:        150,
			ReflectMaxDamage:  5000,
		},
	})

	ri, ok := monster.GetStatusMirror().GetReflect(tm, uniqueId, "PHYSICAL")
	if !ok {
		t.Fatalf("expected PHYSICAL reflect to be present after STATUS_APPLIED handler ran")
	}
	if ri.Percent != 40 {
		t.Fatalf("Percent: want 40, got %d", ri.Percent)
	}
	if ri.LtX != -150 || ri.LtY != -150 || ri.RbX != 150 || ri.RbY != 150 {
		t.Fatalf("reflect bounds wrong: %+v", ri)
	}
	if ri.MaxDamage != 5000 {
		t.Fatalf("MaxDamage: want 5000, got %d", ri.MaxDamage)
	}
	if _, ok := monster.GetStatusMirror().GetReflect(tm, uniqueId, "MAGIC"); ok {
		t.Fatalf("MAGIC lookup should miss when only PHYSICAL is mirrored")
	}
}

// applyVenom is a helper that synthesises a STATUS_APPLIED event for a
// single VENOM stack and runs the apply handler. effectId is generated
// fresh per call so each apply represents a distinct stack in the
// status mirror.
func applyVenom(t *testing.T, sc server.Model, ctx context.Context, uniqueId uint32) string {
	t.Helper()
	effectId := uuid.NewString()
	h := handleStatusEffectApplied(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEffectAppliedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  uniqueId,
		Type:      monster2.EventStatusEffectApplied,
		Body: monster2.StatusEffectAppliedBody{
			EffectId:         effectId,
			SourceType:       "CHARACTER",
			SourceSkillId:    4120005,
			SourceSkillLevel: 1,
			Statuses:         map[string]int32{"VENOM": 1},
			Duration:         8000,
		},
	})
	return effectId
}

func expireVenom(t *testing.T, sc server.Model, ctx context.Context, uniqueId uint32, effectId string) {
	t.Helper()
	h := handleStatusEffectExpired(sc, nil)
	h(logrus.New(), ctx, monster2.StatusEvent[monster2.StatusEffectExpiredBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		UniqueId:  uniqueId,
		Type:      monster2.EventStatusEffectExpired,
		Body: monster2.StatusEffectExpiredBody{
			EffectId: effectId,
			Statuses: map[string]int32{"VENOM": 1},
		},
	})
}

// TestHandleStatusEffectApplied_VenomFirstApply_BroadcastsMonsterStatSet
// verifies the 0->1 transition: the first VENOM apply on a clean
// monster fires exactly one MonsterStatSet broadcast.
func TestHandleStatusEffectApplied_VenomFirstApply_BroadcastsMonsterStatSet(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	uniqueId := uint32(900001)
	defer monster.GetStatusMirror().OnMonsterGone(tm, uniqueId)

	restore, setCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	applyVenom(t, sc, ctx, uniqueId)

	if *setCalls != 1 {
		t.Fatalf("first VENOM apply: want 1 MonsterStatSet broadcast, got %d", *setCalls)
	}
	if c := monster.GetStatusMirror().VenomCount(tm, uniqueId); c != 1 {
		t.Fatalf("VenomCount after first apply: want 1, got %d", c)
	}
}

// TestHandleStatusEffectApplied_VenomSecondAndThirdApply_DoesNotBroadcast
// verifies wire-collapse: only the first apply (0->1) broadcasts; the
// 1->2 and 2->3 transitions are suppressed at the wire.
func TestHandleStatusEffectApplied_VenomSecondAndThirdApply_DoesNotBroadcast(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	uniqueId := uint32(900002)
	defer monster.GetStatusMirror().OnMonsterGone(tm, uniqueId)

	restore, setCalls, _ := withRecordingBroadcasters(t)
	defer restore()

	applyVenom(t, sc, ctx, uniqueId)
	applyVenom(t, sc, ctx, uniqueId)
	applyVenom(t, sc, ctx, uniqueId)

	if *setCalls != 1 {
		t.Fatalf("three sequential VENOM applies: want 1 MonsterStatSet broadcast (only the 0->1 transition), got %d", *setCalls)
	}
	if c := monster.GetStatusMirror().VenomCount(tm, uniqueId); c != 3 {
		t.Fatalf("VenomCount after three applies: want 3, got %d", c)
	}
}

// TestHandleStatusEffectExpired_VenomLastSlot_BroadcastsMonsterStatReset
// verifies the inverse: only the last VENOM expiry (N->0) broadcasts a
// MonsterStatReset; intermediate expiries are suppressed.
func TestHandleStatusEffectExpired_VenomLastSlot_BroadcastsMonsterStatReset(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	uniqueId := uint32(900003)
	defer monster.GetStatusMirror().OnMonsterGone(tm, uniqueId)

	restore, _, resetCalls := withRecordingBroadcasters(t)
	defer restore()

	id1 := applyVenom(t, sc, ctx, uniqueId)
	id2 := applyVenom(t, sc, ctx, uniqueId)
	id3 := applyVenom(t, sc, ctx, uniqueId)

	expireVenom(t, sc, ctx, uniqueId, id1)
	expireVenom(t, sc, ctx, uniqueId, id2)
	if *resetCalls != 0 {
		t.Fatalf("after expiring 2 of 3 VENOM stacks: want 0 MonsterStatReset broadcasts, got %d", *resetCalls)
	}
	if c := monster.GetStatusMirror().VenomCount(tm, uniqueId); c != 1 {
		t.Fatalf("VenomCount after 2 expiries: want 1, got %d", c)
	}

	expireVenom(t, sc, ctx, uniqueId, id3)
	if *resetCalls != 1 {
		t.Fatalf("after expiring last VENOM stack: want 1 MonsterStatReset broadcast, got %d", *resetCalls)
	}
	if c := monster.GetStatusMirror().VenomCount(tm, uniqueId); c != 0 {
		t.Fatalf("VenomCount after all expiries: want 0, got %d", c)
	}
}
