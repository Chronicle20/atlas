package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestOperationMerchantBuyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMerchantBuy{index: 3, quantity: 25, itemCRC: 0x12345678}
			output := OperationMerchantBuy{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Index() != input.Index() {
				t.Errorf("index: got %v, want %v", output.Index(), input.Index())
			}
			if output.Quantity() != input.Quantity() {
				t.Errorf("quantity: got %v, want %v", output.Quantity(), input.Quantity())
			}
			if output.ItemCRC() != input.ItemCRC() {
				t.Errorf("itemCRC: got %v, want %v", output.ItemCRC(), input.ItemCRC())
			}
		})
	}
}

// TestOperationMerchantBuyBytes pins the wire bytes: byte index, short quantity
// (LE), int itemCRC (LE). The entrusted-merchant buy shares
// CPersonalShopDlg::BuyItem (op 0x22 vs 0x17) and carries the same trailing
// CItemInfo::GetItemCRC int in both v83 (@0x6fd261) and v95 (@0x69a7f0).
func TestOperationMerchantBuyBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationMerchantBuy{index: 3, quantity: 25, itemCRC: 0x12345678}
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	// 03 | 1900 | 78563412
	want := "03190078563412"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}
