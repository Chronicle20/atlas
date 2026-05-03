package monster

import (
	"testing"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

// Stat-flag mob debuffs (SEAL/DARKNESS/WEAKEN/STUN/CURSE/SEDUCE/CONFUSE/UNDEAD/FEAR)
// have no `x` in the WZ — Cosmic's giveDebuff writes a literal 1 into the wire
// nValue field. The v83 client treats nValue==0 as "stat not applied" and
// suppresses the debuff icon plus flag-gated effects (e.g. SEAL skill block,
// WEAKEN jump block).
func TestDebuffWireValue_StatFlagDiseases_CoerceZeroToOne(t *testing.T) {
	cases := []struct {
		name    string
		skillId uint16
	}{
		{"SEAL", monster2.SkillTypeSeal},
		{"DARKNESS", monster2.SkillTypeDarkness},
		{"WEAKNESS", monster2.SkillTypeWeakness},
		{"STUN", monster2.SkillTypeStun},
		{"CURSE", monster2.SkillTypeCurse},
		{"SEDUCE", monster2.SkillTypeSeduce},
		{"REVERSE_INPUT", monster2.SkillTypeReverseInput},
		{"FEAR", monster2.SkillTypeFear},
		{"UNDEAD", monster2.SkillTypeUndead},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := debuffWireValue(tc.skillId, 0); got != 1 {
				t.Errorf("debuffWireValue(%d, 0) = %d, want 1", tc.skillId, got)
			}
		})
	}
}

// Magnitude-bearing mob debuffs (POISON dot damage, SLOW speed delta,
// STOP_POTION/STOP_MOTION counts) carry their wire nValue in `x`. Pass it
// through untouched, including zero — that's a real value, not a missing one.
func TestDebuffWireValue_MagnitudeDiseases_PassXThrough(t *testing.T) {
	cases := []struct {
		name    string
		skillId uint16
		x       int32
		want    int32
	}{
		{"SLOW level 1 (x=80)", monster2.SkillTypeSlow, 80, 80},
		{"POISON x=200", monster2.SkillTypePoison, 200, 200},
		{"STOP_POTION x=1", monster2.SkillTypeStopPotion, 1, 1},
		{"STOP_MOTION x=1", monster2.SkillTypeStopMotion, 1, 1},
		{"SLOW with zero x stays zero", monster2.SkillTypeSlow, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := debuffWireValue(tc.skillId, tc.x); got != tc.want {
				t.Errorf("debuffWireValue(%d, %d) = %d, want %d", tc.skillId, tc.x, got, tc.want)
			}
		})
	}
}

// If the WZ ever ships a non-zero x for a stat-flag disease, trust it — that's
// a real magnitude (e.g. SEAL with x=1 in some MapleStory revisions). Only the
// missing-from-WZ case (x=0) gets the literal 1 fallback.
func TestDebuffWireValue_StatFlagDisease_NonZeroXPassesThrough(t *testing.T) {
	if got := debuffWireValue(monster2.SkillTypeSeal, 5); got != 5 {
		t.Errorf("debuffWireValue(SEAL, 5) = %d, want 5", got)
	}
}
