package consumable

import (
	"testing"

	"atlas-consumables/asset"
	"atlas-consumables/compartment"
	consumable3 "atlas-consumables/data/consumable"
	inventoryModel "atlas-consumables/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/google/uuid"
)

// compartmentWith builds a compartment of the given type with `capacity` slots,
// `occupied` of them filled (positive slots 1..occupied).
func compartmentWith(it inventory.Type, capacity uint32, occupied int) compartment.Model {
	cid := uuid.New()
	b := compartment.NewBuilder(cid, 1, it, capacity)
	for s := 1; s <= occupied; s++ {
		b.AddAsset(asset.NewBuilder(cid, 9000000).SetSlot(int16(s)).Build())
	}
	return b.Build()
}

func invWith(cs ...compartment.Model) inventoryModel.Model {
	b := inventoryModel.NewBuilder(1)
	for _, c := range cs {
		b.SetCompartment(c)
	}
	return b.Build()
}

// The strict rule: every distinct inventory type in the pool must have a free
// slot, or the box use is rejected up front.
func TestInventoryAccommodatesRewards(t *testing.T) {
	// Pool spans EQUIP (1132010) and USE (2000002).
	pool := []consumable3.RewardModel{rw(1132010, 1, 5), rw(2000002, 30, 5)}

	// Both types have a free slot -> accommodated.
	if !inventoryAccommodatesRewards(invWith(
		compartmentWith(inventory.TypeValueEquip, 24, 10),
		compartmentWith(inventory.TypeValueUse, 24, 10),
	), pool) {
		t.Fatal("expected accommodated when both EQUIP and USE have room")
	}

	// EQUIP full, USE has room -> NOT accommodated (an EQUIP reward couldn't fit).
	if inventoryAccommodatesRewards(invWith(
		compartmentWith(inventory.TypeValueEquip, 24, 24),
		compartmentWith(inventory.TypeValueUse, 24, 0),
	), pool) {
		t.Fatal("expected rejection when EQUIP is full even though USE has room")
	}

	// USE full, EQUIP has room -> NOT accommodated.
	if inventoryAccommodatesRewards(invWith(
		compartmentWith(inventory.TypeValueEquip, 24, 0),
		compartmentWith(inventory.TypeValueUse, 24, 24),
	), pool) {
		t.Fatal("expected rejection when USE is full")
	}

	// A single-type (USE-only) pool ignores a full EQUIP tab.
	usePool := []consumable3.RewardModel{rw(2000002, 30, 5), rw(2000003, 20, 5)}
	if !inventoryAccommodatesRewards(invWith(
		compartmentWith(inventory.TypeValueEquip, 24, 24),
		compartmentWith(inventory.TypeValueUse, 24, 10),
	), usePool) {
		t.Fatal("expected accommodated for a USE-only pool when USE has room, regardless of EQUIP")
	}
}
