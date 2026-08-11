package consumable

import (
	consumable3 "atlas-consumables/data/consumable"
	"testing"
)

// TestValidateCatchItem is the pre-reserve gate: only class-227 items with a
// non-zero create id may proceed. Everything else is rejected before the
// inventory is touched (FR-3.2).
func TestValidateCatchItem(t *testing.T) {
	cases := []struct {
		name   string
		itemId uint32
		create uint32
		wantOk bool
	}{
		{"a catch item with a reward", 2270000, 1902000, true},
		{"a catch item with no create id", 2270000, 0, false},
		{"a red potion", 2000000, 1902000, false},
		{"a revitalizer", 2260000, 1902000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ci, _ := consumable3.Extract(consumable3.RestModel{Id: tc.itemId, Create: tc.create})
			if got := validateCatchItem(tc.itemId, ci); got != tc.wantOk {
				t.Fatalf("validateCatchItem = %t, want %t", got, tc.wantOk)
			}
		})
	}
}
