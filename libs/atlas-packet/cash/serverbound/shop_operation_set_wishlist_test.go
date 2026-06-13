package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v95 ida=0x4837d0
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v87 ida=0x47b5b6
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v83 ida=0x470d7d
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
