package merchant

import "testing"

// StoreSkinSpec maps a personal-store permit item id to the client's PSSkin
// index (the balloon's nSpec byte). Verified against GMS v83.1 WZ:
// UI/ChatBalloon.img/miniroom/PSSkin has canvases 0,1,2,3,4,6 — an exact 1:1
// match with the store permits 5140000/1/2/3/4/6 (both skip 5), and
// CChatBalloon::MakeMiniRoomBalloon (v95 @0x4a2d90) formats "PSSkin/%d" with
// nSpec for the personal-shop (type 4) case.
func TestStoreSkinSpec(t *testing.T) {
	cases := []struct {
		name     string
		permitId uint32
		want     byte
	}{
		{"default plain sign", 5140000, 0},
		{"sky blue tree", 5140001, 1},
		{"skin 2", 5140002, 2},
		{"skin 6 (5 is skipped)", 5140006, 6},
		{"below base clamps to 0", 0, 0},
		{"non-permit clamps to 0", 2060000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StoreSkinSpec(tc.permitId); got != tc.want {
				t.Fatalf("StoreSkinSpec(%d) = %d, want %d", tc.permitId, got, tc.want)
			}
		})
	}
}
