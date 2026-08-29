package compartment

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// AssetModel is the subset of atlas-inventory's asset resource this service
// consumes: the template item id and stacked quantity, used to evaluate
// whether a character holds enough of a recipe's ingredients.
type AssetModel struct {
	templateId item.Id
	quantity   uint32
}

func (m AssetModel) TemplateId() item.Id {
	return m.templateId
}

func (m AssetModel) Quantity() uint32 {
	return m.quantity
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
