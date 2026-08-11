package surprise

import "testing"

func TestHasRoomForSwap(t *testing.T) {
	tests := []struct {
		name        string
		assetCount  uint32
		capacity    uint32
		boxQuantity uint32
		want        bool
	}{
		// Stack of boxes: the box row survives the consume, so the reward
		// genuinely needs a spare slot.
		{"stacked box, spare slot", 54, 55, 3, true},
		{"stacked box, locker exactly full", 55, 55, 3, false},
		{"stacked box, one short of full", 54, 55, 2, true},

		// Last box: the row is released, so the grant is slot-neutral and a
		// completely full locker is still fine.
		{"last box, locker exactly full", 55, 55, 1, true},
		{"last box, spare slot", 40, 55, 1, true},

		// Over-capacity lockers (data drift) must still be rejected for the
		// stacked case and permitted for the neutral case only at equality.
		{"over capacity, stacked box", 56, 55, 2, false},
		{"over capacity, last box", 56, 55, 1, false},
	}
	for _, tt := range tests {
		if got := HasRoomForSwap(tt.assetCount, tt.capacity, tt.boxQuantity); got != tt.want {
			t.Errorf("%s: HasRoomForSwap(%d, %d, %d) = %v, want %v",
				tt.name, tt.assetCount, tt.capacity, tt.boxQuantity, got, tt.want)
		}
	}
}
