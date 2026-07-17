package item_test

import (
	"atlas-reward-pools/item"
	"atlas-reward-pools/test"
	"testing"
)

func TestUpdateItem(t *testing.T) {
	processor, db, cleanup := test.CreateItemProcessor(t)
	defer cleanup()

	m, err := item.NewBuilder(test.TestTenantId, 0).
		SetGachaponId("4170001").
		SetItemId(2000000).
		SetQuantity(1).
		SetTier("common").
		SetWeight(50).
		Build()
	if err != nil {
		t.Fatalf("Failed to build item model: %v", err)
	}
	if err = item.CreateItem(db, m); err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	created, err := processor.GetByGachaponId("4170001")()
	if err != nil || len(created) != 1 {
		t.Fatalf("Expected 1 item, got %d (err %v)", len(created), err)
	}
	id := created[0].Id()

	if err = processor.Update(id, 2000001, 3, "uncommon", 75); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	after, err := processor.GetByGachaponId("4170001")()
	if err != nil || len(after) != 1 {
		t.Fatalf("Expected 1 item after update, got %d (err %v)", len(after), err)
	}
	got := after[0]
	if got.ItemId() != 2000001 || got.Quantity() != 3 || got.Weight() != 75 || got.Tier() != "uncommon" {
		t.Errorf("Update not applied: itemId=%d qty=%d weight=%d tier=%q",
			got.ItemId(), got.Quantity(), got.Weight(), got.Tier())
	}
	if got.GachaponId() != "4170001" {
		t.Errorf("Update must not re-parent: gachaponId=%q", got.GachaponId())
	}
}
