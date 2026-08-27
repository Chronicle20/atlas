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
		a, err := asset.NewBuilderWithId(id, cid, templateId).SetSlot(slot).SetQuantity(1).Build()
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

func qtyAsset(t *testing.T, slot int16, templateId uint32, qty uint32) asset.Model {
	t.Helper()
	a, err := asset.NewBuilderWithId(uint32(slot), uuid.New(), templateId).SetSlot(slot).SetQuantity(qty).Build()
	if err != nil {
		t.Fatalf("asset build: %v", err)
	}
	return a
}

func qtyCompartment(t *testing.T, assets ...asset.Model) Model {
	t.Helper()
	b := NewBuilder(uuid.New(), 1, inventory.TypeValueUse, 96)
	for _, a := range assets {
		b.AddAsset(a)
	}
	m, err := b.Build()
	if err != nil {
		t.Fatalf("compartment build: %v", err)
	}
	return m
}

func TestFindFirstByItemIdWithQuantity_LowestSlotWinsUnsortedInput(t *testing.T) {
	// Assets deliberately out of slot order; both qualify — slot 2 must win.
	m := qtyCompartment(t, qtyAsset(t, 5, 4006000, 10), qtyAsset(t, 2, 4006000, 3))
	a, found := m.FindFirstByItemIdWithQuantity(4006000, 2)
	if !found || a.Slot() != 2 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=2, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_SkipsShortSlots(t *testing.T) {
	// Slot 1 is short (1 < 2); slot 3 qualifies.
	m := qtyCompartment(t, qtyAsset(t, 1, 4006000, 1), qtyAsset(t, 3, 4006000, 2))
	a, found := m.FindFirstByItemIdWithQuantity(4006000, 2)
	if !found || a.Slot() != 3 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=3, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_ExactBoundary(t *testing.T) {
	m := qtyCompartment(t, qtyAsset(t, 1, 2070000, 200))
	a, found := m.FindFirstByItemIdWithQuantity(2070000, 200)
	if !found || a.Slot() != 1 {
		t.Fatalf("got (slot=%v, found=%v), want (slot=1, found=true)", a, found)
	}
}

func TestFindFirstByItemIdWithQuantity_NoSlotQualifies(t *testing.T) {
	// Aggregate 300 across two slots, but no single slot holds 200.
	m := qtyCompartment(t, qtyAsset(t, 1, 2070000, 150), qtyAsset(t, 2, 2070000, 150))
	if _, found := m.FindFirstByItemIdWithQuantity(2070000, 200); found {
		t.Fatal("expected not found: no single slot holds 200")
	}
}

func TestFindFirstByItemIdWithQuantity_ItemAbsent(t *testing.T) {
	m := qtyCompartment(t, qtyAsset(t, 1, 4006000, 10))
	if _, found := m.FindFirstByItemIdWithQuantity(4006001, 1); found {
		t.Fatal("expected not found: template id absent")
	}
}
