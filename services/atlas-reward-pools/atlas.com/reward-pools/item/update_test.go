package item_test

import (
	"atlas-reward-pools/item"
	"atlas-reward-pools/test"
	"errors"
	"testing"

	"gorm.io/gorm"
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

func TestUpdateItemInvalidTier(t *testing.T) {
	processor, db, cleanup := test.CreateItemProcessor(t)
	defer cleanup()

	// A gachaponId distinct from TestUpdateItem's: the sqlite
	// "file::memory:?cache=shared" DSN backs every test in this package with
	// the same underlying store, so reusing an id here would collide with
	// rows other test functions leave behind.
	const gachaponId = "4170002"
	m, err := item.NewBuilder(test.TestTenantId, 0).
		SetGachaponId(gachaponId).
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

	created, err := processor.GetByGachaponId(gachaponId)()
	if err != nil || len(created) != 1 {
		t.Fatalf("Expected 1 item, got %d (err %v)", len(created), err)
	}
	id := created[0].Id()

	tests := []string{"", "epic", "COMMON", "legendary"}
	for _, tier := range tests {
		t.Run(tier, func(t *testing.T) {
			err := processor.Update(id, 2000001, 3, tier, 75)
			if !errors.Is(err, item.ErrInvalidTier) {
				t.Fatalf("Expected ErrInvalidTier for tier %q, got %v", tier, err)
			}

			after, err := processor.GetByGachaponId(gachaponId)()
			if err != nil || len(after) != 1 {
				t.Fatalf("Expected 1 item, got %d (err %v)", len(after), err)
			}
			got := after[0]
			if got.ItemId() != 2000000 || got.Quantity() != 1 || got.Weight() != 50 || got.Tier() != "common" {
				t.Errorf("Row must be unchanged after rejected update: itemId=%d qty=%d weight=%d tier=%q",
					got.ItemId(), got.Quantity(), got.Weight(), got.Tier())
			}
		})
	}
}

func TestUpdateItemNotFound(t *testing.T) {
	processor, _, cleanup := test.CreateItemProcessor(t)
	defer cleanup()

	err := processor.Update(999999, 2000001, 3, "uncommon", 75)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Expected gorm.ErrRecordNotFound for nonexistent id, got %v", err)
	}
}
