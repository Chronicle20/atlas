package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v95 ida=0x4837d0
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v87 ida=0x47b5b6
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v83 ida=0x470d7d
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=jms_v185 ida=0x481507
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v84 ida=0x473873
//
// v79 CCashShop::OnSetWish@0x46a6d3: COutPacket(221) Encode1(5)=mode (routed op),
// then a fixed loop of 10 × Encode4 = 10 wishlist serial numbers. Body after the
// mode byte == every later version (10 ints, no count prefix).
// packet-audit:verify packet=cash/serverbound/CashShopOperationSetWishlist version=gms_v79 ida=0x46a6d3
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

// TestShopOperationSetWishlistV79Bytes pins the v79 body: exactly 10 little-endian
// serial numbers, no count prefix (CCashShop::OnSetWish@0x46a6d3 fixed 10-iter loop).
func TestShopOperationSetWishlistV79Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationSetWishlist{serialNumbers: []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	want := "01000000" + "02000000" + "03000000" + "04000000" + "05000000" +
		"06000000" + "07000000" + "08000000" + "09000000" + "0a000000"
	if got != want {
		t.Errorf("v79 bytes: got %s, want %s", got, want)
	}
}
