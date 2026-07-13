package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=merchant/clientbound/ShopLinkResult version=gms_v83 ida=0x8a4e7a
// packet-audit:verify packet=merchant/clientbound/ShopLinkResult version=gms_v95 ida=0x847d60
// packet-audit:verify packet=merchant/clientbound/ShopLinkResult version=gms_v79 ida=0x973035
// packet-audit:verify packet=merchant/clientbound/ShopLinkResult version=gms_v72 ida=0x921189
// packet-audit:verify packet=merchant/clientbound/ShopLinkResult version=gms_v61 ida=0x849af0
func TestShopLinkResultRoundTrip(t *testing.T) {
	input := NewShopLinkResult(18)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &ShopLinkResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != 18 {
				t.Errorf("code = %d, want 18", output.Code())
			}
		})
	}
}

// TestShopLinkResultWireShape pins the single-code-byte body. Code set is
// identical in v83 (0x8a4e7a) and v95 (0x847d60) — task-127 design §1.5.
func TestShopLinkResultWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, code := range []byte{0, 1, 2, 3, 4, 7, 17, 18, 23} {
		in := NewShopLinkResult(code)
		b := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
		if len(b) != 1 || b[0] != code {
			t.Errorf("code %d: wire = % x, want single byte 0x%02x", code, b, code)
		}
	}
}
