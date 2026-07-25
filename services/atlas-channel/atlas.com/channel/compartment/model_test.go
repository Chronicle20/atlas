package compartment

import (
	"atlas-channel/asset"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestFindFirstByClassification(t *testing.T) {
	cid := uuid.New()

	mustAsset := func(id uint32, templateId uint32, slot int16) asset.Model {
		a, err := asset.NewModelBuilder(id, cid, templateId).SetSlot(slot).SetQuantity(1).Build()
		if err != nil {
			t.Fatalf("asset build: %v", err)
		}
		return a
	}

	m, err := NewBuilder(cid, 100, inventory.TypeValueCash, 96).
		AddAsset(mustAsset(1, 2000000, 1)). // classification != 509
		AddAsset(mustAsset(2, 5090000, 2)). // Note (509) — first match
		AddAsset(mustAsset(3, 5090001, 3)). // Note (509)
		Build()
	if err != nil {
		t.Fatalf("compartment build: %v", err)
	}

	a, found := m.FindFirstByClassification(item.ClassificationNote)
	if !found {
		t.Fatal("expected a Note-classified asset")
	}
	if a.TemplateId() != 5090000 {
		t.Errorf("templateId: got %d, want 5090000 (first match wins)", a.TemplateId())
	}

	_, found = m.FindFirstByClassification(item.Classification(999))
	if found {
		t.Error("expected no match for classification 999")
	}
}
