package consumable

import (
	"testing"

	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
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
