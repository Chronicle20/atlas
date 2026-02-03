package character

import (
	"atlas-effective-stats/stat"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func createTestContext() (logrus.FieldLogger, context.Context, tenant.Model) {
	l, _ := test.NewNullLogger()
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), t)
	return l, ctx, t
}

func setupProcessorTest(t *testing.T) (Processor, logrus.FieldLogger, context.Context, tenant.Model) {
	t.Helper()
	GetRegistry().ResetForTesting()
	l, ctx, ten := createTestContext()
	p := NewProcessor(l, ctx)
	return p, l, ctx, ten
}

func TestProcessor_AddBonus(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	err := p.AddBonus(1, 2, 12345, "test:1", stat.TypeStrength, 20)
	if err != nil {
		t.Fatalf("AddBonus() error = %v", err)
	}

	m, err := GetRegistry().Get(ten, 12345)
	if err != nil {
		t.Fatalf("Registry.Get() error = %v", err)
	}

	bonuses := m.Bonuses()
	if len(bonuses) != 1 {
		t.Fatalf("Bonuses count = %v, want 1", len(bonuses))
	}

	if bonuses[0].Source() != "test:1" {
		t.Errorf("Bonus source = %v, want test:1", bonuses[0].Source())
	}
	if bonuses[0].StatType() != stat.TypeStrength {
		t.Errorf("Bonus statType = %v, want %v", bonuses[0].StatType(), stat.TypeStrength)
	}
	if bonuses[0].Amount() != 20 {
		t.Errorf("Bonus amount = %v, want 20", bonuses[0].Amount())
	}

	// Check computed stats
	if m.Computed().Strength() != 20 {
		t.Errorf("Computed Strength = %v, want 20", m.Computed().Strength())
	}
}

func TestProcessor_AddMultiplierBonus(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	// First set base stats
	base := stat.NewBase(100, 50, 50, 50, 5000, 3000)
	_ = p.SetBaseStats(1, 2, 12345, base)

	// Add multiplier bonus
	err := p.AddMultiplierBonus(1, 2, 12345, "buff:123", stat.TypeStrength, 0.10)
	if err != nil {
		t.Fatalf("AddMultiplierBonus() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)

	// 100 * 1.10 = 110
	if m.Computed().Strength() != 110 {
		t.Errorf("Computed Strength = %v, want 110", m.Computed().Strength())
	}
}

func TestProcessor_RemoveBonus(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	// Add bonus first
	_ = p.AddBonus(1, 2, 12345, "test:1", stat.TypeStrength, 20)

	// Remove it
	err := p.RemoveBonus(12345, "test:1", stat.TypeStrength)
	if err != nil {
		t.Fatalf("RemoveBonus() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)
	if len(m.Bonuses()) != 0 {
		t.Errorf("Bonuses count = %v, want 0", len(m.Bonuses()))
	}
}

func TestProcessor_RemoveBonus_NotFound(t *testing.T) {
	p, _, _, _ := setupProcessorTest(t)

	// Try to remove bonus for non-existent character
	err := p.RemoveBonus(99999, "test:1", stat.TypeStrength)
	if err == nil {
		t.Error("RemoveBonus() expected error for non-existent character")
	}
}

func TestProcessor_RemoveBonusesBySource(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	// Add multiple bonuses from same source
	_ = p.AddBonus(1, 2, 12345, "equipment:100", stat.TypeStrength, 10)
	_ = p.AddBonus(1, 2, 12345, "equipment:100", stat.TypeDexterity, 5)
	_ = p.AddBonus(1, 2, 12345, "buff:200", stat.TypeStrength, 15)

	// Remove all equipment bonuses
	err := p.RemoveBonusesBySource(12345, "equipment:100")
	if err != nil {
		t.Fatalf("RemoveBonusesBySource() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)
	bonuses := m.Bonuses()

	// Should only have buff bonus left
	if len(bonuses) != 1 {
		t.Fatalf("Bonuses count = %v, want 1", len(bonuses))
	}
	if bonuses[0].Source() != "buff:200" {
		t.Errorf("Remaining bonus source = %v, want buff:200", bonuses[0].Source())
	}
}

func TestProcessor_SetBaseStats(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	base := stat.NewBase(100, 80, 60, 40, 5000, 3000)
	err := p.SetBaseStats(1, 2, 12345, base)
	if err != nil {
		t.Fatalf("SetBaseStats() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)

	if m.BaseStats().Strength() != 100 {
		t.Errorf("BaseStats Strength = %v, want 100", m.BaseStats().Strength())
	}
	if m.Computed().Strength() != 100 {
		t.Errorf("Computed Strength = %v, want 100", m.Computed().Strength())
	}
}

func TestProcessor_AddEquipmentBonuses(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	bonuses := []stat.Bonus{
		stat.NewBonus("", stat.TypeStrength, 15),
		stat.NewBonus("", stat.TypeDexterity, 10),
	}

	err := p.AddEquipmentBonuses(1, 2, 12345, 999, bonuses)
	if err != nil {
		t.Fatalf("AddEquipmentBonuses() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)
	allBonuses := m.Bonuses()

	if len(allBonuses) != 2 {
		t.Fatalf("Bonuses count = %v, want 2", len(allBonuses))
	}

	// Check that bonuses have the equipment source
	for _, b := range allBonuses {
		if b.Source() != "equipment:999" {
			t.Errorf("Bonus source = %v, want equipment:999", b.Source())
		}
	}

	// Check computed stats
	if m.Computed().Strength() != 15 {
		t.Errorf("Computed Strength = %v, want 15", m.Computed().Strength())
	}
	if m.Computed().Dexterity() != 10 {
		t.Errorf("Computed Dexterity = %v, want 10", m.Computed().Dexterity())
	}
}

func TestProcessor_RemoveEquipmentBonuses(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	bonuses := []stat.Bonus{
		stat.NewBonus("", stat.TypeStrength, 15),
	}
	_ = p.AddEquipmentBonuses(1, 2, 12345, 999, bonuses)

	err := p.RemoveEquipmentBonuses(12345, 999)
	if err != nil {
		t.Fatalf("RemoveEquipmentBonuses() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)
	if len(m.Bonuses()) != 0 {
		t.Errorf("Bonuses count = %v, want 0", len(m.Bonuses()))
	}
}

func TestProcessor_AddBuffBonuses(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	// Set base stats first
	base := stat.NewBase(100, 50, 50, 50, 5000, 3000)
	_ = p.SetBaseStats(1, 2, 12345, base)

	bonuses := []stat.Bonus{
		stat.NewMultiplierBonus("", stat.TypeMaxHP, 0.60), // Hyper Body
	}

	err := p.AddBuffBonuses(1, 2, 12345, 2311003, bonuses)
	if err != nil {
		t.Fatalf("AddBuffBonuses() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)

	// 5000 * 1.60 = 8000
	if m.Computed().MaxHP() != 8000 {
		t.Errorf("Computed MaxHP = %v, want 8000", m.Computed().MaxHP())
	}
}

func TestProcessor_RemoveBuffBonuses(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	base := stat.NewBase(100, 50, 50, 50, 5000, 3000)
	_ = p.SetBaseStats(1, 2, 12345, base)

	bonuses := []stat.Bonus{
		stat.NewMultiplierBonus("", stat.TypeMaxHP, 0.60),
	}
	_ = p.AddBuffBonuses(1, 2, 12345, 2311003, bonuses)

	err := p.RemoveBuffBonuses(12345, 2311003)
	if err != nil {
		t.Fatalf("RemoveBuffBonuses() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)

	// Back to base stats
	if m.Computed().MaxHP() != 5000 {
		t.Errorf("Computed MaxHP = %v, want 5000", m.Computed().MaxHP())
	}
}

func TestProcessor_AddPassiveBonuses(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	bonuses := []stat.Bonus{
		stat.NewBonus("", stat.TypeWeaponAttack, 20),
	}

	err := p.AddPassiveBonuses(1, 2, 12345, 1000001, bonuses)
	if err != nil {
		t.Fatalf("AddPassiveBonuses() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)

	// Check source
	allBonuses := m.Bonuses()
	if len(allBonuses) != 1 {
		t.Fatalf("Bonuses count = %v, want 1", len(allBonuses))
	}
	if allBonuses[0].Source() != "passive:1000001" {
		t.Errorf("Bonus source = %v, want passive:1000001", allBonuses[0].Source())
	}

	if m.Computed().WeaponAttack() != 20 {
		t.Errorf("Computed WeaponAttack = %v, want 20", m.Computed().WeaponAttack())
	}
}

func TestProcessor_RemovePassiveBonuses(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	bonuses := []stat.Bonus{
		stat.NewBonus("", stat.TypeWeaponAttack, 20),
	}
	_ = p.AddPassiveBonuses(1, 2, 12345, 1000001, bonuses)

	err := p.RemovePassiveBonuses(12345, 1000001)
	if err != nil {
		t.Fatalf("RemovePassiveBonuses() error = %v", err)
	}

	m, _ := GetRegistry().Get(ten, 12345)
	if len(m.Bonuses()) != 0 {
		t.Errorf("Bonuses count = %v, want 0", len(m.Bonuses()))
	}
}

func TestProcessor_RemoveCharacter(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	// Add some data for the character
	_ = p.AddBonus(1, 2, 12345, "test:1", stat.TypeStrength, 20)

	// Remove the character
	p.RemoveCharacter(12345)

	// Character should no longer exist
	_, err := GetRegistry().Get(ten, 12345)
	if err != ErrNotFound {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestProcessor_GetEffectiveStats_PreInitialized(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	// Pre-populate the registry (simulating initialized character)
	base := stat.NewBase(100, 80, 60, 40, 5000, 3000)
	m := NewModel(ten, 1, 2, 12345).WithBaseStats(base).Recompute().WithInitialized()
	GetRegistry().Update(ten, m)

	// Now GetEffectiveStats should return without calling external services
	computed, bonuses, err := p.GetEffectiveStats(1, 2, 12345)
	if err != nil {
		t.Fatalf("GetEffectiveStats() error = %v", err)
	}

	if computed.Strength() != 100 {
		t.Errorf("Computed Strength = %v, want 100", computed.Strength())
	}
	if computed.Dexterity() != 80 {
		t.Errorf("Computed Dexterity = %v, want 80", computed.Dexterity())
	}
	if len(bonuses) != 0 {
		t.Errorf("Bonuses count = %v, want 0", len(bonuses))
	}
}

func TestProcessor_GetEffectiveStats_WithBonuses(t *testing.T) {
	p, _, _, ten := setupProcessorTest(t)

	// Pre-populate with base stats and bonuses
	base := stat.NewBase(100, 80, 60, 40, 5000, 3000)
	b := stat.NewBonus("equipment:1", stat.TypeStrength, 15)
	m := NewModel(ten, 1, 2, 12345).WithBaseStats(base).WithBonus(b).Recompute().WithInitialized()
	GetRegistry().Update(ten, m)

	computed, bonuses, err := p.GetEffectiveStats(1, 2, 12345)
	if err != nil {
		t.Fatalf("GetEffectiveStats() error = %v", err)
	}

	// 100 + 15 = 115
	if computed.Strength() != 115 {
		t.Errorf("Computed Strength = %v, want 115", computed.Strength())
	}
	if len(bonuses) != 1 {
		t.Fatalf("Bonuses count = %v, want 1", len(bonuses))
	}
	if bonuses[0].Source() != "equipment:1" {
		t.Errorf("Bonus source = %v, want equipment:1", bonuses[0].Source())
	}
}

func TestProcessor_MixedBonuses(t *testing.T) {
	// Test the complete effective stats formula with mixed flat and multiplier bonuses
	p, _, _, ten := setupProcessorTest(t)

	// Set base stats
	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	_ = p.SetBaseStats(1, 2, 12345, base)

	// Add flat equipment bonus
	equipBonuses := []stat.Bonus{
		stat.NewBonus("", stat.TypeStrength, 15),
		stat.NewBonus("", stat.TypeMaxHP, 500),
	}
	_ = p.AddEquipmentBonuses(1, 2, 12345, 100, equipBonuses)

	// Add multiplier buff (10% strength, 60% HP)
	buffBonuses := []stat.Bonus{
		stat.NewMultiplierBonus("", stat.TypeStrength, 0.10),
		stat.NewMultiplierBonus("", stat.TypeMaxHP, 0.60),
	}
	_ = p.AddBuffBonuses(1, 2, 12345, 2311003, buffBonuses)

	m, _ := GetRegistry().Get(ten, 12345)

	// Strength: (50 + 15) * 1.10 = 65 * 1.10 = 71.5 -> 71
	if m.Computed().Strength() != 71 {
		t.Errorf("Computed Strength = %v, want 71", m.Computed().Strength())
	}

	// MaxHP: (5000 + 500) * 1.60 = 5500 * 1.60 = 8800
	if m.Computed().MaxHP() != 8800 {
		t.Errorf("Computed MaxHP = %v, want 8800", m.Computed().MaxHP())
	}

	// Dexterity: 40 (no bonuses)
	if m.Computed().Dexterity() != 40 {
		t.Errorf("Computed Dexterity = %v, want 40", m.Computed().Dexterity())
	}
}
