package validation

import (
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/item"
	"math"

	"github.com/Chronicle20/atlas-constants/inventory"
	atlasItem "github.com/Chronicle20/atlas-constants/item"
)

// CalculateInventorySpace determines if a character can hold X quantity of an item
// Returns:
//   - canHold: true if character has enough space
//   - freeSlots: number of slots available after adding the items
func CalculateInventorySpace(char character.Model, itemId uint32, quantity uint32, itemProcessor item.Processor) (bool, int) {
	// Handle edge case: quantity 0 always passes
	if quantity == 0 {
		return true, 0
	}

	// 1. Get item type and slotMax
	invType, ok := inventory.TypeFromItemId(atlasItem.Id(itemId))
	if !ok {
		// Invalid item ID, use default
		invType = inventory.TypeValueETC
	}

	slotMax := itemProcessor.GetSlotMax(itemId)
	if slotMax == 0 {
		slotMax = 1 // Safety fallback
	}

	// 2. Get compartment
	compartment := char.Inventory().CompartmentByType(invType)

	// 3. Find existing stacks and calculate remaining after filling
	remaining := quantity
	for _, asset := range compartment.Assets() {
		if asset.TemplateId() == itemId {
			currentQty := asset.Quantity()
			spaceInStack := slotMax - currentQty

			if spaceInStack > 0 {
				// Can fill some or all of this stack
				fillAmount := min(spaceInStack, remaining)
				remaining -= fillAmount

				if remaining == 0 {
					// All quantity fits in existing stacks
					break
				}
			}
		}
	}

	// 4. Calculate new slots needed for remaining quantity
	newSlotsNeeded := 0
	if remaining > 0 {
		newSlotsNeeded = int(math.Ceil(float64(remaining) / float64(slotMax)))
	}

	// 5. Check available capacity
	currentUsed := len(compartment.Assets())
	totalCapacity := int(compartment.Capacity())
	freeSlots := totalCapacity - currentUsed

	// 6. Determine if there's enough space
	canHold := freeSlots >= newSlotsNeeded
	slotsAfter := freeSlots - newSlotsNeeded

	return canHold, slotsAfter
}

// min returns the minimum of two uint32 values
func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
