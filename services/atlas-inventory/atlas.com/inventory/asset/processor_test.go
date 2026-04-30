package asset

import (
	"atlas-inventory/data/equipment/statistics"
	"testing"

	"github.com/google/uuid"
)

// buildTestEquipStats constructs a statistics.Model with known values using the
// exported Extract + RestModel path, which is the only way to build one without
// an exported constructor.
func buildTestEquipStats() statistics.Model {
	ea, _ := statistics.Extract(statistics.RestModel{
		Strength:      10,
		Dexterity:     8,
		Intelligence:  6,
		Luck:          4,
		Hp:            100,
		Mp:            50,
		WeaponAttack:  15,
		MagicAttack:   12,
		WeaponDefense: 20,
		MagicDefense:  18,
		Accuracy:      5,
		Avoidability:  3,
		Speed:         10,
		Jump:          5,
		Slots:         7,
	})
	return ea
}

func TestApplyEquipStats_UseAverageStats_True_WritesVerbatim(t *testing.T) {
	ea := buildTestEquipStats()
	b := NewBuilder(uuid.New(), 1040010)
	applyEquipStats(b, ea, true)
	m := b.Build()

	if m.Strength() != 10 {
		t.Errorf("expected Strength 10, got %d", m.Strength())
	}
	if m.Dexterity() != 8 {
		t.Errorf("expected Dexterity 8, got %d", m.Dexterity())
	}
	if m.Intelligence() != 6 {
		t.Errorf("expected Intelligence 6, got %d", m.Intelligence())
	}
	if m.Luck() != 4 {
		t.Errorf("expected Luck 4, got %d", m.Luck())
	}
	if m.Hp() != 100 {
		t.Errorf("expected Hp 100, got %d", m.Hp())
	}
	if m.Mp() != 50 {
		t.Errorf("expected Mp 50, got %d", m.Mp())
	}
	if m.WeaponAttack() != 15 {
		t.Errorf("expected WeaponAttack 15, got %d", m.WeaponAttack())
	}
	if m.MagicAttack() != 12 {
		t.Errorf("expected MagicAttack 12, got %d", m.MagicAttack())
	}
	if m.WeaponDefense() != 20 {
		t.Errorf("expected WeaponDefense 20, got %d", m.WeaponDefense())
	}
	if m.MagicDefense() != 18 {
		t.Errorf("expected MagicDefense 18, got %d", m.MagicDefense())
	}
	if m.Accuracy() != 5 {
		t.Errorf("expected Accuracy 5, got %d", m.Accuracy())
	}
	if m.Avoidability() != 3 {
		t.Errorf("expected Avoidability 3, got %d", m.Avoidability())
	}
	if m.Speed() != 10 {
		t.Errorf("expected Speed 10, got %d", m.Speed())
	}
	if m.Jump() != 5 {
		t.Errorf("expected Jump 5, got %d", m.Jump())
	}
	if m.Slots() != 7 {
		t.Errorf("expected Slots 7, got %d", m.Slots())
	}
}

func TestApplyEquipStats_UseAverageStats_False_RetainsVariance(t *testing.T) {
	ea := buildTestEquipStats()

	// Run 20 iterations; at least one stat should differ from the base across all trials.
	totalDelta := 0
	const trials = 20
	for i := 0; i < trials; i++ {
		b := NewBuilder(uuid.New(), 1040010)
		applyEquipStats(b, ea, false)
		m := b.Build()

		// Sum absolute deltas across stats that have non-zero base values.
		// Slots is always deterministic, so exclude it from delta check.
		totalDelta += abs16(m.Strength(), ea.Strength())
		totalDelta += abs16(m.Dexterity(), ea.Dexterity())
		totalDelta += abs16(m.Intelligence(), ea.Intelligence())
		totalDelta += abs16(m.Luck(), ea.Luck())
		totalDelta += abs16(m.Hp(), ea.Hp())
		totalDelta += abs16(m.Mp(), ea.Mp())
		totalDelta += abs16(m.WeaponAttack(), ea.WeaponAttack())
		totalDelta += abs16(m.MagicAttack(), ea.MagicAttack())
		totalDelta += abs16(m.WeaponDefense(), ea.WeaponDefense())
		totalDelta += abs16(m.MagicDefense(), ea.MagicDefense())
	}

	if totalDelta == 0 {
		t.Error("expected at least some stat variance over 20 trials, but all rolls equalled the base values")
	}

	// Slots must always equal the base regardless of variance mode.
	b := NewBuilder(uuid.New(), 1040010)
	applyEquipStats(b, ea, false)
	if b.Build().Slots() != ea.Slots() {
		t.Errorf("expected Slots %d (deterministic), got %d", ea.Slots(), b.Build().Slots())
	}
}

func abs16(a, b uint16) int {
	if a >= b {
		return int(a - b)
	}
	return int(b - a)
}
