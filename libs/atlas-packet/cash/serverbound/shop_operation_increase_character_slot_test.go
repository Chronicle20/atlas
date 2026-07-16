package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseCharacterSlot version=gms_v95 ida=0x48dec0
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseCharacterSlot version=gms_v87 ida=0x47657d
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseCharacterSlot version=gms_v83 ida=0x46c6f8
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseCharacterSlot version=jms_v185 ida=0x47c8ce
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseCharacterSlot version=gms_v84 ida=0x46edcd
func TestShopOperationIncreaseCharacterSlotRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationIncreaseCharacterSlot{isPoints: true, currency: 1, serialNumber: 12345}
			output := ShopOperationIncreaseCharacterSlot{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.IsPoints() != input.IsPoints() {
				t.Errorf("isPoints: got %v, want %v", output.IsPoints(), input.IsPoints())
			}
			// Legacy GMS (<83, e.g. v79) omits the currency int on the wire; it
			// decodes back as zero.
			if v.Region == "GMS" && v.MajorVersion < 83 {
				if output.Currency() != 0 {
					t.Errorf("currency: got %v, want 0 for %s", output.Currency(), v.Name)
				}
			} else if output.Currency() != input.Currency() {
				t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}

// TestShopOperationIncreaseCharacterSlotV79Bytes pins the v79 body. IDA v79
// CCashShop::OnIncCharacterSlotCount@0x4673be: COutPacket(221) Encode1(9)=mode
// (routed op) then Encode1(v30==2)=isPoints, Encode4(a2)=serialNumber. Body after
// the mode byte is isPoints(1)+serialNumber(4); no currency int.
// packet-audit:verify packet=cash/serverbound/CashShopOperationIncreaseCharacterSlot version=gms_v79 ida=0x4673be
func TestShopOperationIncreaseCharacterSlotV79Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationIncreaseCharacterSlot{isPoints: true, currency: 0x01020304, serialNumber: 0x05060708}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got != "01"+"08070605" {
		t.Errorf("v79 bytes: got %s, want 0108070605", got)
	}
}
