package monster

import (
	consumable2 "atlas-channel/kafka/message/consumable"
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/monster"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// bossHpRecord is a recorded invocation of bossHpBroadcaster, local to this
// package's tests (the map consumer package declares its own, task 7).
type bossHpRecord struct {
	monsterTemplateId uint32
	currentHp         uint32
	maxHp             uint32
}

// withRecordingBossHp swaps bossHpBroadcaster for a synchronous recording
// stub so DAMAGED handler tests can assert the FR-4/FR-6 gauge broadcast
// without standing up ForSessionsInMap or atlas-data.
func withRecordingBossHp(t *testing.T) (restore func(), records *[]bossHpRecord) {
	t.Helper()
	var recs []bossHpRecord
	orig := bossHpBroadcaster
	bossHpBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, _ field.Model, monsterTemplateId uint32, currentHp uint32, maxHp uint32) {
		recs = append(recs, bossHpRecord{monsterTemplateId: monsterTemplateId, currentHp: currentHp, maxHp: maxHp})
	}
	return func() {
		bossHpBroadcaster = orig
	}, &recs
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

func TestHandleStatusEventCreated_SeedsLiveMirror(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	prev := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
		return monster.NewBuilder(uniqueId, f, 100100).
			SetMp(60).
			SetMaxMp(90).
			SetControllerHasAggro(true).
			Build()
	}
	defer func() { monsterGetByIdFn = prev }()

	e := monster2.StatusEvent[monster2.StatusEventCreatedBody]{
		WorldId:   0,
		ChannelId: 1,
		MapId:     100000000,
		UniqueId:  7001,
		MonsterId: 100100,
		Type:      monster2.EventStatusCreated,
		Body:      monster2.StatusEventCreatedBody{ActorId: 1},
	}
	handleStatusEventCreated(sc, nil)(logrus.New(), ctx, e)

	got, ok := monster.GetLiveMirror().Lookup(tm, 7001)
	if !ok {
		t.Fatalf("CREATED must seed the mirror")
	}
	if got.MonsterId != 100100 || got.Mp != 60 || got.MaxMp != 90 || !got.ControllerHasAggro {
		t.Fatalf("seed mismatch: %+v", got)
	}
}

func TestHandleStatusEventCreated_FetchError_NoSeed(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	prev := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		return monster.Model{}, errors.New("boom")
	}
	defer func() { monsterGetByIdFn = prev }()

	e := monster2.StatusEvent[monster2.StatusEventCreatedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7002,
		MonsterId: 100100, Type: monster2.EventStatusCreated,
	}
	handleStatusEventCreated(sc, nil)(logrus.New(), ctx, e)

	if _, ok := monster.GetLiveMirror().Lookup(tm, 7002); ok {
		t.Fatalf("fetch failure must not seed the mirror")
	}
}

func TestHandleStatusEventMpChanged_UpdatesMirrorForUnknownReasonWithoutSession(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	monster.GetLiveMirror().Put(tm, 7003, monster.LiveEntry{Field: f, MonsterId: 100100, Mp: 60, MaxMp: 90, ControllerHasAggro: true})

	e := monster2.StatusEvent[monster2.StatusEventMpChangedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7003,
		MonsterId: 100100, Type: monster2.EventStatusMpChanged,
		Body: monster2.StatusEventMpChangedBody{Reason: "SKILL_CAST", Amount: 23, MonsterMpAfter: 37},
	}
	// No session exists for CharacterId 0 — the mirror update must land anyway.
	handleStatusEventMpChanged(sc, nil)(logrus.New(), ctx, e)

	got, ok := monster.GetLiveMirror().Lookup(tm, 7003)
	if !ok || got.Mp != 37 {
		t.Fatalf("MP_CHANGED must update mirror before session gating / reason dispatch, got %+v ok=%v", got, ok)
	}
	if !got.ControllerHasAggro {
		t.Fatalf("MP update must not clobber aggro")
	}
}

func TestHandleStatusEventStartStopAggro_UpdateMirrorAggro(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	monster.GetLiveMirror().Put(tm, 7004, monster.LiveEntry{Field: f, MonsterId: 100100})

	sce := monster2.StatusEvent[monster2.StatusEventStartControlBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7004,
		MonsterId: 100100, Type: monster2.EventStatusStartControl,
		Body: monster2.StatusEventStartControlBody{ActorId: 1, ControllerHasAggro: true},
	}
	handleStatusEventStartControl(sc, nil)(logrus.New(), ctx, sce)
	if got, _ := monster.GetLiveMirror().Lookup(tm, 7004); !got.ControllerHasAggro {
		t.Fatalf("START_CONTROL must set aggro from body")
	}

	ste := monster2.StatusEvent[monster2.StatusEventStopControlBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7004,
		MonsterId: 100100, Type: monster2.EventStatusStopControl,
		Body: monster2.StatusEventStopControlBody{ActorId: 1},
	}
	handleStatusEventStopControl(sc, nil)(logrus.New(), ctx, ste)
	if got, _ := monster.GetLiveMirror().Lookup(tm, 7004); got.ControllerHasAggro {
		t.Fatalf("STOP_CONTROL must clear aggro (no controller => no aggro)")
	}

	ace := monster2.StatusEvent[monster2.StatusEventAggroChangedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7004,
		MonsterId: 100100, Type: monster2.EventStatusAggroChanged,
		Body: monster2.StatusEventAggroChangedBody{ControllerCharacterId: 1, ControllerHasAggro: true},
	}
	handleStatusEventAggroChanged(sc, nil)(logrus.New(), ctx, ace)
	if got, _ := monster.GetLiveMirror().Lookup(tm, 7004); !got.ControllerHasAggro {
		t.Fatalf("AGGRO_CHANGED must set aggro from body")
	}
}

func TestHandleStatusEventDestroyedAndKilled_RemoveMirrorEntry(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)
	f := field.NewBuilder(0, 1, 100000000).Build()

	monster.GetLiveMirror().Put(tm, 7005, monster.LiveEntry{Field: f, MonsterId: 100100})
	de := monster2.StatusEvent[monster2.StatusEventDestroyedBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7005,
		MonsterId: 100100, Type: monster2.EventStatusDestroyed,
	}
	handleStatusEventDestroyed(sc, nil)(logrus.New(), ctx, de)
	if _, ok := monster.GetLiveMirror().Lookup(tm, 7005); ok {
		t.Fatalf("DESTROYED must evict the mirror entry")
	}

	monster.GetLiveMirror().Put(tm, 7006, monster.LiveEntry{Field: f, MonsterId: 100100})
	ke := monster2.StatusEvent[monster2.StatusEventKilledBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7006,
		MonsterId: 100100, Type: monster2.EventStatusKilled,
	}
	handleStatusEventKilled(sc, nil)(logrus.New(), ctx, ke)
	if _, ok := monster.GetLiveMirror().Lookup(tm, 7006); ok {
		t.Fatalf("KILLED must evict the mirror entry")
	}
}

// withRecordingControlGrant swaps the control-grant seam for a stub that
// records what each grant delivered, so a test can assert Spawn-then-Control
// ordering without standing up a session.
type grantRecord struct {
	characterId uint32
	uniqueId    uint32
	monsterId   uint32
	aggro       bool
}

func withRecordingControlGrant(t *testing.T) (restore func(), grants *[]grantRecord) {
	t.Helper()
	var recorded []grantRecord
	orig := controlGrantFn
	controlGrantFn = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, m monster.Model, aggro bool, characterId uint32) error {
		recorded = append(recorded, grantRecord{
			characterId: characterId,
			uniqueId:    m.UniqueId(),
			monsterId:   m.MonsterId(),
			aggro:       aggro,
		})
		return nil
	}
	return func() { controlGrantFn = orig }, &recorded
}

// TestHandleStatusEventStartControl_GrantsThroughSpawnThenControl is the
// channel half of the frozen-monsters-on-re-entry regression.
//
// atlas-monsters now always emits StartControl for a map-enter assignment
// (it used to assign the entering player silently and rely on the channel's
// map-enter spawn, which races ahead of it and loses the grant entirely). That
// only works if this handler actually delivers the grant, and delivers it as
// Spawn-then-Control — a bare Control for a mob the client has not been told
// about makes v79/v83 materialize the mob from the Control body.
func TestHandleStatusEventStartControl_GrantsThroughSpawnThenControl(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	prev := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
		return monster.NewBuilder(uniqueId, f, 100100).
			SetControlCharacterId(42).
			SetControllerHasAggro(true).
			Build()
	}
	defer func() { monsterGetByIdFn = prev }()

	restore, grants := withRecordingControlGrant(t)
	defer restore()

	e := monster2.StatusEvent[monster2.StatusEventStartControlBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7010,
		MonsterId: 100100, Type: monster2.EventStatusStartControl,
		Body: monster2.StatusEventStartControlBody{ActorId: 42, ControllerHasAggro: true},
	}
	handleStatusEventStartControl(sc, nil)(logrus.New(), ctx, e)

	if len(*grants) != 1 {
		t.Fatalf("START_CONTROL must deliver exactly one control grant; got %d", len(*grants))
	}
	g := (*grants)[0]
	if g.characterId != 42 || g.uniqueId != 7010 || g.monsterId != 100100 || !g.aggro {
		t.Fatalf("grant mismatch: %+v", g)
	}
}

// TestHandleStatusEventStartControl_FetchFailureStillGrants pins the fallback:
// a REST failure must degrade to the event envelope, not swallow the grant.
// Dropping it would reproduce the original bug — controller assigned upstream,
// client frozen forever, with nothing to retrigger the grant.
func TestHandleStatusEventStartControl_FetchFailureStillGrants(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	prev := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		return monster.Model{}, errors.New("boom")
	}
	defer func() { monsterGetByIdFn = prev }()

	restore, grants := withRecordingControlGrant(t)
	defer restore()

	e := monster2.StatusEvent[monster2.StatusEventStartControlBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 7011,
		MonsterId: 100100, Type: monster2.EventStatusStartControl,
		Body: monster2.StatusEventStartControlBody{ActorId: 43},
	}
	handleStatusEventStartControl(sc, nil)(logrus.New(), ctx, e)

	if len(*grants) != 1 {
		t.Fatalf("a REST failure must still deliver the grant from the envelope; got %d grants", len(*grants))
	}
	if g := (*grants)[0]; g.characterId != 43 || g.uniqueId != 7011 || g.monsterId != 100100 {
		t.Fatalf("fallback grant mismatch: %+v", g)
	}
}

// TestSpawnThenControlOperator_EmitsSpawnBeforeControl pins the packet order
// that keeps a control grant safe for a client that may not have the mob yet.
//
// A MonsterControl for an unknown mob is not dropped by the client:
// CMobPool::SetLocalMob (v83 0x678308, v79 0x645ce1) misses on GetMob and
// materializes the mob from the Control body via CreateMob -> CMob::Init,
// which null-derefs on a 0/1 stance and mis-seats the mob on a slope. The
// leading Spawn is harmless when the client already has the mob:
// CMobPool::OnMobEnterField's GetMob-hit branch (v83 0x67945a, v79 0x646e33)
// only sets m_bInViewSplit and re-applies temporary stats.
func TestSpawnThenControlOperator_EmitsSpawnBeforeControl(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	var order []string
	orig := announceFn
	announceFn = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, _ packet.Encode, _ session.Model) error {
		order = append(order, writerName)
		return nil
	}
	defer func() { announceFn = orig }()

	f := field.NewBuilder(0, 1, 100000000).Build()
	m := monster.NewBuilder(7020, f, 100100).SetControlCharacterId(42).MustBuild()

	if err := spawnThenControlOperator(logrus.New(), ctx, nil, m, false)(session.Model{}); err != nil {
		t.Fatalf("spawnThenControlOperator: %v", err)
	}

	want := []string{monsterpkt.MonsterSpawnWriter, monsterpkt.MonsterControlWriter}
	if len(order) != len(want) {
		t.Fatalf("want %d packets %v; got %d %v", len(want), want, len(order), order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("packet order mismatch: want %v, got %v", want, order)
		}
	}
}

// TestSpawnThenControlOperator_SpawnFailureSuppressesControl guards the
// ordering invariant under error: if Spawn could not be delivered, Control
// must not go out on its own, or the client materializes the mob from the
// Control body — the exact crash/fall-through the ordering exists to prevent.
func TestSpawnThenControlOperator_SpawnFailureSuppressesControl(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	var order []string
	orig := announceFn
	announceFn = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, _ packet.Encode, _ session.Model) error {
		order = append(order, writerName)
		if writerName == monsterpkt.MonsterSpawnWriter {
			return errors.New("spawn write failed")
		}
		return nil
	}
	defer func() { announceFn = orig }()

	f := field.NewBuilder(0, 1, 100000000).Build()
	m := monster.NewBuilder(7021, f, 100100).SetControlCharacterId(42).MustBuild()

	if err := spawnThenControlOperator(logrus.New(), ctx, nil, m, false)(session.Model{}); err == nil {
		t.Fatalf("a failed Spawn must surface as an error")
	}
	if len(order) != 1 || order[0] != monsterpkt.MonsterSpawnWriter {
		t.Fatalf("Control must not follow a failed Spawn; got %v", order)
	}
}

// A DoT tick must NOT get a server-side MonsterDamage packet: the client
// renders poison ticks itself from the POISON magnitude in the temporary-stat
// packet. Echoing one here previously sent the monster's CUMULATIVE
// per-character damage total (the last DamageEntries element) as if it were
// the tick, which read as five-figure numbers on a 15,200 HP mob.
func TestShouldEchoDamagePacket(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{source: monster2.DamageSourceMonsterAttack, want: true},
		{source: monster2.DamageSourceDamageOverTime, want: false},
		{source: monster2.DamageSourceCharacterAttack, want: false},
		{source: monster2.DamageSourceHeal, want: false},
		{source: "", want: false},
	}
	for _, tt := range tests {
		if got := shouldEchoDamagePacket(tt.source); got != tt.want {
			t.Errorf("shouldEchoDamagePacket(%q) = %v, want %v", tt.source, got, tt.want)
		}
	}
}

// TestHandleStatusEventDamaged_BossHpGauge covers FR-4 (boss damage
// broadcasts the gauge), FR-6 (an echo-suppressed damage source still
// broadcasts the gauge) and NFR-1 (a non-boss event never reaches the
// resolver).
func TestHandleStatusEventDamaged_BossHpGauge(t *testing.T) {
	tests := []struct {
		name         string
		boss         bool
		damageSource string
		fetchErr     error
		want         []bossHpRecord
	}{
		{
			name:         "boss damage broadcasts (FR-4)",
			boss:         true,
			damageSource: monster2.DamageSourceMonsterAttack,
			want:         []bossHpRecord{{monsterTemplateId: 8800002, currentHp: 50000, maxHp: 100000}},
		},
		{
			name:         "echo-suppressed source still broadcasts (FR-6)",
			boss:         true,
			damageSource: monster2.DamageSourceCharacterAttack,
			want:         []bossHpRecord{{monsterTemplateId: 8800002, currentHp: 50000, maxHp: 100000}},
		},
		{
			name:         "non-boss event does not broadcast (NFR-1 pre-filter)",
			boss:         false,
			damageSource: monster2.DamageSourceMonsterAttack,
			want:         nil,
		},
		{
			name:         "monster fetch failure aborts before the gauge",
			boss:         true,
			damageSource: monster2.DamageSourceMonsterAttack,
			fetchErr:     errors.New("boom"),
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tm)
			sc := newTestServer(t, tm)

			f := field.NewBuilder(0, 1, 100000000).Build()
			prevGet := monsterGetByIdFn
			monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
				if tt.fetchErr != nil {
					return monster.Model{}, tt.fetchErr
				}
				return monster.NewBuilder(uniqueId, f, 8800002).SetHp(50000).SetMaxHp(100000).MustBuild(), nil
			}
			defer func() { monsterGetByIdFn = prevGet }()

			restore, records := withRecordingBossHp(t)
			defer restore()

			e := monster2.StatusEvent[monster2.StatusEventDamagedBody]{
				WorldId:   0,
				ChannelId: 1,
				MapId:     100000000,
				UniqueId:  7101,
				MonsterId: 8800002,
				Type:      monster2.EventStatusDamaged,
				Body: monster2.StatusEventDamagedBody{
					Boss:         tt.boss,
					DamageSource: tt.damageSource,
				},
			}
			handleStatusEventDamaged(sc, nil)(logrus.New(), ctx, e)

			if len(*records) != len(tt.want) {
				t.Fatalf("records = %+v, want %+v", *records, tt.want)
			}
			for i, r := range *records {
				if r != tt.want[i] {
					t.Errorf("record[%d] = %+v, want %+v", i, r, tt.want[i])
				}
			}
		})
	}
}

// TestHandleStatusEventDamaged_GaugeDoesNotDisturbHealthPath covers FR-5:
// the gauge broadcast must reuse the health path's single monster fetch and
// must not prevent the health/damage-echo goroutines from being launched.
// The MonsterHealth wire effect itself is not observable from this
// package's tests (session.Announce writes through a nil connection here);
// the live smoke (AC-13) is what actually proves the two bars coexist.
func TestHandleStatusEventDamaged_GaugeDoesNotDisturbHealthPath(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	f := field.NewBuilder(0, 1, 100000000).Build()
	fetchCalls := 0
	prevGet := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, uniqueId uint32) (monster.Model, error) {
		fetchCalls++
		return monster.NewBuilder(uniqueId, f, 8800002).SetHp(50000).SetMaxHp(100000).MustBuild(), nil
	}
	defer func() { monsterGetByIdFn = prevGet }()

	restore, records := withRecordingBossHp(t)
	defer restore()

	e := monster2.StatusEvent[monster2.StatusEventDamagedBody]{
		WorldId:   0,
		ChannelId: 1,
		MapId:     100000000,
		UniqueId:  7101,
		MonsterId: 8800002,
		Type:      monster2.EventStatusDamaged,
		Body: monster2.StatusEventDamagedBody{
			Boss:         true,
			DamageSource: monster2.DamageSourceMonsterAttack,
		},
	}
	handleStatusEventDamaged(sc, nil)(logrus.New(), ctx, e)

	if fetchCalls != 1 {
		t.Fatalf("monsterGetByIdFn calls = %d, want exactly 1", fetchCalls)
	}
	if len(*records) != 1 {
		t.Fatalf("boss HP records = %+v, want exactly 1", *records)
	}
}

// TestBridleFailReason maps internal causes onto the only two values the client
// understands. CWvsContext::OnBridleMobCatchFail @0x9d9a80 branches on exactly
// two: 0 renders string 0x110E, 1 renders the item's delayMsg (falling back to
// 0x110F), and ANY other value renders nothing at all. Reason 1 is reserved for
// the not-yet/try-again case, which is why useDelay is server-enforced.
// UNRESOLVED sends no packet: the request was legitimate and lost a race, so the
// client should simply unlock.
func TestBridleFailReason(t *testing.T) {
	cases := []struct {
		cause      string
		wantReason byte
		wantSend   bool
	}{
		{monster2.CatchCauseSpeciesMismatch, 0, true},
		{monster2.CatchCauseHpTooHigh, 0, true},
		{monster2.CatchCauseRollFailed, 0, true},
		{consumable2.CatchCauseInventoryFull, 0, true},
		{consumable2.CatchCauseInvalidItem, 0, true},
		{consumable2.CatchCauseUseDelay, 1, true},
		{monster2.CatchCauseUnresolved, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.cause, func(t *testing.T) {
			reason, send := bridleFailReason(tc.cause)
			if reason != tc.wantReason || send != tc.wantSend {
				t.Fatalf("bridleFailReason(%q) = (%d, %t), want (%d, %t)", tc.cause, reason, send, tc.wantReason, tc.wantSend)
			}
		})
	}
}

// TestDestroyCodeFor maps a KILLED/DESTROYED event's DeathType* semantic key
// onto the DestroyMonster writer's operations-table key (DOM-25). The empty
// string means "the producer did not set it" -- an old atlas-monsters
// mid-rolling-deploy -- and renders as fade-out, byte-identical to the
// pre-task-253 hardcode (task-253 design D9).
func TestDestroyCodeFor(t *testing.T) {
	tests := []struct {
		name      string
		deathType string
		want      writer.DestroyMonsterCode
	}{
		{"producer omitted the field", "", writer.DestroyMonsterFadeOut},
		{"ordinary fade-out", "FADE_OUT", writer.DestroyMonsterFadeOut},
		{"bomb", "BOMB", writer.DestroyMonsterBomb},
		{"destruct-by-miss", "DESTRUCT_BY_MISS", writer.DestroyMonsterDestructByMiss},
		{"swallow", "SWALLOW", writer.DestroyMonsterSwallow},
		{"self-destruct", "SELF_DESTRUCT", writer.DestroyMonsterSelfDestruct},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := destroyCodeFor(tt.deathType); got != tt.want {
				t.Fatalf("destroyCodeFor(%q) = %q, want %q", tt.deathType, got, tt.want)
			}
		})
	}
}

// TestStatusEventKilledBodyDecodesDeathType asserts the rolling-deploy
// compatibility contract (task-253 design D9): an old atlas-monsters that
// never emits "deathType" decodes to the empty string, which destroyCodeFor
// renders as fade-out.
func TestStatusEventKilledBodyDecodesDeathType(t *testing.T) {
	var withField monster2.StatusEventKilledBody
	if err := json.Unmarshal([]byte(`{"x":0,"y":0,"actorId":9,"boss":false,"damageEntries":null,"deathType":"DESTRUCT_BY_MISS"}`), &withField); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if withField.DeathType != "DESTRUCT_BY_MISS" {
		t.Fatalf("DeathType = %q, want DESTRUCT_BY_MISS", withField.DeathType)
	}

	var withoutField monster2.StatusEventKilledBody
	if err := json.Unmarshal([]byte(`{"x":0,"y":0,"actorId":9,"boss":false,"damageEntries":null}`), &withoutField); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if withoutField.DeathType != "" {
		t.Fatalf("DeathType = %q, want empty", withoutField.DeathType)
	}
}
