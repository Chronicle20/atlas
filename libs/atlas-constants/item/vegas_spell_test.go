package item_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestIsVegasSpell(t *testing.T) {
	cases := []struct {
		name string
		id   item.Id
		want bool
	}{
		{"Vega's Spell 10", item.VegasSpell10, true},
		{"Vega's Spell 60", item.VegasSpell60, true},
		{"adjacent cash id", item.Id(5610002), false},
		{"chaos scroll", item.ChaosScrollSixtyPercent, false},
		{"zero", item.Id(0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := item.IsVegasSpell(tc.id); got != tc.want {
				t.Errorf("IsVegasSpell(%d) = %t, want %t", tc.id, got, tc.want)
			}
		})
	}
}

func TestVegasSpellClassification(t *testing.T) {
	if got := item.GetClassification(item.VegasSpell10); got != item.ClassificationVegasSpell {
		t.Errorf("GetClassification(VegasSpell10) = %d, want %d", got, item.ClassificationVegasSpell)
	}
	if got := item.GetClassification(item.VegasSpell60); got != item.ClassificationVegasSpell {
		t.Errorf("GetClassification(VegasSpell60) = %d, want %d", got, item.ClassificationVegasSpell)
	}
}
