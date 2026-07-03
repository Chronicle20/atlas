package consumable

import (
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// vegaRates returns the natural scroll success rate a Vega's Spell requires
// (exact match only) and the boosted rate it applies. This is server policy
// (PRD FR-4.1), not WZ data — the Item.wz entries for 0561 carry only info
// nodes. Non-vega ids return ok=false.
func vegaRates(id item2.Id) (required uint32, boosted uint32, ok bool) {
	switch id {
	case item2.VegasSpell10:
		return 10, 30, true
	case item2.VegasSpell60:
		return 60, 90, true
	}
	return 0, 0, false
}
