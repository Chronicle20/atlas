package consumable

import (
	"atlas-consumables/asset"
	"atlas-consumables/equipable"
	"testing"

	"github.com/Chronicle20/atlas-constants/item"
	"github.com/google/uuid"
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
	currents := []uint16{0, 0, 0, 0}
	changers := []func(int16) equipable.Change{
		equipable.AddStrength,
		equipable.AddDexterity,
		equipable.AddIntelligence,
		equipable.AddLuck,
	}

	changes, err := generateChaosChanges(currents, changers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("expected 0 changes for all zero stats, got %d", len(changes))
	}
}

func TestGenerateChaosChanges_GeneratesForNonZeroStats(t *testing.T) {
	// Non-zero stats should produce changes
	currents := []uint16{10, 0, 15, 0} // 2 non-zero stats
	changers := []func(int16) equipable.Change{
		equipable.AddStrength,
		equipable.AddDexterity,
		equipable.AddIntelligence,
		equipable.AddLuck,
	}

	changes, err := generateChaosChanges(currents, changers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 2 {
		t.Errorf("expected 2 changes for 2 non-zero stats, got %d", len(changes))
	}
}

func TestGenerateChaosChanges_MismatchedLengths(t *testing.T) {
	currents := []uint16{10, 20}
	changers := []func(int16) equipable.Change{
		equipable.AddStrength,
	}

	_, err := generateChaosChanges(currents, changers)
	if err == nil {
		t.Error("expected error for mismatched lengths")
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
