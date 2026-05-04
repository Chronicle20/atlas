package character

import (
	"atlas-effective-stats/external/data/equipment"
	"atlas-effective-stats/stat"
	"context"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func createTestTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func TestNewModel(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	if m.Tenant() != ten {
		t.Errorf("Tenant() = %v, want %v", m.Tenant(), ten)
	}
	if m.WorldId() != 1 {
		t.Errorf("WorldId() = %v, want 1", m.WorldId())
	}
	if m.ChannelId() != 2 {
		t.Errorf("ChannelId() = %v, want 2", m.ChannelId())
	}
	if m.CharacterId() != 12345 {
		t.Errorf("CharacterId() = %v, want 12345", m.CharacterId())
	}
	if len(m.Bonuses()) != 0 {
		t.Errorf("Bonuses() length = %v, want 0", len(m.Bonuses()))
	}
	if m.Initialized() {
		t.Error("Initialized() = true, want false")
	}
}

func TestModelWithBaseStats(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	m2 := m.WithBaseStats(base)

	// Original should be unchanged (immutable)
	if m.BaseStats().Strength() != 0 {
		t.Errorf("Original model modified: Strength = %v, want 0", m.BaseStats().Strength())
	}

	// New model should have the base stats
	if m2.BaseStats().Strength() != 50 {
		t.Errorf("New model Strength() = %v, want 50", m2.BaseStats().Strength())
	}
	if m2.BaseStats().Dexterity() != 40 {
		t.Errorf("New model Dexterity() = %v, want 40", m2.BaseStats().Dexterity())
	}
}

func TestModelWithBonus_AddNew(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	m2 := m.WithBonus(b)

	// Original should be unchanged (immutable)
	if len(m.Bonuses()) != 0 {
		t.Errorf("Original model modified: bonuses = %v, want 0", len(m.Bonuses()))
	}

	// New model should have the bonus
	if len(m2.Bonuses()) != 1 {
		t.Errorf("New model bonuses = %v, want 1", len(m2.Bonuses()))
	}
}

func TestModelWithBonus_ReplaceSameSourceAndType(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b1 := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	m = m.WithBonus(b1)

	b2 := stat.NewBonus("equipment:1", stat.TypeStrength, 30)
	m = m.WithBonus(b2)

	// Should still have only 1 bonus
	if len(m.Bonuses()) != 1 {
		t.Errorf("Bonuses count = %v, want 1", len(m.Bonuses()))
	}

	// Should have the updated value
	if m.Bonuses()[0].Amount() != 30 {
		t.Errorf("Bonus Amount() = %v, want 30", m.Bonuses()[0].Amount())
	}
}

func TestModelWithBonus_DifferentSources(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b1 := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	b2 := stat.NewBonus("buff:2311003", stat.TypeStrength, 15)
	m = m.WithBonus(b1).WithBonus(b2)

	if len(m.Bonuses()) != 2 {
		t.Errorf("Bonuses count = %v, want 2", len(m.Bonuses()))
	}
}

func TestModelWithBonus_SameSourceDifferentTypes(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b1 := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	b2 := stat.NewBonus("equipment:1", stat.TypeDexterity, 15)
	m = m.WithBonus(b1).WithBonus(b2)

	if len(m.Bonuses()) != 2 {
		t.Errorf("Bonuses count = %v, want 2", len(m.Bonuses()))
	}
}

func TestModelWithoutBonus_RemoveExisting(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	m = m.WithBonus(b)

	m2 := m.WithoutBonus("equipment:1", stat.TypeStrength)

	// Original should still have the bonus (immutable)
	if len(m.Bonuses()) != 1 {
		t.Errorf("Original model modified: bonuses = %v, want 1", len(m.Bonuses()))
	}

	// New model should not have the bonus
	if len(m2.Bonuses()) != 0 {
		t.Errorf("New model bonuses = %v, want 0", len(m2.Bonuses()))
	}
}

func TestModelWithoutBonus_RemoveNonExistent(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	m = m.WithBonus(b)

	// Try to remove a bonus that doesn't exist
	m2 := m.WithoutBonus("buff:123", stat.TypeStrength)

	// Should still have the original bonus
	if len(m2.Bonuses()) != 1 {
		t.Errorf("Bonuses count = %v, want 1", len(m2.Bonuses()))
	}
}

func TestModelWithoutBonusesBySource(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b1 := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	b2 := stat.NewBonus("equipment:1", stat.TypeDexterity, 15)
	b3 := stat.NewBonus("buff:123", stat.TypeStrength, 10)
	m = m.WithBonus(b1).WithBonus(b2).WithBonus(b3)

	m2 := m.WithoutBonusesBySource("equipment:1")

	// Should only have the buff bonus left
	if len(m2.Bonuses()) != 1 {
		t.Errorf("Bonuses count = %v, want 1", len(m2.Bonuses()))
	}

	bonuses := m2.Bonuses()
	if bonuses[0].Source() != "buff:123" {
		t.Errorf("Remaining bonus source = %v, want buff:123", bonuses[0].Source())
	}
}

func TestModelBonuses_DefensiveCopy(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	b := stat.NewBonus("equipment:1", stat.TypeStrength, 20)
	m = m.WithBonus(b)

	bonuses1 := m.Bonuses()
	bonuses2 := m.Bonuses()

	// Modify the first slice
	if len(bonuses1) > 0 {
		bonuses1[0] = stat.NewBonus("modified", stat.TypeDexterity, 99)
	}

	// Second slice should be unaffected
	if bonuses2[0].Source() == "modified" {
		t.Error("Bonuses() does not return a defensive copy")
	}
}

func TestModelComputeEffectiveStats_BaseOnly(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	m = m.WithBaseStats(base)

	computed := m.ComputeEffectiveStats(nil)

	if computed.Strength() != 50 {
		t.Errorf("Strength() = %v, want 50", computed.Strength())
	}
	if computed.Dexterity() != 40 {
		t.Errorf("Dexterity() = %v, want 40", computed.Dexterity())
	}
	if computed.Luck() != 30 {
		t.Errorf("Luck() = %v, want 30", computed.Luck())
	}
	if computed.Intelligence() != 25 {
		t.Errorf("Intelligence() = %v, want 25", computed.Intelligence())
	}
	if computed.MaxHp() != 5000 {
		t.Errorf("MaxHP() = %v, want 5000", computed.MaxHp())
	}
	if computed.MaxMp() != 3000 {
		t.Errorf("MaxMP() = %v, want 3000", computed.MaxMp())
	}
}

func TestModelComputeEffectiveStats_WithFlatBonus(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	b := stat.NewBonus("equipment:1", stat.TypeStrength, 15)
	m = m.WithBaseStats(base).WithBonus(b)

	computed := m.ComputeEffectiveStats(nil)

	// (50 + 15) * 1.0 = 65
	if computed.Strength() != 65 {
		t.Errorf("Strength() = %v, want 65", computed.Strength())
	}
}

func TestModelComputeEffectiveStats_WithMultiplierBonus(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	b := stat.NewMultiplierBonus("buff:2311003", stat.TypeStrength, 0.10) // +10%
	m = m.WithBaseStats(base).WithBonus(b)

	computed := m.ComputeEffectiveStats(nil)

	// 50 * 1.10 = 55
	if computed.Strength() != 55 {
		t.Errorf("Strength() = %v, want 55", computed.Strength())
	}
}

func TestModelComputeEffectiveStats_MixedBonuses(t *testing.T) {
	// Example from plan: Base 50 + Equipment 15, Maple Warrior 10%
	// effective_str = floor((50 + 15) * (1.0 + 0.10)) = floor(65 * 1.10) = floor(71.5) = 71
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	bEquip := stat.NewBonus("equipment:1", stat.TypeStrength, 15)
	bBuff := stat.NewMultiplierBonus("buff:2311003", stat.TypeStrength, 0.10)
	m = m.WithBaseStats(base).WithBonus(bEquip).WithBonus(bBuff)

	computed := m.ComputeEffectiveStats(nil)

	// (50 + 15) * 1.10 = 65 * 1.10 = 71.5 -> 71
	if computed.Strength() != 71 {
		t.Errorf("Strength() = %v, want 71", computed.Strength())
	}
}

func TestModelComputeEffectiveStats_MultipleMultipliers(t *testing.T) {
	// Example from plan: Base 5000 + Equipment 500 + Passive 200, Hyper Body 60% + Maple Warrior 10%
	// effective_maxhp = floor((5000 + 500 + 200) * (1.0 + 0.60 + 0.10)) = floor(5700 * 1.70) = floor(9690) = 9690
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	bEquip := stat.NewBonus("equipment:1", stat.TypeMaxHp, 500)
	bPassive := stat.NewBonus("passive:1001", stat.TypeMaxHp, 200)
	bHyperBody := stat.NewMultiplierBonus("buff:hyper", stat.TypeMaxHp, 0.60)
	bMapleWarrior := stat.NewMultiplierBonus("buff:mw", stat.TypeMaxHp, 0.10)

	m = m.WithBaseStats(base).WithBonus(bEquip).WithBonus(bPassive).WithBonus(bHyperBody).WithBonus(bMapleWarrior)

	computed := m.ComputeEffectiveStats(nil)

	// (5000 + 500 + 200) * (1.0 + 0.60 + 0.10) = 5700 * 1.70 = 9690
	if computed.MaxHp() != 9690 {
		t.Errorf("MaxHP() = %v, want 9690", computed.MaxHp())
	}
}

func TestModelComputeEffectiveStats_SecondaryStats(t *testing.T) {
	// Test stats that start at 0 (WATK, MATK, etc.)
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	bWatk := stat.NewBonus("equipment:1", stat.TypeWeaponAttack, 100)
	bWatkBuff := stat.NewBonus("buff:rage", stat.TypeWeaponAttack, 20)

	m = m.WithBaseStats(base).WithBonus(bWatk).WithBonus(bWatkBuff)

	computed := m.ComputeEffectiveStats(nil)

	// (0 + 100 + 20) * 1.0 = 120
	if computed.WeaponAttack() != 120 {
		t.Errorf("WeaponAttack() = %v, want 120", computed.WeaponAttack())
	}
}

func TestModelWithInitialized(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	if m.Initialized() {
		t.Error("New model should not be initialized")
	}

	m2 := m.WithInitialized()

	// Original unchanged
	if m.Initialized() {
		t.Error("Original model should not be modified")
	}

	// New model is initialized
	if !m2.Initialized() {
		t.Error("WithInitialized() should return initialized model")
	}
}

func TestModelRecompute(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	b := stat.NewBonus("equipment:1", stat.TypeStrength, 15)
	m = m.WithBaseStats(base).WithBonus(b)

	m2 := m.Recompute()

	if m2.Computed().Strength() != 65 {
		t.Errorf("Recompute() Strength() = %v, want 65", m2.Computed().Strength())
	}
	if m2.LastUpdated().IsZero() {
		t.Error("Recompute() should update LastUpdated()")
	}
}

func TestModelComputeEffectiveStats_MaxHpCappedAt30k(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	// Base 25000 + 10000 flat bonus would compute to 35000 — must clamp to 30000.
	base := stat.NewBase(50, 40, 30, 25, 25000, 20000)
	bHp := stat.NewBonus("equipment:1", stat.TypeMaxHp, 10000)
	bMp := stat.NewBonus("equipment:2", stat.TypeMaxMp, 15000)
	m = m.WithBaseStats(base).WithBonus(bHp).WithBonus(bMp)

	computed := m.ComputeEffectiveStats(nil)

	if computed.MaxHp() != MaxHpMpCap {
		t.Errorf("MaxHp() = %v, want %v (cap)", computed.MaxHp(), MaxHpMpCap)
	}
	if computed.MaxMp() != MaxHpMpCap {
		t.Errorf("MaxMp() = %v, want %v (cap)", computed.MaxMp(), MaxHpMpCap)
	}
}

func TestModelComputeEffectiveStats_MaxHpUnderCapNotClamped(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	// Base 5000 + 1000 = 6000 — well under cap, must pass through unchanged.
	base := stat.NewBase(50, 40, 30, 25, 5000, 3000)
	bHp := stat.NewBonus("equipment:1", stat.TypeMaxHp, 1000)
	m = m.WithBaseStats(base).WithBonus(bHp)

	computed := m.ComputeEffectiveStats(nil)

	if computed.MaxHp() != 6000 {
		t.Errorf("MaxHp() = %v, want 6000 (no clamp applied)", computed.MaxHp())
	}
	if computed.MaxMp() != 3000 {
		t.Errorf("MaxMp() = %v, want 3000 (no clamp applied)", computed.MaxMp())
	}
}

func TestModelComputeEffectiveStats_MaxHpCappedWithMultiplier(t *testing.T) {
	ten := createTestTenant()
	ch := channel.NewModel(1, 2)
	m := NewModel(ten, ch, 12345)

	// 25000 * 2.0 = 50000 → clamp to 30000.
	base := stat.NewBase(50, 40, 30, 25, 25000, 25000)
	bHp := stat.NewMultiplierBonus("buff:hyper-body", stat.TypeMaxHp, 1.0)
	bMp := stat.NewMultiplierBonus("buff:hyper-body", stat.TypeMaxMp, 1.0)
	m = m.WithBaseStats(base).WithBonus(bHp).WithBonus(bMp)

	computed := m.ComputeEffectiveStats(nil)

	if computed.MaxHp() != MaxHpMpCap {
		t.Errorf("MaxHp() = %v, want %v (cap)", computed.MaxHp(), MaxHpMpCap)
	}
	if computed.MaxMp() != MaxHpMpCap {
		t.Errorf("MaxMp() = %v, want %v (cap)", computed.MaxMp(), MaxHpMpCap)
	}
}

func TestModel_JSONRoundTrip_PreservesWearerAndEquipped(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 39, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(35, job.Id(200))).
		WithEquippedAsset(NewEquippedAsset(42, 1052095, []stat.Bonus{
			stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
		})).
		withQualifiedSnapshot(map[uint32]bool{42: true})

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Model
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Wearer().Level() != 35 || got.Wearer().JobId() != job.Id(200) {
		t.Errorf("wearer not preserved: %+v", got.Wearer())
	}
	eq := got.Equipped()
	if len(eq) != 1 {
		t.Fatalf("equipped len = %d, want 1", len(eq))
	}
	asset := eq[42]
	if asset.TemplateId() != 1052095 {
		t.Errorf("template not preserved: %d", asset.TemplateId())
	}
	if len(asset.Bonuses()) != 1 || asset.Bonuses()[0].Amount() != 50 {
		t.Errorf("snapshot bonuses not preserved: %+v", asset.Bonuses())
	}
	if !got.qualifiedSnapshot[42] {
		t.Errorf("qualifiedSnapshot not preserved: %+v", got.qualifiedSnapshot)
	}
}

func TestRecomputeWith_DropsUnqualifiedEquipmentFromComputed(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 39, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a)
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))
	if m.Computed().MaxMp() != 6330 {
		t.Errorf("MaxMp = %d, want 6330 (unqualified item dropped)", m.Computed().MaxMp())
	}
}

func TestRecomputeWith_IncludesQualifiedEquipmentInComputed(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 40, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a)
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))
	if m.Computed().MaxMp() != 6380 {
		t.Errorf("MaxMp = %d, want 6380", m.Computed().MaxMp())
	}
}

func TestBonuses_OmitsUnqualifiedEquipment(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 39, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a).
		WithBonus(stat.NewBonus("buff:7", stat.TypeStrength, 5))
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))

	got := m.Bonuses()
	if len(got) != 1 || got[0].Source() != "buff:7" {
		t.Errorf("Bonuses() = %+v, want only buff:7 entry", got)
	}
}

func TestBonuses_IncludesQualifiedEquipment(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ch := channel.NewModel(0, 0)
	a := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	m := NewModel(tn, ch, 12345).
		WithBaseStats(stat.NewBase(4, 25, 40, 4, 1430, 6330)).
		WithWearer(NewWearerProfile(30, job.Id(200))).
		WithEquippedAsset(a)
	prov := func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		return equipment.EquipmentRequirements{ReqLuk: 40}, true
	}
	m = m.RecomputeWith(prov, tenant.WithContext(context.Background(), tn))

	got := m.Bonuses()
	if len(got) != 1 || got[0].Source() != "equipment:42" || got[0].Amount() != 50 {
		t.Errorf("Bonuses() = %+v, want one equipment:42 MaxMp=50 entry", got)
	}
}
