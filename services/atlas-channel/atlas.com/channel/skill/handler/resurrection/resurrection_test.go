package resurrection

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"fmt"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	testCasterId = uint32(1001)
	testLevel    = byte(7)
)

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
	for _, id := range []skill2.Id{skill2.BishopResurrectionId, skill2.GmResurrectionId, skill2.SuperGmResurrectionId} {
		h, ok := channelhandler.Lookup(id)
		if !ok || h == nil {
			t.Fatalf("Lookup(%d) = (%v, %v), want non-nil handler", id, h, ok)
		}
	}
}

func TestResurrection_SetHPBeforeWarpPerRecipient(t *testing.T) {
	events, broadcast := installHandlerSeams(t,
		[]channelhandler.PartyRecipient{mkRecipient(42, 100, 50), mkRecipient(43, -10, 20)},
		nil, nil, nil)

	err := Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
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
	err := Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
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

	_ = Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
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

	err := Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
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

	err := Apply(testLogger())(context.Background())(nil, testField(), testCasterId, bishopInfo(), effect.Model{})
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
