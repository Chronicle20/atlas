package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

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

// TestShopOperationBuyOtherPackageV95Bytes pins the field order behind the
// round-trip test above -- derivation.md D3a (§4, CCashShop::OnGiftPackage
// @ 0x4907b0, the same address the round-trip test's own
// packet-audit:verify marker above already cites): the body after the mode
// byte is spw (asciiString), serialNumber (uint32 LE), name (asciiString),
// message (asciiString) -- no pointType, no option. A round-trip test alone
// cannot see a self-consistent field-order defect (e.g. name/message
// swapped identically in both Encode and Decode); this asserts the literal
// wire bytes against that fixed order instead.
func TestShopOperationBuyOtherPackageV95Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ShopOperationBuyOtherPackage{spw: "ABCD", serialNumber: 0x05060708, name: "Bob", message: "Hi"}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	// spw "ABCD": uint16 LE length(4) + 4 ASCII bytes
	// serialNumber 0x05060708: uint32 LE
	// name "Bob": uint16 LE length(3) + 3 ASCII bytes
	// message "Hi": uint16 LE length(2) + 2 ASCII bytes
	want := "040041424344" + "08070605" + "0300426f62" + "02004869"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}
