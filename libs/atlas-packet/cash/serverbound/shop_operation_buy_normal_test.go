package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNormal version=gms_v95 ida=0x48f580
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNormal version=jms_v185 ida=0x47f5ba
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNormal version=gms_v87 ida=0x478cdd
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyNormal version=gms_v83 ida=0x46e5c5
func TestShopOperationBuyNormalRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyNormal{serialNumber: 12345}
			output := ShopOperationBuyNormal{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}
