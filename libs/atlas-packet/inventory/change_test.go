package inventory

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-packet/test"
)

func TestQuantityUpdateRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewQuantityUpdate(true, 2, 5, 100)
			output := QuantityUpdate{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Quantity() != input.Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Quantity(), input.Quantity())
			}
		})
	}
}

func TestChangeMoveRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// non-equip type, no addMov byte
			input := NewChangeMove(false, 2, 3, 7)
			output := ChangeMove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.OldSlot() != input.OldSlot() {
				t.Errorf("oldSlot: got %v, want %v", output.OldSlot(), input.OldSlot())
			}
			if output.NewSlot() != input.NewSlot() {
				t.Errorf("newSlot: got %v, want %v", output.NewSlot(), input.NewSlot())
			}
		})
	}
}

func TestChangeMoveEquipToSlotRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// equip type, newSlot < 0 => addMov = 2
			input := NewChangeMove(true, 1, 5, -1)
			output := ChangeMove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OldSlot() != input.OldSlot() {
				t.Errorf("oldSlot: got %v, want %v", output.OldSlot(), input.OldSlot())
			}
			if output.NewSlot() != input.NewSlot() {
				t.Errorf("newSlot: got %v, want %v", output.NewSlot(), input.NewSlot())
			}
		})
	}
}

func TestChangeMoveUnequipRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// equip type, oldSlot < 0 => addMov = 1
			input := NewChangeMove(true, 1, -1, 5)
			output := ChangeMove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OldSlot() != input.OldSlot() {
				t.Errorf("oldSlot: got %v, want %v", output.OldSlot(), input.OldSlot())
			}
			if output.NewSlot() != input.NewSlot() {
				t.Errorf("newSlot: got %v, want %v", output.NewSlot(), input.NewSlot())
			}
		})
	}
}

func TestRemoveRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// non-equip type, no addMov byte
			input := NewInventoryRemove(false, 2, 3)
			output := Remove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
		})
	}
}

func TestRemoveEquipRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// equip type, slot < 0 => addMov byte written
			input := NewInventoryRemove(true, 1, -1)
			output := Remove{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
		})
	}
}

func TestAddStackableRoundTrip(t *testing.T) {
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).
		SetStackableInfo(5, 0, 0)
	input := NewInventoryAdd(false, 2, 5, asset)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Add{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Silent() != input.Silent() {
				t.Errorf("silent: got %v, want %v", output.Silent(), input.Silent())
			}
			if output.InventoryType() != input.InventoryType() {
				t.Errorf("inventoryType: got %v, want %v", output.InventoryType(), input.InventoryType())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Asset().TemplateId() != input.Asset().TemplateId() {
				t.Errorf("templateId: got %v, want %v", output.Asset().TemplateId(), input.Asset().TemplateId())
			}
			if output.Asset().Quantity() != input.Asset().Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Asset().Quantity(), input.Asset().Quantity())
			}
		})
	}
}

func TestAddEquipmentRoundTrip(t *testing.T) {
	asset := model.NewAsset(true, 0, 1302000, time.Time{}).
		SetEquipmentStats(5, 3, 2, 1, 10, 5, 15, 8, 4, 3, 7, 6, 10, 5, 3).
		SetEquipmentMeta(7, 0, 0, 0, 0, 0)
	input := NewInventoryAdd(false, 1, -1, asset)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Add{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Asset().TemplateId() != input.Asset().TemplateId() {
				t.Errorf("templateId: got %v, want %v", output.Asset().TemplateId(), input.Asset().TemplateId())
			}
			if output.Asset().Strength() != input.Asset().Strength() {
				t.Errorf("strength: got %v, want %v", output.Asset().Strength(), input.Asset().Strength())
			}
			if output.Asset().Slots() != input.Asset().Slots() {
				t.Errorf("slots: got %v, want %v", output.Asset().Slots(), input.Asset().Slots())
			}
		})
	}
}
