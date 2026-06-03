package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestOperationPersonalStoreBuyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationPersonalStoreBuy{index: 2, quantity: 10, itemCRC: 0xDEADBEEF}
			output := OperationPersonalStoreBuy{}
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

// TestOperationPersonalStoreBuyBytes pins the wire bytes: byte index, short
// quantity (LE), int itemCRC (LE). CItemInfo::GetItemCRC trailing field is
// present in both GMS v83 (IDA CPersonalShopDlg::BuyItem@0x6fd261 Encode4
// ItemCRC) and v95 (@0x69a7f0), so it is unconditional.
func TestOperationPersonalStoreBuyBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationPersonalStoreBuy{index: 2, quantity: 10, itemCRC: 0xDEADBEEF}
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	// 02 | 0a00 | efbeadde
	want := "020a00efbeadde"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}
