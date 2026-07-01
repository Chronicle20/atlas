package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=gms_v87 ida=0x74076b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=gms_v95 ida=0x69a7f0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=jms_v185 ida=0x762365
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=gms_v84 ida=0x71951e
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
			crcPresent := (v.Region == "GMS" && v.MajorVersion >= 83) || v.Region != "GMS"
			if crcPresent {
				if output.ItemCRC() != input.ItemCRC() {
					t.Errorf("itemCRC: got %v, want %v", output.ItemCRC(), input.ItemCRC())
				}
			} else if output.ItemCRC() != 0 {
				t.Errorf("itemCRC should be absent (0) for %s, got %v", v.Name, output.ItemCRC())
			}
		})
	}
}

// TestOperationPersonalStoreBuyBytes pins the wire bytes: byte index, short
// quantity (LE), and (v83+) int itemCRC (LE). The CItemInfo::GetItemCRC
// trailing field is present in GMS v83 (IDA CPersonalShopDlg::BuyItem@0x6fd261
// Encode4 ItemCRC) and v95 (@0x69a7f0) but ABSENT in GMS v79
// (CPersonalShopDlg::BuyItem@0x689ce7 sends only Encode1(mode),Encode1(index),
// Encode2(quantity)) — hence the tradeCrcPresent gate.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=gms_v83 ida=0x6fd261
// packet-audit:verify packet=interaction/serverbound/InteractionOperationPersonalStoreBuy version=gms_v79 ida=0x689ce7
func TestOperationPersonalStoreBuyBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationPersonalStoreBuy{index: 2, quantity: 10, itemCRC: 0xDEADBEEF}

	// v83: index | quantity(LE) | itemCRC(LE)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83 != "020a00efbeadde" {
		t.Errorf("v83 bytes: got %s, want 020a00efbeadde", got83)
	}

	// v79: index | quantity(LE) only — no trailing itemCRC
	got79 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got79 != "020a00" {
		t.Errorf("v79 bytes: got %s, want 020a00", got79)
	}
}
