package compartment

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// AssetModel is the subset of atlas-inventory's asset resource this service
// consumes: the template item id, stacked quantity, and slot, used to
// evaluate whether a character holds enough of a recipe's ingredients and,
// per-slot, which stacks a craft would consume (Task 21/23).
type AssetModel struct {
	templateId item.Id
	quantity   uint32
	slot       int16
}

// NewAssetModel builds an AssetModel directly, for tests and other code
// outside this package that needs to construct a compartment snapshot
// without a round-trip through atlas-inventory's wire format.
func NewAssetModel(templateId item.Id, quantity uint32, slot int16) AssetModel {
	return AssetModel{templateId: templateId, quantity: quantity, slot: slot}
}

func (m AssetModel) TemplateId() item.Id {
	return m.templateId
}

func (m AssetModel) Quantity() uint32 {
	return m.quantity
}

// Slot is the compartment slot this stack occupies, as atlas-inventory
// reports it (services/atlas-inventory/atlas.com/inventory/asset/rest.go).
func (m AssetModel) Slot() int16 {
	return m.slot
}

// Model is one inventory compartment snapshot (one of EQUIP/USE/ETC), as
// read from atlas-inventory.
type Model struct {
	id            uuid.UUID
	inventoryType inventory.Type
	capacity      uint32
	assets        []AssetModel
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) Type() inventory.Type {
	return m.inventoryType
}

func (m Model) Capacity() uint32 {
	return m.capacity
}

func (m Model) Assets() []AssetModel {
	return m.assets
}

// QuantityOf sums the quantity of every asset in the compartment matching
// itemId, for checking whether a recipe's material requirement is met.
func (m Model) QuantityOf(itemId item.Id) uint32 {
	var total uint32
	for _, a := range m.assets {
		if a.TemplateId() == itemId {
			total += a.Quantity()
		}
	}
	return total
}

// AccommodationItem is one (itemId, quantity) entry of a CanAccommodate
// request — a recipe award atlas-maker asks atlas-inventory whether it would
// currently accept.
type AccommodationItem struct {
	ItemId   item.Id
	Quantity uint32
}
