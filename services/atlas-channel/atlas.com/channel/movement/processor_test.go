package movement

import (
	"testing"

	"atlas-channel/monster/information"
)

func TestNarrowSkill_HappyPath(t *testing.T) {
	id, lvl, ok := narrowSkillBytes(100, 2)
	if !ok || id != 100 || lvl != 2 {
		t.Fatalf("got id=%d lvl=%d ok=%v; want 100 2 true", id, lvl, ok)
	}
}

func TestNarrowSkill_NegativeRejected(t *testing.T) {
	if _, _, ok := narrowSkillBytes(-1, 1); ok {
		t.Fatalf("expected reject for negative skillId")
	}
	if _, _, ok := narrowSkillBytes(1, -1); ok {
		t.Fatalf("expected reject for negative skillLevel")
	}
}

func TestNarrowSkill_OverflowRejected(t *testing.T) {
	if _, _, ok := narrowSkillBytes(256, 1); ok {
		t.Fatalf("expected reject for skillId > 255")
	}
	if _, _, ok := narrowSkillBytes(1, 256); ok {
		t.Fatalf("expected reject for skillLevel > 255")
	}
}

func TestNarrowSkill_BoundaryAccepted(t *testing.T) {
	id, lvl, ok := narrowSkillBytes(255, 255)
	if !ok || id != 255 || lvl != 255 {
		t.Fatalf("got id=%d lvl=%d ok=%v; want 255 255 true", id, lvl, ok)
	}
}

func TestComputeAckMp_BasicAttackPath_DecrementsByConMp(t *testing.T) {
	atks := []information.AttackInfo{
		{Pos: 2, ConMP: 5, AttackAfter: 1500},
	}
	got := computeAckMp(uint16(100), uint8(1), atks)
	if got != 95 {
		t.Errorf("computeAckMp(100, pos0=1, conMP=5) = %d, want 95", got)
	}
}

func TestComputeAckMp_BasicAttackPath_NoAttackInfo_Untouched(t *testing.T) {
	got := computeAckMp(uint16(100), uint8(0), nil)
	if got != 100 {
		t.Errorf("computeAckMp with no attack info = %d, want 100", got)
	}
}

func TestComputeAckMp_BasicAttackPath_ConMpExceedsMp_ClampsToZero(t *testing.T) {
	atks := []information.AttackInfo{{Pos: 1, ConMP: 50, AttackAfter: 1500}}
	got := computeAckMp(uint16(10), uint8(0), atks)
	if got != 0 {
		t.Errorf("computeAckMp clamps to zero on overflow, got %d", got)
	}
}

func TestComputeAckMp_BasicAttackPath_PosNotFound_Untouched(t *testing.T) {
	atks := []information.AttackInfo{{Pos: 1, ConMP: 5, AttackAfter: 1500}}
	got := computeAckMp(uint16(100), uint8(2), atks)
	if got != 100 {
		t.Errorf("computeAckMp with pos not found = %d, want 100", got)
	}
}
