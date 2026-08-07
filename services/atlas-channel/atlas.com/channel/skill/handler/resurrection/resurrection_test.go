package resurrection

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"fmt"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	testCasterId = uint32(1001)
	testLevel    = byte(7)
)

// testCtx builds a tenant-bearing context: Apply resolves the cast's wire
// skill id through the tenant's version set (task-187), so it can no longer
// run against a bare context.Background().
func testCtx(t *testing.T, region string, major, minor uint16) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func bishopInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.BishopResurrectionId)).
		SetSkillLevel(1).
		SetAffectedPartyMemberBitmap(0x7E).
		Build()
}

// installHandlerSeams swaps every Apply seam with deterministic stubs and
// returns a pointer to the recorded event log and whether broadcastEffects fired.
// Pass nil for setHPErr or warpErr to have those stubs always succeed.
func installHandlerSeams(
	t *testing.T,
	recipients []channelhandler.PartyRecipient,
	casterErr error,
	setHPErr map[uint32]error,
	warpErr map[uint32]error,
) (*[]string, *bool) {
	t.Helper()
	prevCaster, prevParty, prevMap := loadCaster, selectDeadParty, selectDeadMap
	prevSetHP, prevWarp, prevBroadcast := setHP, warpToPosition, broadcastEffects

	events := []string{}
	broadcastCalled := false

	loadCaster = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, byte, error) {
		return 0, 0, testLevel, casterErr
	}
	selectDeadParty = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model, _ byte) []channelhandler.PartyRecipient {
		return recipients
	}
	selectDeadMap = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model) []channelhandler.PartyRecipient {
		return recipients
	}
	setHP = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, id uint32, amount uint16) error {
		events = append(events, fmt.Sprintf("setHP:%d:%d", id, amount))
		if setHPErr != nil {
			return setHPErr[id]
		}
		return nil
	}
	warpToPosition = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, id uint32, x, y int16) error {
		events = append(events, fmt.Sprintf("warp:%d:%d:%d", id, x, y))
		if warpErr != nil {
			return warpErr[id]
		}
		return nil
	}
	broadcastEffects = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, _ uint32, _ byte, _ uint32, _ byte) {
		broadcastCalled = true
	}

	t.Cleanup(func() {
		loadCaster, selectDeadParty, selectDeadMap = prevCaster, prevParty, prevMap
		setHP, warpToPosition, broadcastEffects = prevSetHP, prevWarp, prevBroadcast
	})
	return &events, &broadcastCalled
}

func mkRecipient(id uint32, x, y int16) channelhandler.PartyRecipient {
	return channelhandler.NewPartyRecipientBuilder().SetId(id).SetX(x).SetY(y).Build()
}

func TestResurrection_RegistersAllThreeIds(t *testing.T) {
	for _, id := range []skill2.Identity{skill2.BishopResurrection, skill2.GmResurrection, skill2.SuperGmResurrection} {
		h, ok := channelhandler.Lookup(id)
		if !ok || h == nil {
			t.Fatalf("Lookup(%v) = (%v, %v), want non-nil handler", id, h, ok)
		}
	}
}

// TestResurrection_v48SuperGmWireUsesMapSelector proves the task-187
// wire-to-identity resolution end to end for the divergent Resurrection
// variant: at v0.48, SuperGmResurrection is wire 5101005 (NOT the canonical
// 9101005). A v48 cast of wire 5101005 must resolve to SuperGmResurrection
// and take the map-scoped (party-agnostic) recipient path, not the
// party-scoped Bishop path.
func TestResurrection_v48SuperGmWireUsesMapSelector(t *testing.T) {
	prevParty, prevMap := selectDeadParty, selectDeadMap
	partyCalled, mapCalled := false, false
	selectDeadParty = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model, _ byte) []channelhandler.PartyRecipient {
		partyCalled = true
		return nil
	}
	selectDeadMap = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _, _ int16, _ effect.Model) []channelhandler.PartyRecipient {
		mapCalled = true
		return nil
	}
	t.Cleanup(func() { selectDeadParty, selectDeadMap = prevParty, prevMap })

	prevCaster, prevBroadcast := loadCaster, broadcastEffects
	loadCaster = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, byte, error) {
		return 0, 0, testLevel, nil
	}
	broadcastEffects = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, _ uint32, _ byte, _ uint32, _ byte) {
	}
	t.Cleanup(func() { loadCaster, broadcastEffects = prevCaster, prevBroadcast })

	v48SuperGmInfo := packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(5101005).
		SetSkillLevel(1).
		Build()

	err := Apply(testLogger())(testCtx(t, "GMS", 48, 1))(nil, testField(), testCasterId, v48SuperGmInfo, effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	if partyCalled || !mapCalled {
		t.Fatalf("v48 wire 5101005 (SuperGmResurrection): partyCalled=%v mapCalled=%v, want map only", partyCalled, mapCalled)
	}
}

func TestResurrection_SetHPBeforeWarpPerRecipient(t *testing.T) {
	events, broadcast := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 100, 50), mkRecipient(43, -10, 20)},
		nil, nil, nil)

	err := Apply(testLogger())(testCtx(t, "GMS", 83, 1))(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	want := []string{"setHP:42:65535", "warp:42:100:50", "setHP:43:65535", "warp:43:-10:20"}
	if fmt.Sprint(*events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", *events, want)
	}
	if !*broadcast {
		t.Fatal("broadcastEffects not called")
	}
}

func TestResurrection_EmptyRecipientsBroadcastsNoSetHP(t *testing.T) {
	events, broadcast := installHandlerSeams(t, nil, nil, nil, nil)
	err := Apply(testLogger())(testCtx(t, "GMS", 83, 1))(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("events = %v, want none (no recipients)", *events)
	}
	if !*broadcast {
		t.Fatal("broadcastEffects must fire even with no recipients")
	}
}

func TestResurrection_PerRecipientFailureIsolation(t *testing.T) {
	events, broadcast := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 0, 0), mkRecipient(43, 0, 0)},
		nil,
		map[uint32]error{42: errors.New("setHP boom")},
		nil)

	_ = Apply(testLogger())(testCtx(t, "GMS", 83, 1))(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	want := []string{"setHP:42:65535", "setHP:43:65535", "warp:43:0:0"}
	if fmt.Sprint(*events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", *events, want)
	}
	if !*broadcast {
		t.Fatal("broadcastEffects must fire even when some recipients fail SetHP")
	}
}

func TestResurrection_CasterLoadErrorNoOp(t *testing.T) {
	events, broadcast := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 0, 0)},
		errors.New("caster load failed"), nil, nil)

	err := Apply(testLogger())(testCtx(t, "GMS", 83, 1))(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("events = %v, want none on caster load failure", *events)
	}
	if *broadcast {
		t.Fatal("broadcastEffects must not fire on caster load failure")
	}
}

// TestResurrection_WarpFailureIsolation verifies that a warpToPosition error for
// one recipient does not abort processing of subsequent recipients, and that
// broadcastEffects still fires. The warp stub records the attempt before
// returning the error, so the event log includes the failed warp call.
func TestResurrection_WarpFailureIsolation(t *testing.T) {
	events, broadcast := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 10, 20), mkRecipient(43, 30, 40)},
		nil,
		nil,
		map[uint32]error{42: errors.New("warp boom")})

	err := Apply(testLogger())(testCtx(t, "GMS", 83, 1))(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
	if err != nil {
		t.Fatalf("Apply err: %v", err)
	}
	// Both setHP calls and both warp calls are attempted; 42's warp returns an
	// error and is skipped, but 43 proceeds to completion.
	want := []string{"setHP:42:65535", "warp:42:10:20", "setHP:43:65535", "warp:43:30:40"}
	if fmt.Sprint(*events) != fmt.Sprint(want) {
		t.Fatalf("events = %v, want %v", *events, want)
	}
	if !*broadcast {
		t.Fatal("broadcastEffects must fire even when some recipients fail warp")
	}
}
