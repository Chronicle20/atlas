package global_test

import (
	"atlas-reward-pools/global"
	"atlas-reward-pools/test"
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func TestUpdateGlobalItem(t *testing.T) {
	processor, db, cleanup := test.CreateGlobalProcessor(t)
	defer cleanup()

	m, err := global.NewBuilder(test.TestTenantId, 0).
		SetItemId(2000000).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build global item model: %v", err)
	}
	if err = global.CreateItem(db, m); err != nil {
		t.Fatalf("Failed to create global item: %v", err)
	}

	paged, err := processor.GetAll(model.Page{Number: 1, Size: 10})()
	if err != nil || len(paged.Items) != 1 {
		t.Fatalf("Expected 1 global item, got %d (err %v)", len(paged.Items), err)
	}
	id := paged.Items[0].Id()

	if err = processor.Update(id, 2000001, 5, "rare"); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	after, err := processor.GetAll(model.Page{Number: 1, Size: 10})()
	if err != nil || len(after.Items) != 1 {
		t.Fatalf("Expected 1 global item after update, got %d (err %v)", len(after.Items), err)
	}
	got := after.Items[0]
	if got.ItemId() != 2000001 || got.Quantity() != 5 || got.Tier() != "rare" {
		t.Errorf("Update not applied: itemId=%d qty=%d tier=%q", got.ItemId(), got.Quantity(), got.Tier())
	}
}

func TestUpdateGlobalItemInvalidTier(t *testing.T) {
	processor, db, cleanup := test.CreateGlobalProcessor(t)
	defer cleanup()

	// itemId is unique within this package's shared sqlite
	// "file::memory:?cache=shared" test store (see TestUpdateGlobalItem)
	// so this test's row can be located by itemId rather than assuming
	// GetAll returns only what this test created.
	const itemId = 2100000
	m, err := global.NewBuilder(test.TestTenantId, 0).
		SetItemId(itemId).
		SetQuantity(1).
		SetTier("common").
		Build()
	if err != nil {
		t.Fatalf("Failed to build global item model: %v", err)
	}
	if err = global.CreateItem(db, m); err != nil {
		t.Fatalf("Failed to create global item: %v", err)
	}

	id := findGlobalItemByItemId(t, processor, itemId).Id()

	tests := []string{"", "epic", "COMMON", "legendary"}
	for _, tier := range tests {
		t.Run(tier, func(t *testing.T) {
			err := processor.Update(id, itemId+1, 5, tier)
			if !errors.Is(err, global.ErrInvalidTier) {
				t.Fatalf("Expected ErrInvalidTier for tier %q, got %v", tier, err)
			}

			got := findGlobalItemByItemId(t, processor, itemId)
			if got.Quantity() != 1 || got.Tier() != "common" {
				t.Errorf("Row must be unchanged after rejected update: itemId=%d qty=%d tier=%q",
					got.ItemId(), got.Quantity(), got.Tier())
			}
		})
	}
}

// findGlobalItemByItemId scans every page for the row with the given
// itemId, failing the test if it is not found.
func findGlobalItemByItemId(t *testing.T, processor global.Processor, itemId uint32) global.Model {
	t.Helper()
	paged, err := processor.GetAll(model.Page{Number: 1, Size: 100})()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	for _, m := range paged.Items {
		if m.ItemId() == itemId {
			return m
		}
	}
	t.Fatalf("Expected a global item with itemId=%d, found none among %d items", itemId, len(paged.Items))
	return global.Model{}
}

func TestUpdateGlobalItemNotFound(t *testing.T) {
	processor, _, cleanup := test.CreateGlobalProcessor(t)
	defer cleanup()

	err := processor.Update(999999, 2000001, 5, "rare")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Expected gorm.ErrRecordNotFound for nonexistent id, got %v", err)
	}
}
