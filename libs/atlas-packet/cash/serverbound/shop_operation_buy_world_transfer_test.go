package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopOperationBuyWorldTransferRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyWorldTransfer{serialNumber: 12345, targetWorld: 2}
			output := ShopOperationBuyWorldTransfer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.TargetWorld() != input.TargetWorld() {
				t.Errorf("targetWorld: got %v, want %v", output.TargetWorld(), input.TargetWorld())
			}
		})
	}
}
