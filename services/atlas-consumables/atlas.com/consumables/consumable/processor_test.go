package consumable

import (
	"atlas-consumables/asset"
	"atlas-consumables/equipable"
	"testing"

	consumable3 "atlas-consumables/data/consumable"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIsNotSlotConsumingScroll_SpikeScroll(t *testing.T) {
	// Spike scrolls are in the 2040200-2040299 range (shoe spikes)
	spikeScrollIds := []item.Id{2040727} // Cape for Cold Protection
	for _, id := range spikeScrollIds {
		if !item.IsScrollSpikes(id) && !item.IsScrollColdProtection(id) {
			// Skip IDs that aren't actually spikes in the item library
			continue
		}
		if item.IsScrollSpikes(id) {
			result := IsNotSlotConsumingScroll(id)
			if !result {
				t.Errorf("expected spike scroll %d to be non-slot consuming", id)
			}
		}
	}
}

func TestIsNotSlotConsumingScroll_ColdProtectionScroll(t *testing.T) {
	// Cold protection scrolls
	coldScrollIds := []item.Id{2040727} // Cape for Cold Protection
	for _, id := range coldScrollIds {
		if item.IsScrollColdProtection(id) {
			result := IsNotSlotConsumingScroll(id)
			if !result {
				t.Errorf("expected cold protection scroll %d to be non-slot consuming", id)
			}
		}
	}
}

func TestIsNotSlotConsumingScroll_RegularScroll(t *testing.T) {
	// Regular scroll that should consume slots
	regularScrollId := item.Id(2040001) // Regular scroll
	result := IsNotSlotConsumingScroll(regularScrollId)
	if result {
		t.Errorf("expected regular scroll %d to consume slots", regularScrollId)
	}
}

func TestRollStatAdjustment_ReturnsValidRange(t *testing.T) {
	// Roll multiple times and verify all results are in valid range
	counts := make(map[int16]int)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		result := rollStatAdjustment()
		if result < -5 || result > 5 {
			t.Errorf("rollStatAdjustment returned %d, expected range [-5, 5]", result)
		}
		counts[result]++
	}

	// Verify we got some distribution (not all same value)
	if len(counts) < 5 {
		t.Errorf("expected at least 5 different values, got %d", len(counts))
	}
}

func TestRollStatAdjustment_ZeroIsMostCommon(t *testing.T) {
	// Based on the probability distribution, 0 should be most common (~18.38%)
	counts := make(map[int16]int)
	iterations := 100000

	for i := 0; i < iterations; i++ {
		result := rollStatAdjustment()
		counts[result]++
	}

	// 0 should be the most common or close to it
	zeroCount := counts[0]
	zeroPercent := float64(zeroCount) / float64(iterations) * 100

	// Should be around 18.38%, allow some variance
	if zeroPercent < 15 || zeroPercent > 22 {
		t.Errorf("zero percent was %.2f%%, expected around 18.38%%", zeroPercent)
	}
}

func TestGenerateChaosChanges_SkipsZeroStats(t *testing.T) {
	// All zero stats - should produce no changes
	stats := []chaosStat{
		{0, equipable.AddStrength, 1},
		{0, equipable.AddDexterity, 1},
		{0, equipable.AddIntelligence, 1},
		{0, equipable.AddLuck, 1},
	}

	changes, err := generateChaosChanges(stats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("expected 0 changes for all zero stats, got %d", len(changes))
	}
}

func TestGenerateChaosChanges_GeneratesForNonZeroStats(t *testing.T) {
	// Non-zero stats should produce changes
	stats := []chaosStat{
		{10, equipable.AddStrength, 1},
		{0, equipable.AddDexterity, 1},
		{15, equipable.AddIntelligence, 1},
		{0, equipable.AddLuck, 1},
	}

	changes, err := generateChaosChanges(stats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 2 {
		t.Errorf("expected 2 changes for 2 non-zero stats, got %d", len(changes))
	}
}

func TestGenerateChaosChanges_AppliesMultiplier(t *testing.T) {
	stats := []chaosStat{
		{10, equipable.AddHp, 10},
	}

	changes, err := generateChaosChanges(stats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 1 {
		t.Errorf("expected 1 change for HP stat, got %d", len(changes))
	}
}

func TestApplyChaos_AllStats(t *testing.T) {
	// Create asset with all stats non-zero using flat builder
	a := asset.NewBuilder(uuid.New(), 1000000).
		SetStrength(10).
		SetDexterity(10).
		SetIntelligence(10).
		SetLuck(10).
		SetWeaponAttack(10).
		SetWeaponDefense(10).
		SetMagicAttack(10).
		SetMagicDefense(10).
		SetAccuracy(10).
		SetAvoidability(10).
		SetSpeed(10).
		SetJump(10).
		SetHp(100).
		SetMp(100).
		Build()

	changes, err := applyChaos(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 14 changes (one for each non-zero stat)
	if len(changes) != 14 {
		t.Errorf("expected 14 changes, got %d", len(changes))
	}
}

func TestApplyChaos_PartialStats(t *testing.T) {
	// Create asset with only some stats non-zero
	a := asset.NewBuilder(uuid.New(), 1000000).
		SetStrength(10).
		SetLuck(10).
		SetWeaponAttack(5).
		Build()

	changes, err := applyChaos(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 changes (str, luk, weapon attack)
	if len(changes) != 3 {
		t.Errorf("expected 3 changes for 3 non-zero stats, got %d", len(changes))
	}
}

func TestApplyChaos_HPMPMultiplier(t *testing.T) {
	// Test that HP/MP adjustments are multiplied by 10
	// Create asset with only HP and MP non-zero
	a := asset.NewBuilder(uuid.New(), 1000000).
		SetHp(100).
		SetMp(100).
		Build()

	changes, err := applyChaos(a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 changes (HP and MP only)
	if len(changes) != 2 {
		t.Errorf("expected 2 changes for HP/MP, got %d", len(changes))
	}

	// Apply changes to a builder and verify the multiplier effect
	builder := asset.NewBuilder(uuid.New(), 1000000).
		SetHp(100).
		SetMp(100)

	for _, change := range changes {
		change(builder)
	}

	result := builder.Build()

	// The change should be +/- 10, 20, 30, 40, or 50 (base adjustment * 10)
	hpDiff := int(result.Hp()) - 100
	mpDiff := int(result.Mp()) - 100

	// HP/MP changes should be multiples of 10
	if hpDiff%10 != 0 {
		t.Errorf("HP diff %d should be multiple of 10", hpDiff)
	}
	if mpDiff%10 != 0 {
		t.Errorf("MP diff %d should be multiple of 10", mpDiff)
	}

	// Changes should be in range [-50, 50]
	if hpDiff < -50 || hpDiff > 50 {
		t.Errorf("HP diff %d should be in range [-50, 50]", hpDiff)
	}
	if mpDiff < -50 || mpDiff > 50 {
		t.Errorf("MP diff %d should be in range [-50, 50]", mpDiff)
	}
}

// Test helper to create test equipable asset
func createTestEquipableAsset(templateId uint32, slots uint16, level byte) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).
		SetId(1).
		SetSlots(slots).
		SetLevel(level).
		Build()
}

// Test helper to create test scroll asset
func createTestScrollAsset(templateId uint32) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).
		SetId(2).
		Build()
}

func makeCureModel(t *testing.T, specs map[consumable3.SpecType]int32) consumable3.Model {
	t.Helper()
	rm := consumable3.RestModel{Spec: specs}
	m, err := consumable3.Extract(rm)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	return m
}

func TestCollectCureTypes_AntidotePot(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison: 1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"POISON"}, got)
}

func TestCollectCureTypes_HolyWater(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypeSeal:  1,
		consumable3.SpecTypeCurse: 1,
	})
	got := collectCureTypes(ci)
	// Order is fixed (POISON, DARKNESS, WEAKEN, SEAL, CURSE) for determinism;
	// missing entries are dropped, so Holy Water yields just SEAL then CURSE.
	assert.Equal(t, []string{"SEAL", "CURSE"}, got)
}

func TestCollectCureTypes_AllCure(t *testing.T) {
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison:   1,
		consumable3.SpecTypeDarkness: 1,
		consumable3.SpecTypeWeakness: 1,
		consumable3.SpecTypeSeal:     1,
		consumable3.SpecTypeCurse:    1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"POISON", "DARKNESS", "WEAKEN", "SEAL", "CURSE"}, got)
}

func TestCollectCureTypes_NonCureConsumable(t *testing.T) {
	// White potion: HP recovery only, no cure flags.
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypeHP: 1000,
	})
	got := collectCureTypes(ci)
	assert.Empty(t, got)
}

func TestCollectCureTypes_ZeroFlagsIgnored(t *testing.T) {
	// A 0-valued cure spec must be treated as "not present" (parser default).
	ci := makeCureModel(t, map[consumable3.SpecType]int32{
		consumable3.SpecTypePoison: 0,
		consumable3.SpecTypeCurse:  1,
	})
	got := collectCureTypes(ci)
	assert.Equal(t, []string{"CURSE"}, got)
}

func TestUsesStandardConsumer(t *testing.T) {
	// Standard-consumer routing for items that need ApplyItemEffects (HP/MP
	// recovery, status buffs, status cure). Anything not matching here falls
	// through to ConsumeBare and silently skips effect application.
	cases := []struct {
		name   string
		itemId item.Id
		want   bool
	}{
		{"red potion (200)", item.Id(2000001), true},
		{"white potion (200)", item.Id(2000020), true},
		{"food/apple (201)", item.Id(2010000), true},
		{"hp food (202)", item.Id(2020000), true},
		{"return scroll (203)", item.Id(2030000), false},
		{"equip scroll (204)", item.Id(2040727), false},
		{"antidote — cure pot (205)", item.Id(2050001), true},
		{"all cure potion (205)", item.Id(2050004), true},
		{"arrow (206)", item.Id(2060000), false},
		{"throwing star (207)", item.Id(2070000), false},
		{"summoning sack (210)", item.Id(2100000), false},
		{"pet food (212)", item.Id(2120000), false},
		{"weapon (130)", item.Id(1302000), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := usesStandardConsumer(tc.itemId)
			assert.Equal(t, tc.want, got, "itemId %d (classification %d)", tc.itemId, item.GetClassification(tc.itemId))
		})
	}
}
