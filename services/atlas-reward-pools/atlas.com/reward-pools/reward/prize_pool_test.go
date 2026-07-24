package reward_test

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/item"
	"atlas-reward-pools/test"
	"testing"
)

func TestGetPrizePoolIncubator(t *testing.T) {
	processor, db, cleanup := test.CreateRewardProcessor(t)
	defer cleanup()

	g, err := gachapon.NewBuilder(test.TestTenantId, "4170001").
		SetName("Pigmy Egg (Victoria)").
		SetNpcIds([]uint32{1012004}).
		SetKind(gachapon.KindIncubator).
		Build()
	if err != nil {
		t.Fatalf("Failed to build incubator pool: %v", err)
	}
	if err = gachapon.CreateGachapon(db, g); err != nil {
		t.Fatalf("Failed to create incubator pool: %v", err)
	}

	for _, spec := range []struct {
		itemId uint32
		weight uint32
	}{{2000000, 50}, {1302000, 5}} {
		m, err := item.NewBuilder(test.TestTenantId, 0).
			SetGachaponId("4170001").
			SetItemId(spec.itemId).
			SetQuantity(1).
			SetTier("common").
			SetWeight(spec.weight).
			Build()
		if err != nil {
			t.Fatalf("Failed to build item: %v", err)
		}
		if err = item.CreateItem(db, m); err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}
	}

	pool, err := processor.GetPrizePool("4170001", "")
	if err != nil {
		t.Fatalf("GetPrizePool failed: %v", err)
	}
	if len(pool) != 2 {
		t.Fatalf("Incubator prize pool must return the weighted items, got %d", len(pool))
	}
	weights := map[uint32]uint32{}
	for _, m := range pool {
		if m.Tier() != "" {
			t.Errorf("Incubator rows carry no tier, got %q", m.Tier())
		}
		weights[m.ItemId()] = m.Weight()
	}
	if weights[2000000] != 50 || weights[1302000] != 5 {
		t.Errorf("Weights not threaded: %v", weights)
	}
}
