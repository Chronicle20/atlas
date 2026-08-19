package handler

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
)

// TestCashShopWishListBodyEncodesStoredSerials pins the wire layout
// CashShopWishListUpdateBody produces for the APPLY_WISHLIST answer
// (derivation.md D2b: RESOLVED but flagged INFERENTIAL — the UPDATE_WISHLIST
// arm, mode 98, not LOAD_WISHLIST). The mode byte is followed by exactly 10
// little-endian uint32 slots (WishListUpdate.Encode's fixed-width pad),
// unused slots zero-filled — never a variable-length or empty payload.
func TestCashShopWishListBodyEncodesStoredSerials(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationUpdateWishlist: float64(98),
		},
	}

	tests := []struct {
		name string
		sns  []uint32
	}{
		{"full wishlist", []uint32{10000, 20000, 30000}},
		{"empty wishlist", []uint32{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := cashcb.CashShopWishListUpdateBody(tt.sns)(logrus.New(), context.Background())(options)

			want := make([]byte, 1+10*4)
			want[0] = 98
			for i := 0; i < 10; i++ {
				var v uint32
				if i < len(tt.sns) {
					v = tt.sns[i]
				}
				binary.LittleEndian.PutUint32(want[1+i*4:1+i*4+4], v)
			}

			if len(body) != len(want) {
				t.Fatalf("body length = %d, want %d (mode + 10 fixed uint32 slots)", len(body), len(want))
			}
			if body[0] != 98 {
				t.Fatalf("mode byte = %d, want 98", body[0])
			}
			for i := range want {
				if body[i] != want[i] {
					t.Fatalf("body = %#v, want %#v", body, want)
				}
			}
		})
	}
}
