package consumable

import (
	"testing"

	"atlas-consumables/asset"
	"atlas-consumables/character"
	"atlas-consumables/compartment"
	"atlas-consumables/equipment"
	"atlas-consumables/inventory"

	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
)

func TestVegaRates(t *testing.T) {
	cases := []struct {
		name         string
		id           item2.Id
		wantRequired uint32
		wantBoosted  uint32
		wantOk       bool
	}{
		{"Vega's Spell 10", item2.VegasSpell10, 10, 30, true},
		{"Vega's Spell 60", item2.VegasSpell60, 60, 90, true},
		{"non-vega cash item", item2.Id(5610002), 0, 0, false},
		{"scroll id", item2.ChaosScrollSixtyPercent, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			required, boosted, ok := vegaRates(tc.id)
			if required != tc.wantRequired || boosted != tc.wantBoosted || ok != tc.wantOk {
				t.Errorf("vegaRates(%d) = (%d, %d, %t), want (%d, %d, %t)",
					tc.id, required, boosted, ok, tc.wantRequired, tc.wantBoosted, tc.wantOk)
			}
		})
	}
}

func TestResolveVegaEquip_PositiveSlotFromEquipInventory(t *testing.T) {
	equip := asset.NewBuilder(uuid.New(), 1302000).SetId(1).SetSlot(3).SetSlots(7).Build()
	comp := compartment.NewBuilder(uuid.New(), 1, inventory2.TypeValueEquip, 96).AddAsset(equip).Build()
	inv := inventory.NewBuilder(1).SetEquipable(comp).Build()
	c := character.NewModelBuilder().SetId(1).SetInventory(inv).SetEquipment(equipment.NewModel()).Build()

	got, err := resolveVegaEquip(c, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateId() != 1302000 {
		t.Errorf("resolved template: got %d, want 1302000", got.TemplateId())
	}
}

func TestResolveVegaEquip_PositiveSlotMissing(t *testing.T) {
	comp := compartment.NewBuilder(uuid.New(), 1, inventory2.TypeValueEquip, 96).Build()
	inv := inventory.NewBuilder(1).SetEquipable(comp).Build()
	c := character.NewModelBuilder().SetId(1).SetInventory(inv).SetEquipment(equipment.NewModel()).Build()

	if _, err := resolveVegaEquip(c, 3); err == nil {
		t.Error("expected error for empty slot, got nil")
	}
}

func TestResolveVegaEquip_NegativeSlotFromEquipped(t *testing.T) {
	weapon := asset.NewBuilder(uuid.New(), 1302000).SetId(1).SetSlots(7).Build()
	eq := equipment.NewModel()
	s, err := slot.GetSlotByPosition(slot.Position(-11)) // weapon slot
	if err != nil {
		t.Fatalf("slot lookup: %v", err)
	}
	sm, _ := eq.Get(s.Type)
	sm.Equipable = &weapon
	eq.Set(s.Type, sm)
	c := character.NewModelBuilder().SetId(1).SetEquipment(eq).Build()

	got, err := resolveVegaEquip(c, -11)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TemplateId() != 1302000 {
		t.Errorf("resolved template: got %d, want 1302000", got.TemplateId())
	}
}

func TestResolveVegaEquip_NegativeSlotEmpty(t *testing.T) {
	c := character.NewModelBuilder().SetId(1).SetEquipment(equipment.NewModel()).Build()
	if _, err := resolveVegaEquip(c, -11); err == nil {
		t.Error("expected error for empty equipped slot, got nil")
	}
}
