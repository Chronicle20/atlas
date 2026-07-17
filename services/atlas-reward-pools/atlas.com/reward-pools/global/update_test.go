package global_test

import (
	"atlas-reward-pools/global"
	"atlas-reward-pools/test"
	"testing"

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
