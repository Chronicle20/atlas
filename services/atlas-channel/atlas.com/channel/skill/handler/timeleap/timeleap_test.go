package timeleap

import (
	"atlas-channel/data/skill/effect"
	"context"
	"errors"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	testCharId = uint32(1001)
	testX      = int16(100)
	testY      = int16(200)
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func testField() field.Model {
	return field.NewBuilder(world.Id(1), channel.Id(0), _map.Id(100000000)).Build()
}

func testInfo(bitmap byte) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.BuccaneerTimeLeapId)).
		SetSkillLevel(1).
		SetAffectedPartyMemberBitmap(bitmap).
		Build()
}

type resetCall struct {
	transactionId  uuid.UUID
	characterId    uint32
	exceptSkillIds []uint32
	sourceSkillId  uint32
}

// invokeApply swaps the three seams, runs Apply, and returns the captured
// reset emissions.
func invokeApply(
	t *testing.T,
	casterLoader func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error),
	partySelector func(logrus.FieldLogger, context.Context, field.Model, uint32, int16, int16, effect.Model, byte) []channelhandler.PartyRecipient,
	emitErr error,
	bitmap byte,
) []resetCall {
	t.Helper()
	origCaster := loadCaster
	origParty := selectParty
	origEmit := emitReset
	t.Cleanup(func() {
		loadCaster = origCaster
		selectParty = origParty
		emitReset = origEmit
	})

	var calls []resetCall
	loadCaster = casterLoader
	if partySelector != nil {
		selectParty = partySelector
	}
	emitReset = func(_ logrus.FieldLogger, _ context.Context, transactionId uuid.UUID, _ field.Model, exceptSkillIds []uint32, sourceSkillId uint32, characterId uint32) error {
		calls = append(calls, resetCall{
			transactionId:  transactionId,
			characterId:    characterId,
			exceptSkillIds: exceptSkillIds,
			sourceSkillId:  sourceSkillId,
		})
		return emitErr
	}

	err := Apply(testLogger())(context.Background())(nil, testField(), testCharId, testInfo(bitmap), effect.Model{})
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	return calls
}

func okCasterLoader(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, error) {
	return testX, testY, nil
}

func partyOf(ids ...uint32) func(logrus.FieldLogger, context.Context, field.Model, uint32, int16, int16, effect.Model, byte) []channelhandler.PartyRecipient {
	return func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, _ int16, _ int16, _ effect.Model, _ byte) []channelhandler.PartyRecipient {
		out := make([]channelhandler.PartyRecipient, 0, len(ids))
		for _, id := range ids {
			out = append(out, channelhandler.NewPartyRecipientBuilder().SetId(id).Build())
		}
		return out
	}
}

// Solo cast: exactly one command, for the caster, excepting Time Leap.
func TestTimeLeapSoloCast_ResetsCasterOnly(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, partyOf(), nil, 0)
	if len(calls) != 1 {
		t.Fatalf("emitted %d commands, want 1", len(calls))
	}
	if calls[0].characterId != testCharId {
		t.Errorf("recipient = %d, want caster %d", calls[0].characterId, testCharId)
	}
	if len(calls[0].exceptSkillIds) != 1 || calls[0].exceptSkillIds[0] != uint32(skill2.BuccaneerTimeLeapId) {
		t.Errorf("exceptSkillIds = %v, want [5121010]", calls[0].exceptSkillIds)
	}
	if calls[0].sourceSkillId != uint32(skill2.BuccaneerTimeLeapId) {
		t.Errorf("sourceSkillId = %d, want 5121010", calls[0].sourceSkillId)
	}
}

// Party cast: one command per member + caster, one shared transactionId,
// every command excepting Time Leap.
func TestTimeLeapPartyCast_ResetsAllRecipients(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, partyOf(2002, 3003), nil, 0x06)
	if len(calls) != 3 {
		t.Fatalf("emitted %d commands, want 3", len(calls))
	}
	want := map[uint32]bool{testCharId: false, 2002: false, 3003: false}
	for _, c := range calls {
		if _, ok := want[c.characterId]; !ok {
			t.Errorf("unexpected recipient %d", c.characterId)
		}
		want[c.characterId] = true
		if c.transactionId != calls[0].transactionId {
			t.Errorf("transactionId differs across recipients: %s vs %s", c.transactionId, calls[0].transactionId)
		}
		if len(c.exceptSkillIds) != 1 || c.exceptSkillIds[0] != uint32(skill2.BuccaneerTimeLeapId) {
			t.Errorf("recipient %d exceptSkillIds = %v, want [5121010]", c.characterId, c.exceptSkillIds)
		}
	}
	for id, seen := range want {
		if !seen {
			t.Errorf("recipient %d received no command", id)
		}
	}
}

// Caster load failure: zero commands, no panic, nil error.
func TestTimeLeapCasterLoadFailure_EmitsNothing(t *testing.T) {
	failLoader := func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, error) {
		return 0, 0, errors.New("character service down")
	}
	calls := invokeApply(t, failLoader, partyOf(2002), nil, 0x02)
	if len(calls) != 0 {
		t.Fatalf("emitted %d commands after caster load failure, want 0", len(calls))
	}
}

// Missing rectangle (zero-value effect.Model) through the REAL selector:
// SelectInRangePartyMembers returns nil before any I/O, so the cast
// degrades to caster-only.
func TestTimeLeapMissingRect_CasterOnly(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, nil, nil, 0x02)
	if len(calls) != 1 {
		t.Fatalf("emitted %d commands, want 1 (caster only)", len(calls))
	}
	if calls[0].characterId != testCharId {
		t.Errorf("recipient = %d, want caster %d", calls[0].characterId, testCharId)
	}
}

// Emission failure for one recipient must not abort the rest.
func TestTimeLeapEmissionFailure_ContinuesWithRemaining(t *testing.T) {
	calls := invokeApply(t, okCasterLoader, partyOf(2002, 3003), errors.New("kafka down"), 0x06)
	if len(calls) != 3 {
		t.Fatalf("emitted %d commands, want all 3 attempted despite failures", len(calls))
	}
}

// Registration: the blank-importable init() must install the handler.
func TestTimeLeapRegistered(t *testing.T) {
	if _, ok := channelhandler.Lookup(skill2.BuccaneerTimeLeapId); !ok {
		t.Fatal("Lookup(BuccaneerTimeLeapId) returned ok=false; init() registration missing")
	}
}
