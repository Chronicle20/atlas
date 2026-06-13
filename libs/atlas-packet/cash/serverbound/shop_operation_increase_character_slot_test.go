package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
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
			if output.Currency() != input.Currency() {
				t.Errorf("currency: got %v, want %v", output.Currency(), input.Currency())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}
