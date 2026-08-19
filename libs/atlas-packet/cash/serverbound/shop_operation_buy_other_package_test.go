package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyOtherPackage version=gms_v95 ida=0x4907b0
func TestShopOperationBuyOtherPackageRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyOtherPackage{spw: "secret", serialNumber: 9100000, name: "Recipient", message: "Enjoy!"}
			output := ShopOperationBuyOtherPackage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SPW() != input.SPW() {
				t.Errorf("spw: got %v, want %v", output.SPW(), input.SPW())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
