package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=gms_v87 ida=0x74076b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=jms_v185 ida=0x762365
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=gms_v84 ida=0x71951e
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

// TestOperationMerchantBuyBytes pins the wire bytes: byte index, short quantity
// (LE), and (v83+) int itemCRC (LE). The entrusted-merchant buy shares
// CPersonalShopDlg::BuyItem (op 0x22 vs 0x17) and carries the trailing
// CItemInfo::GetItemCRC int in v83 (@0x6fd261) and v95 (@0x69a7f0), but it is
// ABSENT in GMS v79 (CPersonalShopDlg::BuyItem@0x689ce7) — hence the
// tradeCrcPresent gate.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=gms_v83 ida=0x6fd261
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=gms_v95 ida=0x69a7f0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantBuy version=gms_v79 ida=0x689ce7
func TestOperationMerchantBuyBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationMerchantBuy{index: 3, quantity: 25, itemCRC: 0x12345678}

	// v83: index | quantity(LE) | itemCRC(LE)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	if got83 != "03190078563412" {
		t.Errorf("v83 bytes: got %s, want 03190078563412", got83)
	}

	// v79: index | quantity(LE) only — no trailing itemCRC
	got79 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got79 != "031900" {
		t.Errorf("v79 bytes: got %s, want 031900", got79)
	}
}
