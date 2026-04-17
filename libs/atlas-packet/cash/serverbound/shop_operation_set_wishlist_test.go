package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestShopOperationSetWishlistRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationSetWishlist{serialNumbers: []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
			output := ShopOperationSetWishlist{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.SerialNumbers()) != len(input.SerialNumbers()) {
				t.Fatalf("serialNumbers length: got %v, want %v", len(output.SerialNumbers()), len(input.SerialNumbers()))
			}
			for i, sn := range output.SerialNumbers() {
				if sn != input.SerialNumbers()[i] {
					t.Errorf("serialNumbers[%d]: got %v, want %v", i, sn, input.SerialNumbers()[i])
				}
			}
		})
	}
}
