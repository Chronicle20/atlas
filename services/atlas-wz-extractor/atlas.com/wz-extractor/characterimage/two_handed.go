package characterimage

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// ResolveStance applies the two-handed override: if any equipped weapon (slot
// -11) is two-handed, the rendered stance becomes "stand2" regardless of the
// requested value. Returns the override flag for observability.
func ResolveStance(requested string, equipment map[int]int) (string, bool) {
	weaponId, ok := equipment[-11]
	if !ok {
		return requested, false
	}
	if !item.IsTwoHanded(item.Id(weaponId)) {
		return requested, false
	}
	if requested == "stand2" {
		return "stand2", false
	}
	return "stand2", true
}
