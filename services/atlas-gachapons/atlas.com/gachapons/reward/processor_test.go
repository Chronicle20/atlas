package reward_test

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"atlas-gachapons/test"
	"testing"
)

func TestSelectReward(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	// Create a test gachapon
	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-1").
		SetName("Test Gachapon").
		SetNpcIds([]uint32{9100100}).
		SetCommonWeight(70).
		SetUncommonWeight(25).
		SetRareWeight(5).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}

	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	// Create items for all tiers
	commonItem, err := item.NewBuilder(tenantId, 0).
		SetGachaponId("test-gachapon-1").
		SetItemId(2000000).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build common item: %v", err)
	}
	err = item.CreateItem(db, commonItem)
	if err != nil {
		t.Fatalf("Failed to create common item: %v", err)
	}

	uncommonItem, err := item.NewBuilder(tenantId, 0).
		SetGachaponId("test-gachapon-1").
		SetItemId(2000001).
		SetQuantity(2).
		SetTier("uncommon").
		Build()
	if err != nil {
		t.Fatalf("Failed to build uncommon item: %v", err)
	}
	err = item.CreateItem(db, uncommonItem)
	if err != nil {
		t.Fatalf("Failed to create uncommon item: %v", err)
	}

	rareItem, err := item.NewBuilder(tenantId, 0).
		SetGachaponId("test-gachapon-1").
		SetItemId(2000002).
		SetQuantity(3).
		SetTier("rare").
		Build()
	if err != nil {
		t.Fatalf("Failed to build rare item: %v", err)
	}
	err = item.CreateItem(db, rareItem)
	if err != nil {
		t.Fatalf("Failed to create rare item: %v", err)
	}

	// Select a reward multiple times to verify it works
	validItemIds := map[uint32]bool{2000000: true, 2000001: true, 2000002: true}
	validTiers := map[string]bool{"common": true, "uncommon": true, "rare": true}

	for i := 0; i < 10; i++ {
		result, err := processor.SelectReward("test-gachapon-1")
		if err != nil {
			t.Fatalf("Failed to select reward: %v", err)
		}

		if !validItemIds[result.ItemId()] {
			t.Errorf("Unexpected item ID: %d", result.ItemId())
		}

		if !validTiers[result.Tier()] {
			t.Errorf("Unexpected tier: %s", result.Tier())
		}

		if result.GachaponId() != "test-gachapon-1" {
			t.Errorf("Expected gachapon ID 'test-gachapon-1', got '%s'", result.GachaponId())
		}
	}
}

func TestSelectRewardWithGlobalItems(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	// Create a gachapon with only common weight (no machine-specific items)
	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-2").
		SetName("Global Items Gachapon").
		SetNpcIds([]uint32{9100101}).
		SetCommonWeight(100).
		SetUncommonWeight(0).
		SetRareWeight(0).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}

	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	// Create a global common item
	globalItem, err := global.NewBuilder(tenantId, 0).
		SetItemId(3000000).
		SetQuantity(5).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build global item: %v", err)
	}
	err = global.CreateItem(db, globalItem)
	if err != nil {
		t.Fatalf("Failed to create global item: %v", err)
	}

	// Select a reward - should get the global item
	result, err := processor.SelectReward("test-gachapon-2")
	if err != nil {
		t.Fatalf("Failed to select reward: %v", err)
	}

	if result.ItemId() != 3000000 {
		t.Errorf("Expected item ID 3000000, got %d", result.ItemId())
	}

	if result.Quantity() != 5 {
		t.Errorf("Expected quantity 5, got %d", result.Quantity())
	}

	if result.Tier() != "common" {
		t.Errorf("Expected tier 'common', got '%s'", result.Tier())
	}
}

func TestSelectRewardEmptyPool(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	// Create a gachapon with no items (and no global items)
	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-3").
		SetName("Empty Gachapon").
		SetNpcIds([]uint32{9100102}).
		SetCommonWeight(100).
		SetUncommonWeight(0).
		SetRareWeight(0).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}

	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	// Select a reward - should fail because pool is empty
	_, err = processor.SelectReward("test-gachapon-3")
	if err == nil {
		t.Error("Expected error when selecting from empty pool, got nil")
	}
}

func TestSelectRewardGachaponNotFound(t *testing.T) {
	processor, _, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	// Select a reward from non-existent gachapon
	_, err := processor.SelectReward("non-existent-gachapon")
	if err == nil {
		t.Error("Expected error when gachapon not found, got nil")
	}
}

func TestGetPrizePool(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	// Create a gachapon with items
	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-4").
		SetName("Prize Pool Gachapon").
		SetNpcIds([]uint32{9100103}).
		SetCommonWeight(70).
		SetUncommonWeight(25).
		SetRareWeight(5).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}

	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	// Create items for different tiers
	for i, tier := range []string{"common", "uncommon", "rare"} {
		itemModel, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("test-gachapon-4").
			SetItemId(uint32(4000000 + i)).
			SetQuantity(1).
			SetTier(tier).
			Build()
		if err != nil {
			t.Fatalf("Failed to build item: %v", err)
		}
		err = item.CreateItem(db, itemModel)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
	}

	// Get full prize pool
	pool, err := processor.GetPrizePool("test-gachapon-4", "")
	if err != nil {
		t.Fatalf("Failed to get prize pool: %v", err)
	}

	if len(pool) != 3 {
		t.Errorf("Expected 3 items in pool, got %d", len(pool))
	}

	// Verify all tiers are represented
	tierCounts := make(map[string]int)
	for _, r := range pool {
		tierCounts[r.Tier()]++
	}

	for _, tier := range []string{"common", "uncommon", "rare"} {
		if tierCounts[tier] != 1 {
			t.Errorf("Expected 1 %s item, got %d", tier, tierCounts[tier])
		}
	}
}

func TestGetPrizePoolByTier(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	// Create a gachapon with items
	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-5").
		SetName("Prize Pool By Tier Gachapon").
		SetNpcIds([]uint32{9100104}).
		SetCommonWeight(70).
		SetUncommonWeight(25).
		SetRareWeight(5).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}

	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	// Create items for different tiers
	for i, tier := range []string{"common", "uncommon", "rare"} {
		itemModel, err := item.NewBuilder(tenantId, 0).
			SetGachaponId("test-gachapon-5").
			SetItemId(uint32(5000000 + i)).
			SetQuantity(1).
			SetTier(tier).
			Build()
		if err != nil {
			t.Fatalf("Failed to build item: %v", err)
		}
		err = item.CreateItem(db, itemModel)
		if err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
	}

	// Get prize pool for specific tier
	pool, err := processor.GetPrizePool("test-gachapon-5", "rare")
	if err != nil {
		t.Fatalf("Failed to get prize pool by tier: %v", err)
	}

	if len(pool) != 1 {
		t.Errorf("Expected 1 rare item, got %d", len(pool))
	}

	if len(pool) > 0 && pool[0].Tier() != "rare" {
		t.Errorf("Expected rare tier, got %s", pool[0].Tier())
	}
}

// TestSelectRewardIncubatorKind seeds an incubator-kind gachapon whose
// tier weights are ALL zero (which would make selectTier error out with
// "total weight cannot be zero" if it were ever invoked), two
// machine-scoped weighted items across different tiers, and a global item
// on the "common" tier that must never be reachable. It asserts the reward
// always comes from the machine pool, never the global pool, and that
// SelectReward succeeds despite the zero tier weights — proving
// selectTier was never called for an incubator-kind machine.
func TestSelectRewardIncubatorKind(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-incubator-1").
		SetName("Test Incubator").
		SetNpcIds([]uint32{9100200}).
		SetCommonWeight(0).
		SetUncommonWeight(0).
		SetRareWeight(0).
		SetKind("incubator").
		Build()
	if err != nil {
		t.Fatalf("Failed to build incubator gachapon model: %v", err)
	}
	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create incubator gachapon: %v", err)
	}

	commonItem, err := item.NewBuilder(tenantId, 0).
		SetGachaponId("test-gachapon-incubator-1").
		SetItemId(7000000).
		SetQuantity(1).
		SetTier("common").
		SetWeight(1).
		Build()
	if err != nil {
		t.Fatalf("Failed to build incubator common item: %v", err)
	}
	err = item.CreateItem(db, commonItem)
	if err != nil {
		t.Fatalf("Failed to create incubator common item: %v", err)
	}

	rareItem, err := item.NewBuilder(tenantId, 0).
		SetGachaponId("test-gachapon-incubator-1").
		SetItemId(7000001).
		SetQuantity(1).
		SetTier("rare").
		SetWeight(3).
		Build()
	if err != nil {
		t.Fatalf("Failed to build incubator rare item: %v", err)
	}
	err = item.CreateItem(db, rareItem)
	if err != nil {
		t.Fatalf("Failed to create incubator rare item: %v", err)
	}

	// Global item on the "common" tier — must never appear in an incubator
	// roll, since incubator kind does not merge the global pool.
	globalItem, err := global.NewBuilder(tenantId, 0).
		SetItemId(7000002).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build global item: %v", err)
	}
	err = global.CreateItem(db, globalItem)
	if err != nil {
		t.Fatalf("Failed to create global item: %v", err)
	}

	validItemIds := map[uint32]bool{7000000: true, 7000001: true}
	seen := map[uint32]bool{}

	for i := 0; i < 100; i++ {
		result, err := processor.SelectReward("test-gachapon-incubator-1")
		if err != nil {
			t.Fatalf("Failed to select incubator reward: %v", err)
		}

		if !validItemIds[result.ItemId()] {
			t.Fatalf("Unexpected incubator item ID: %d (expected only machine items 7000000/7000001)", result.ItemId())
		}
		if result.ItemId() == 7000002 {
			t.Fatalf("Global item 7000002 must never appear in an incubator-kind roll")
		}
		if result.Tier() != "" {
			t.Errorf("Expected empty tier for incubator-kind reward, got %q", result.Tier())
		}

		seen[result.ItemId()] = true
	}

	if !seen[7000000] {
		t.Error("expected weight-1 incubator item 7000000 to appear across 100 draws, never did")
	}
	if !seen[7000001] {
		t.Error("expected weight-3 incubator item 7000001 to appear across 100 draws, never did")
	}
}

// TestSelectRewardGachaponKindUnchanged is the explicit control case for
// the incubator branch above: a gachapon-kind machine (the default, and
// the only kind that existed before this feature) must still go through
// selectTier -> getMergedPool -> selectItem, reaching both machine and
// global items for the tier it lands on.
func TestSelectRewardGachaponKindUnchanged(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-classic-1").
		SetName("Test Classic Gachapon").
		SetNpcIds([]uint32{9100201}).
		SetCommonWeight(100).
		SetUncommonWeight(0).
		SetRareWeight(0).
		Build()
	if err != nil {
		t.Fatalf("Failed to build classic gachapon model: %v", err)
	}
	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create classic gachapon: %v", err)
	}

	if gachaponModel.Kind() != "gachapon" {
		t.Fatalf("Expected default Kind() = %q, got %q", "gachapon", gachaponModel.Kind())
	}

	machineItem, err := item.NewBuilder(tenantId, 0).
		SetGachaponId("test-gachapon-classic-1").
		SetItemId(8000000).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build machine item: %v", err)
	}
	err = item.CreateItem(db, machineItem)
	if err != nil {
		t.Fatalf("Failed to create machine item: %v", err)
	}

	globalItem, err := global.NewBuilder(tenantId, 0).
		SetItemId(8000001).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build global item: %v", err)
	}
	err = global.CreateItem(db, globalItem)
	if err != nil {
		t.Fatalf("Failed to create global item: %v", err)
	}

	validItemIds := map[uint32]bool{8000000: true, 8000001: true}
	seen := map[uint32]bool{}

	for i := 0; i < 100; i++ {
		result, err := processor.SelectReward("test-gachapon-classic-1")
		if err != nil {
			t.Fatalf("Failed to select classic reward: %v", err)
		}
		if !validItemIds[result.ItemId()] {
			t.Fatalf("Unexpected classic item ID: %d", result.ItemId())
		}
		if result.Tier() != "common" {
			t.Errorf("Expected tier 'common', got %q", result.Tier())
		}
		seen[result.ItemId()] = true
	}

	if !seen[8000000] {
		t.Error("expected machine item 8000000 to appear across 100 draws, never did")
	}
	if !seen[8000001] {
		t.Error("expected global item 8000001 to appear across 100 draws, never did")
	}
}

func TestGetPrizePoolMergesGlobalItems(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	tenantId := test.TestTenantId

	// Create a gachapon
	gachaponModel, err := gachapon.NewBuilder(tenantId, "test-gachapon-6").
		SetName("Merged Pool Gachapon").
		SetNpcIds([]uint32{9100105}).
		SetCommonWeight(100).
		SetUncommonWeight(0).
		SetRareWeight(0).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}

	err = gachapon.CreateGachapon(db, gachaponModel)
	if err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	// Create a machine-specific common item
	machineItem, err := item.NewBuilder(tenantId, 0).
		SetGachaponId("test-gachapon-6").
		SetItemId(6000000).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build machine item: %v", err)
	}
	err = item.CreateItem(db, machineItem)
	if err != nil {
		t.Fatalf("Failed to create machine item: %v", err)
	}

	// Create a global common item
	globalItem, err := global.NewBuilder(tenantId, 0).
		SetItemId(6000001).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build global item: %v", err)
	}
	err = global.CreateItem(db, globalItem)
	if err != nil {
		t.Fatalf("Failed to create global item: %v", err)
	}

	// Get prize pool - should have both items
	pool, err := processor.GetPrizePool("test-gachapon-6", "common")
	if err != nil {
		t.Fatalf("Failed to get prize pool: %v", err)
	}

	if len(pool) != 2 {
		t.Errorf("Expected 2 common items (1 machine + 1 global), got %d", len(pool))
	}

	// Verify both item IDs are present
	itemIds := make(map[uint32]bool)
	for _, r := range pool {
		itemIds[r.ItemId()] = true
	}

	if !itemIds[6000000] {
		t.Error("Expected machine item 6000000 in pool")
	}
	if !itemIds[6000001] {
		t.Error("Expected global item 6000001 in pool")
	}
}
