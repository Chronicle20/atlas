package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=gms_v95 ida=0x48ed40
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=gms_v87 ida=0x4786cc
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=gms_v83 ida=0x46e121
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=jms_v185 ida=0x47f01d
func TestShopOperationBuyPackageRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyPackage{pointType: true, option: 1, serialNumber: 12345}
			output := ShopOperationBuyPackage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PointType() != input.PointType() {
				t.Errorf("pointType: got %v, want %v", output.PointType(), input.PointType())
			}
			if output.Option() != input.Option() {
				t.Errorf("option: got %v, want %v", output.Option(), input.Option())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}
