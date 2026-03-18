package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

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
