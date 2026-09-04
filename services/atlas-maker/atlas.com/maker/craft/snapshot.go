// Package craft evaluates per-character recipe eligibility (FR-2.1, FR-2.2,
// FR-3.5, design §4.2.2).
package craft

import (
	"atlas-maker/compartment"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// SlotHolding is one compartment slot's stack of an item, in slot order.
// Task 23's consumption plan destroys assets per slot (DestroyAsset acts on
// one asset), which is why Snapshot preserves per-slot detail rather than
// only the summed quantity.
type SlotHolding struct {
	Slot     int16
	Quantity uint32
}

// Snapshot is a point-in-time read of a character's EQUIP, USE, and ETC
// compartments (design §4.2.2). atlas-inventory has no batched
// all-compartments endpoint, so a snapshot is built from exactly one read
// per inventory type; every candidate recipe is then evaluated against it
// in memory, satisfying the NFR's "single batched inventory read, not one
// read per recipe".
type Snapshot struct {
	held     map[item.Id]uint32
	slots    map[item.Id][]SlotHolding
	equipped map[item.Id]bool
}

// NewSnapshot reads characterId's EQUIP, USE, and ETC compartments from cp —
// exactly three upstream calls, regardless of how many recipes are then
// evaluated against the result.
func NewSnapshot(cp compartment.Processor, characterId uint32) (Snapshot, error) {
	s := Snapshot{
		held:     map[item.Id]uint32{},
		slots:    map[item.Id][]SlotHolding{},
		equipped: map[item.Id]bool{},
	}
	for _, invType := range []inventory.Type{inventory.TypeValueEquip, inventory.TypeValueUse, inventory.TypeValueETC} {
		m, err := cp.GetByType(characterId, invType)
		if err != nil {
			return Snapshot{}, err
		}
		for _, a := range m.Assets() {
			s.held[a.TemplateId()] += a.Quantity()
			s.slots[a.TemplateId()] = append(s.slots[a.TemplateId()], SlotHolding{Slot: a.Slot(), Quantity: a.Quantity()})
			// A negative slot is a worn equip, not a stored one
			// (services/atlas-inventory/atlas.com/inventory/compartment/processor_accommodation.go
			// freeSlots); only the EQUIP compartment's negative slots count
			// as "equipped" for a recipe's reqEquip.
			if invType == inventory.TypeValueEquip && a.Slot() < 0 {
				s.equipped[a.TemplateId()] = true
			}
		}
	}
	for id, holdings := range s.slots {
		sorted := holdings
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Slot < sorted[j].Slot })
		s.slots[id] = sorted
	}
	return s, nil
}

// Held is the summed quantity of itemId across every slot in the snapshot.
func (s Snapshot) Held(itemId item.Id) uint32 {
	return s.held[itemId]
}

// Slots is every slot holding itemId, ordered by slot ascending. Task 23's
// consumption plan walks this to destroy the exact assets a craft consumes.
func (s Snapshot) Slots(itemId item.Id) []SlotHolding {
	return s.slots[itemId]
}

// Equipped reports whether itemId is currently worn (a negative EQUIP slot),
// for a recipe's reqEquip.
func (s Snapshot) Equipped(itemId item.Id) bool {
	return s.equipped[itemId]
}
