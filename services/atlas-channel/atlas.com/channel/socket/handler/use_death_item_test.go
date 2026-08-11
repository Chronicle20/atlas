package handler

import "testing"

// canShowTombEffect is the whole authorisation decision for the USE_DEATHITEM
// relay: the packet is unauthenticated player input carrying free-form
// coordinates that get broadcast to every other client in the map, so a living
// player must not be able to spam tombstones.
func TestCanShowTombEffect(t *testing.T) {
	const wheel = 5510000
	tests := []struct {
		name          string
		hp            uint16
		itemId        uint32
		wheelQuantity uint32
		want          bool
	}{
		{"dead with a wheel", 0, wheel, 1, true},
		{"dead with several charges", 0, wheel, 3, true},
		{"alive with a wheel", 50, wheel, 1, false},
		{"dead without a wheel", 0, wheel, 0, false},
		{"dead but claims another item", 0, 5130000, 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := canShowTombEffect(tc.hp, tc.itemId, tc.wheelQuantity); got != tc.want {
				t.Errorf("canShowTombEffect(%d, %d, %d) = %v, want %v", tc.hp, tc.itemId, tc.wheelQuantity, got, tc.want)
			}
		})
	}
}
