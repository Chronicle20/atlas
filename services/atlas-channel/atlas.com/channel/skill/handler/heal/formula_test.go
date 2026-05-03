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

func TestHealXp_AllRecipientsFullHp_ReturnsZero(t *testing.T) {
	rs := []recipient{
		{Hp: 1000, MaxHp: 1000},
		{Hp: 800, MaxHp: 800},
	}
	if got := HealXp(200, rs, 5); got != 0 {
		t.Fatalf("HealXp full-hp = %d, want 0", got)
	}
}

func TestHealXp_AppliedHealAccumulates(t *testing.T) {
	// per=200, recip 1 missing 150 → applied 150
	// per=200, recip 2 missing 300 → applied 200
	// total 350, /10 = 35, * skillLevel 5 = 175
	rs := []recipient{
		{Hp: 850, MaxHp: 1000},
		{Hp: 500, MaxHp: 800},
	}
	if got := HealXp(200, rs, 5); got != 175 {
		t.Fatalf("HealXp accumulate = %d, want 175", got)
	}
}

func TestHealXp_SkillLevelZeroReturnsZero(t *testing.T) {
	rs := []recipient{{Hp: 0, MaxHp: 1000}}
	if got := HealXp(200, rs, 0); got != 0 {
		t.Fatalf("HealXp skillLevel=0 = %d, want 0", got)
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
