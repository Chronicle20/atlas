package serverbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=merchant/serverbound/OwlAction version=gms_v83 ida=0x8a0e9a
// packet-audit:verify packet=merchant/serverbound/OwlAction version=gms_v95 ida=0x848b90
func TestOwlActionRoundTrip(t *testing.T) {
	input := NewOwlAction(5)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &OwlAction{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != 5 {
				t.Errorf("mode = %d, want 5", output.Mode())
			}
		})
	}
}

// TestOwlActionWireShape pins the exact layout: a single mode byte.
// CUIShopScanner::OnCreate builds [opcode][byte 5] (v83 0x8a0e9a, v95 0x848b90).
func TestOwlActionWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewOwlAction(5)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 1 {
				t.Fatalf("wire size = %d bytes, want 1: % x", len(b), b)
			}
			if b[0] != 0x05 {
				t.Errorf("byte[0] mode = 0x%02x, want 0x05", b[0])
			}
		})
	}
}
