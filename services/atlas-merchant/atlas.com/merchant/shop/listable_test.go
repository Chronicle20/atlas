package shop

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
)

// IsListableItem gates what can be put up for sale: pets and cash items are
// never listable, nor is any item instance flagged untradeable.
func TestIsListableItem(t *testing.T) {
	cases := []struct {
		name   string
		itemId uint32
		flag   uint16
		want   error
	}{
		{"normal use item", 2000000, 0, nil},
		{"normal equip", 1302000, 0, nil},
		{"etc item", 4000000, 0, nil},
		{"pet", 5000000, 0, ErrPetItem},
		{"cash item (store permit)", 5140000, 0, ErrCashItem},
		{"cash item (owl)", 5230000, 0, ErrCashItem},
		{"untradeable flagged use item", 2000000, uint16(asset.FlagUntradeable), ErrUntradeableItem},
		{"untradeable flagged equip", 1302000, uint16(asset.FlagUntradeable), ErrUntradeableItem},
		// Pet beats the untradeable flag in check order; either error is a
		// rejection, but the pin documents the precedence.
		{"untradeable pet still ErrPetItem", 5000000, uint16(asset.FlagUntradeable), ErrPetItem},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := IsListableItem(tc.itemId, tc.flag)
			if tc.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.want)
			}
		})
	}
}
