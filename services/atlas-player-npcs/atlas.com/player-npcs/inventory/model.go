// Package inventory reads a character's inventory from atlas-inventory —
// the actual upstream for equipped items (design §6.1 correction; see the
// Task 13/14 fix round brief). atlas-character does not serve equipment.
package inventory

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// Asset is a single item occupying a slot in a compartment, narrowed to
// the fields a deploy snapshot needs (slot/templateId — mirrors
// services/atlas-channel/atlas.com/channel/asset's slot+templateId
// shape). Callers that need the FR-5.2 signed slot filtering (1-11,
// 101-111) apply it themselves, same as character.EquippedItem did.
type Asset struct {
	slot       int16
	templateId uint32
}

func (a Asset) Slot() int16        { return a.slot }
func (a Asset) TemplateId() uint32 { return a.templateId }

// Model is a character's inventory: every compartment, keyed by type.
type Model struct {
	characterId  uint32
	compartments map[inventory.Type][]Asset
}

func (m Model) CharacterId() uint32 { return m.characterId }

// Equipment returns the equip compartment's (inventory.TypeValueEquip)
// assets — the raw slot/templateId pairs a deploy snapshot masks and
// filters (design §6.1, FR-5.2).
func (m Model) Equipment() []Asset {
	return m.compartments[inventory.TypeValueEquip]
}
