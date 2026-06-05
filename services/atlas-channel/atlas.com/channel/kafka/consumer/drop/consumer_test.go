package drop

import "testing"

// TestIsConsumedOnPickupCard locks the classification that drives the
// monster-book-card pickup path: cards (item classification 238, e.g. 2380000)
// are consumed on pickup, so the handler suppresses the generic pickup message
// and sends an action-unlock; every other item keeps the normal pickup message.
func TestIsConsumedOnPickupCard(t *testing.T) {
	cases := []struct {
		name   string
		itemId uint32
		want   bool
	}{
		{name: "first monster-book card", itemId: 2380000, want: true},
		{name: "another monster-book card", itemId: 2380001, want: true},
		{name: "high monster-book card", itemId: 2389999, want: true},
		{name: "use-tab potion is not a card", itemId: 2000000, want: false},
		{name: "etc item is not a card", itemId: 4000000, want: false},
		{name: "equip is not a card", itemId: 1302000, want: false},
		{name: "zero (no item / meso pickup) is not a card", itemId: 0, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isConsumedOnPickupCard(tc.itemId); got != tc.want {
				t.Errorf("isConsumedOnPickupCard(%d) = %v, want %v", tc.itemId, got, tc.want)
			}
		})
	}
}
