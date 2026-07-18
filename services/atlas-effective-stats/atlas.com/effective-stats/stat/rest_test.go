package stat

import "testing"

// TestTransformBonus_BasePercent verifies that a base-percent bonus (e.g. Maple
// Warrior, which grants floor(base*pct/100) rather than a flat amount or a
// multiplier) carries its basePercent dimension through to the REST
// projection. Before the fix, BonusRestModel had no BasePercent field, so
// this bonus was indistinguishable from a zero bonus ({amount:0,multiplier:0})
// in the GET /characters/{id}/stats bonuses breakdown.
func TestTransformBonus_BasePercent(t *testing.T) {
	b := NewBasePercentBonus("buff:2311003", TypeStrength, 10)

	m := TransformBonus(b)

	if m.Source != "buff:2311003" {
		t.Errorf("Source = %v, want buff:2311003", m.Source)
	}
	if m.StatType != string(TypeStrength) {
		t.Errorf("StatType = %v, want %v", m.StatType, TypeStrength)
	}
	if m.Amount != 0 {
		t.Errorf("Amount = %v, want 0", m.Amount)
	}
	if m.Multiplier != 0.0 {
		t.Errorf("Multiplier = %v, want 0", m.Multiplier)
	}
	if m.BasePercent != 10 {
		t.Errorf("BasePercent = %v, want 10", m.BasePercent)
	}
}

// TestTransformBonus_FlatAmount verifies a flat-amount bonus still projects
// with BasePercent==0 and does not regress Amount.
func TestTransformBonus_FlatAmount(t *testing.T) {
	b := NewBonus("equipment:1", TypeStrength, 20)

	m := TransformBonus(b)

	if m.Amount != 20 {
		t.Errorf("Amount = %v, want 20", m.Amount)
	}
	if m.Multiplier != 0.0 {
		t.Errorf("Multiplier = %v, want 0", m.Multiplier)
	}
	if m.BasePercent != 0 {
		t.Errorf("BasePercent = %v, want 0", m.BasePercent)
	}
}

// TestTransformBonus_Multiplier verifies a multiplier bonus still projects
// with BasePercent==0 and does not regress Multiplier.
func TestTransformBonus_Multiplier(t *testing.T) {
	b := NewMultiplierBonus("buff:x", TypeMaxHp, 0.60)

	m := TransformBonus(b)

	if m.Multiplier != 0.60 {
		t.Errorf("Multiplier = %v, want 0.60", m.Multiplier)
	}
	if m.Amount != 0 {
		t.Errorf("Amount = %v, want 0", m.Amount)
	}
	if m.BasePercent != 0 {
		t.Errorf("BasePercent = %v, want 0", m.BasePercent)
	}
}
