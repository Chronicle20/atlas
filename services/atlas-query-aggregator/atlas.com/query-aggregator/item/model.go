package item

import (
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
)

// ItemData represents basic item metadata needed for validation
type ItemData struct {
	id      uint32
	slotMax uint32
}

// Id returns the item ID
func (d ItemData) Id() uint32 {
	return d.id
}

// SlotMax returns the maximum stack size for this item
func (d ItemData) SlotMax() uint32 {
	return d.slotMax
}

// NewItemData creates a new ItemData instance
func NewItemData(id uint32, slotMax uint32) ItemData {
	return ItemData{
		id:      id,
		slotMax: slotMax,
	}
}

// GetDefaultSlotMax returns a reasonable default slotMax based on item type
func GetDefaultSlotMax(itemId uint32) uint32 {
	invType, ok := inventory.TypeFromItemId(item.Id(itemId))
	if !ok {
		return 100 // Conservative default for unknown items
	}

	if invType == inventory.TypeValueEquip {
		return 1 // Equipment never stacks
	}

	return 100 // Conservative default for stackable items
}
