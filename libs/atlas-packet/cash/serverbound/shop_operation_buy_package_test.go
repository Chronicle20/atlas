package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
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
			// Legacy GMS (<83, e.g. v79) carries only serialNumber; pointType/option
			// are not on the wire and decode back as zero.
			if v.Region == "GMS" && v.MajorVersion < 83 {
				if output.PointType() != false {
					t.Errorf("pointType: got %v, want false for %s", output.PointType(), v.Name)
				}
				if output.Option() != 0 {
					t.Errorf("option: got %v, want 0 for %s", output.Option(), v.Name)
				}
			} else {
				if output.PointType() != input.PointType() {
					t.Errorf("pointType: got %v, want %v", output.PointType(), input.PointType())
				}
				if output.Option() != input.Option() {
					t.Errorf("option: got %v, want %v", output.Option(), input.Option())
				}
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}

// TestShopOperationBuyPackageV79Bytes pins the v79 body. IDA v79
// CCashShop::OnBuyPackage@0x468a40: COutPacket(221) Encode1(0x20)=mode (routed as
// the op byte, not body) then Encode4(a2)=serialNumber. The body after the mode
// byte is exactly serialNumber(4) — no pointType, no option.
// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=gms_v79 ida=0x468a40
func TestShopOperationBuyPackageV79Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyPackage{pointType: true, option: 1, serialNumber: 0x05060708}
	got72 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	// v72 body == v79: serialNumber(4) only. IDA v72 CCashShop::OnBuyPackage@0x4678da
	// COutPacket(219) Encode1(0x1F)=mode @0x467ac9, Encode4(a2)=serialNumber @0x467ad4.
	// packet-audit:verify packet=cash/serverbound/CashShopOperationBuyPackage version=gms_v72 ida=0x4678da
	if got72 != "08070605" {
		t.Errorf("v72 bytes: got %s, want 08070605", got72)
	}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got != "08070605" {
		t.Errorf("v79 bytes: got %s, want 08070605", got)
	}
}
