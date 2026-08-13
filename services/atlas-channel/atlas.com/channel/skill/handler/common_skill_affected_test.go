package handler

import (
	"atlas-channel/character"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// skillAffectedCall records one production emit the seam intercepted.
type skillAffectedCall struct {
	recipientId uint32
	skillId     uint32
	skillLevel  byte
}

// captureSkillAffected swaps the per-recipient emit seam for a recorder.
func captureSkillAffected(t *testing.T) *[]skillAffectedCall {
	t.Helper()
	calls := make([]skillAffectedCall, 0)
	prev := skillAffectedEmitFunc
	skillAffectedEmitFunc = func(
		_ logrus.FieldLogger, _ context.Context, _ writer.Producer,
		_ field.Model, recipientId uint32, skillId uint32, skillLevel byte,
	) {
		calls = append(calls, skillAffectedCall{recipientId, skillId, skillLevel})
	}
	t.Cleanup(func() { skillAffectedEmitFunc = prev })
	return &calls
}

// buffedParty installs the party seams for a three-person party whose members
// are all online, alive, and in the caster's map.
func buffedParty(t *testing.T) {
	t.Helper()
	a := mkPartyMember(testMemberA, true, channel.Id(0), _map.Id(40000))
	b := mkPartyMember(testMemberB, true, channel.Id(0), _map.Id(40000))
	installPartySeams(t, threePersonParty(a, b), nil,
		map[uint32]struct{}{testCasterId: {}, testMemberA: {}, testMemberB: {}},
		map[uint32]character.Model{
			testCasterId: mkMemberChar(testCasterId, 500),
			testMemberA:  mkMemberChar(testMemberA, 500),
			testMemberB:  mkMemberChar(testMemberB, 500),
		},
	)
}

// TestApplyToPartyReturnsAppliedIds pins that applyToParty reports exactly the
// members it applied the buff to — announceSkillAffected consumes that list
// rather than re-running selection.
func TestApplyToPartyReturnsAppliedIds(t *testing.T) {
	buffedParty(t)

	applied := make([]uint32, 0)
	got := applyToParty(testLogger())(context.Background())(mkField(), testCasterId, 0b11000)(
		func(id uint32) error {
			applied = append(applied, id)
			return nil
		})

	if want := []uint32{testMemberA, testMemberB}; !eqIds(sortedCopy(got), want) {
		t.Fatalf("applyToParty returned %v, want %v", got, want)
	}
	if !eqIds(sortedCopy(applied), sortedCopy(got)) {
		t.Fatalf("applied %v but reported %v — the two must not diverge", applied, got)
	}
}

// TestAnnounceSkillAffectedSkipsCaster is the bug-2 regression: party members
// buffed by someone else must receive the SKILL_AFFECTED effect (gms_v83
// CUser::OnEffect @0x9377d9 case 2 -> CUser::ShowSkillAffected @0x93632a),
// while the caster must not — their client renders the cast locally and the
// server already sends them SKILL_USE.
func TestAnnounceSkillAffectedSkipsCaster(t *testing.T) {
	calls := captureSkillAffected(t)

	announceSkillAffected(testLogger(), context.Background(), nil, mkField(),
		testCasterId, []uint32{testCasterId, testMemberA, testMemberB}, 3221002, 30)

	if len(*calls) != 2 {
		t.Fatalf("emitted %d SKILL_AFFECTED effects, want 2 (both non-caster recipients)", len(*calls))
	}
	for _, c := range *calls {
		if c.recipientId == testCasterId {
			t.Fatalf("emitted SKILL_AFFECTED to the caster [%d]", testCasterId)
		}
		if c.skillId != 3221002 || c.skillLevel != 30 {
			t.Fatalf("emitted skill [%d] level [%d], want [3221002] level [30]", c.skillId, c.skillLevel)
		}
	}
}

// TestAnnounceSkillAffectedNoRecipientsIsSilent guards the solo-cast case: a
// caster-only buff must produce no effect packets at all.
func TestAnnounceSkillAffectedNoRecipientsIsSilent(t *testing.T) {
	calls := captureSkillAffected(t)

	announceSkillAffected(testLogger(), context.Background(), nil, mkField(),
		testCasterId, []uint32{testCasterId}, 4101004, 20)

	if len(*calls) != 0 {
		t.Fatalf("emitted %d SKILL_AFFECTED effects for a caster-only cast, want 0", len(*calls))
	}
}

func sortedCopy(in []uint32) []uint32 {
	out := make([]uint32, len(in))
	copy(out, in)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
