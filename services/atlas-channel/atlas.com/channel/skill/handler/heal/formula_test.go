package heal

import "testing"

func TestHealAmount_BaseFormula_NoVariance(t *testing.T) {
	// skillHpPct=200, MA=100, INT=50, partyTargets=1, variance=1.0
	// base = 200 * (100*1.5 + 50*0.8) / 100 = 200 * 190 / 100 = 380
	// perTarget = floor(380 * 1.0 / 1) = 380
	got := HealAmount(200, 100, 50, 1, 1.0)
	if got != 380 {
		t.Fatalf("HealAmount(200,100,50,1,1.0) = %d, want 380", got)
	}
}

func TestHealAmount_PartySplit_FloorDivision(t *testing.T) {
	// base 380, partyTargets=3, variance=1.0 → floor(380/3) = 126
	got := HealAmount(200, 100, 50, 3, 1.0)
	if got != 126 {
		t.Fatalf("HealAmount split-by-3 = %d, want 126", got)
	}
}

func TestHealAmount_VarianceLow(t *testing.T) {
	// variance 0.9 → floor(380 * 0.9) = 342
	got := HealAmount(200, 100, 50, 1, 0.9)
	if got != 342 {
		t.Fatalf("HealAmount low-variance = %d, want 342", got)
	}
}

func TestHealAmount_VarianceHigh(t *testing.T) {
	// variance 1.1 → floor(380 * 1.1) = 418
	got := HealAmount(200, 100, 50, 1, 1.1)
	if got != 418 {
		t.Fatalf("HealAmount high-variance = %d, want 418", got)
	}
}

func TestHealAmount_PartyTargetsClampToOne(t *testing.T) {
	got := HealAmount(200, 100, 50, 0, 1.0)
	if got != 380 {
		t.Fatalf("HealAmount partyTargets=0 = %d, want 380 (clamped to 1)", got)
	}
}

func TestHealAmount_NegativeInputsClampToZero(t *testing.T) {
	got := HealAmount(0, 0, 0, 1, 1.0)
	if got != 0 {
		t.Fatalf("HealAmount zero inputs = %d, want 0", got)
	}
}

func TestHealAmount_OverInt16ClampsToMax(t *testing.T) {
	// Pathological: skillHpPct=1000, MA=10000, INT=10000, variance=1.1
	got := HealAmount(1000, 10000, 10000, 1, 1.1)
	if got != 32767 {
		t.Fatalf("HealAmount over-int16 = %d, want 32767", got)
	}
}

func TestHealXp_CasterExcludedFromSum(t *testing.T) {
	// Only the caster in the recipient list: no non-caster contribution.
	rs := []recipient{
		{Hp: 500, MaxHp: 1000, Level: 70, IsCaster: true},
	}
	if got := HealXp(2880, rs, 70); got != 0 {
		t.Fatalf("HealXp caster-only = %d, want 0", got)
	}
}

func TestHealXp_FullHpMemberYieldsZero(t *testing.T) {
	rs := []recipient{
		{Hp: 1000, MaxHp: 1000, Level: 70},
	}
	if got := HealXp(2880, rs, 70); got != 0 {
		t.Fatalf("HealXp full-hp member = %d, want 0", got)
	}
}

func TestHealXp_OverhealClampedToMissingHp(t *testing.T) {
	// perTarget 2880 far exceeds the 100 missing HP; only 100 counts.
	// 100 * 70 / 70 / 15 = 6
	rs := []recipient{
		{Hp: 900, MaxHp: 1000, Level: 70},
	}
	if got := HealXp(2880, rs, 70); got != 6 {
		t.Fatalf("HealXp overheal clamp = %d, want 6", got)
	}
}

func TestHealXp_LoggedCastMatchesExpectedFormula(t *testing.T) {
	// From the bug report: perTarget=2880, one party member, both level 70.
	// 2880 * 70 / 70 / 15 = 192.
	rs := []recipient{
		{Hp: 0, MaxHp: 5000, Level: 70},
	}
	if got := HealXp(2880, rs, 70); got != 192 {
		t.Fatalf("HealXp logged cast = %d, want 192", got)
	}
}

func TestHealXp_LowerLevelTargetYieldsMoreThanHigherLevel(t *testing.T) {
	low := []recipient{{Hp: 0, MaxHp: 1000, Level: 10}}
	high := []recipient{{Hp: 0, MaxHp: 1000, Level: 100}}
	gotLow := HealXp(500, low, 70)
	gotHigh := HealXp(500, high, 70)
	if gotLow <= gotHigh {
		t.Fatalf("HealXp lower-level target = %d, want > higher-level target = %d", gotLow, gotHigh)
	}
}

func TestHealXp_TwoRecipientsSummedIndependently(t *testing.T) {
	// recip 1: applied 150, level 70 -> 150*70/70/15 = 10
	// recip 2: applied 200, level 35 -> 200*70/35/15 = 26 (floor)
	rs := []recipient{
		{Hp: 850, MaxHp: 1000, Level: 70},
		{Hp: 500, MaxHp: 800, Level: 35},
	}
	if got := HealXp(200, rs, 70); got != 36 {
		t.Fatalf("HealXp two recipients summed = %d, want 36", got)
	}
}

func TestHealXp_LevelZeroRecipientSkippedNoPanic(t *testing.T) {
	rs := []recipient{
		{Hp: 0, MaxHp: 1000, Level: 0},
		{Hp: 0, MaxHp: 1000, Level: 70},
	}
	if got := HealXp(200, rs, 70); got != 13 {
		t.Fatalf("HealXp level-0 recipient skipped = %d, want 13", got)
	}
}

func TestAppliedPerRecipient_ClampsToMissing(t *testing.T) {
	got := appliedPerRecipient(380, recipient{Hp: 850, MaxHp: 1000})
	if got != 150 {
		t.Fatalf("appliedPerRecipient(380, missing=150) = %d, want 150", got)
	}
}

func TestAppliedPerRecipient_FullHpReturnsZero(t *testing.T) {
	got := appliedPerRecipient(380, recipient{Hp: 1000, MaxHp: 1000})
	if got != 0 {
		t.Fatalf("appliedPerRecipient(380, full hp) = %d, want 0", got)
	}
}

func TestAppliedPerRecipient_PerTargetSmallerThanMissing(t *testing.T) {
	got := appliedPerRecipient(100, recipient{Hp: 500, MaxHp: 1000})
	if got != 100 {
		t.Fatalf("appliedPerRecipient(100, missing=500) = %d, want 100", got)
	}
}

func TestAppliedPerRecipient_NegativePerTargetReturnsZero(t *testing.T) {
	got := appliedPerRecipient(-10, recipient{Hp: 500, MaxHp: 1000})
	if got != 0 {
		t.Fatalf("appliedPerRecipient(-10, ...) = %d, want 0", got)
	}
}

func TestAppliedPerRecipient_HpAboveMaxReturnsZero(t *testing.T) {
	// Defensive: Hp > MaxHp (stale snapshot, e.g. MaxHp dropped) yields
	// missing<0 → applied=0.
	got := appliedPerRecipient(380, recipient{Hp: 2000, MaxHp: 1000})
	if got != 0 {
		t.Fatalf("appliedPerRecipient(380, hp>max) = %d, want 0", got)
	}
}

func TestHealDelta(t *testing.T) {
	tests := []struct {
		name      string
		perTarget int16
		hp, maxHp uint16
		zombified bool
		want      int16
	}{
		{"not zombified full headroom", 80, 900, 1000, false, 80},
		{"not zombified headroom clamp", 80, 950, 1000, false, 50},
		{"not zombified at max hp", 80, 1000, 1000, false, 0},
		{"zombified full magnitude", 80, 900, 1000, true, -80},
		{"zombified clamped to current hp", 80, 50, 1000, true, -50},
		{"zombified exact kill", 80, 80, 1000, true, -80},
		{"zombified recipient already dead", 80, 0, 1000, true, 0},
		{"zombified zero magnitude", 0, 500, 1000, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := recipient{Id: 1, Hp: tt.hp, MaxHp: tt.maxHp}
			got := healDelta(tt.perTarget, r, tt.zombified)
			if got != tt.want {
				t.Fatalf("healDelta(%d, %+v, %v) = %d, want %d", tt.perTarget, r, tt.zombified, got, tt.want)
			}
			if !tt.zombified {
				if want := appliedPerRecipient(tt.perTarget, r); got != want {
					t.Fatalf("healDelta(%d, %+v, false) = %d, want equal to appliedPerRecipient = %d", tt.perTarget, r, got, want)
				}
			}
		})
	}
}
