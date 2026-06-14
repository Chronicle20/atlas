package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=ui/clientbound/Open version=gms_v83 ida=0x9600f0
// packet-audit:verify packet=ui/clientbound/Open version=gms_v87 ida=0x9e32c0
// packet-audit:verify packet=ui/clientbound/Open version=gms_v95 ida=0x9055f0
// packet-audit:verify packet=ui/clientbound/Open version=jms_v185 ida=0xa2cf38
// packet-audit:verify packet=ui/clientbound/Open version=gms_v84 ida=0x99eea8
func TestUiOpen(t *testing.T) {
	input := NewUiOpen(5)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestUiOpenWireShape proves the exact wire layout matches what
// CUserLocal::OnOpenUI (GMS v95 @ 0x9055f0) reads:
//
//	Decode1 (windowMode) = 1 byte.
//
// All versions share the same single-byte layout.
func TestUiOpenWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewUiOpen(7)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 1 {
				t.Errorf("wire size = %d bytes, want 1: % x", len(b), b)
			}
			if b[0] != 7 {
				t.Errorf("byte[0] = 0x%02x, want 0x07 (windowMode)", b[0])
			}
		})
	}
}
