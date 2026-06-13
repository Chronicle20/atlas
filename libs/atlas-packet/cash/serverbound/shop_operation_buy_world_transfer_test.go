package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyWorldTransfer version=gms_v95 ida=0x482f30
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyWorldTransfer version=gms_v87 ida=0x47df7f
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyWorldTransfer version=gms_v83 ida=0x473601
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyWorldTransfer version=jms_v185 ida=0x485038
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
