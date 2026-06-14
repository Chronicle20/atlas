package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=ui/clientbound/Disable version=gms_v87 ida=0x9e3172
// packet-audit:verify packet=ui/clientbound/Disable version=gms_v95 ida=0x905550
// packet-audit:verify packet=ui/clientbound/Disable version=jms_v185 ida=0xa2cdcb
// packet-audit:verify packet=ui/clientbound/Disable version=gms_v84 ida=0x99ed5a
func TestUiDisable(t *testing.T) {
	input := NewUiDisable(true)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestUiDisableWireShape proves the exact wire layout matches what
// CUserLocal::OnSetStandAloneMode (GMS v95 @ 0x905550) reads:
//
//	Decode1 (standalone/enable flag) = 1 byte.
//
// All versions share the same single-byte layout.
//
// Read-order source for the gms_v83 marker below: the checked-in IDA export
// docs/packets/ida-exports/gms_v83.json, entry
// CUserLocal::OnSetStandAloneMode @ 0x95ffa2, whose single ordered call is
// Decode1 "bStandAlone (enable flag; stored to CWvsContext+0x70)".
//
// packet-audit:verify packet=ui/clientbound/Disable version=gms_v83 ida=0x95ffa2
func TestUiDisableWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			// enable=true → 0x01
			bTrue := NewUiDisable(true).Encode(l, ctx)(nil)
			if len(bTrue) != 1 {
				t.Errorf("enable=true wire size = %d bytes, want 1: % x", len(bTrue), bTrue)
			}
			if bTrue[0] != 0x01 {
				t.Errorf("enable=true byte[0] = 0x%02x, want 0x01", bTrue[0])
			}

			// enable=false → 0x00
			bFalse := NewUiDisable(false).Encode(l, ctx)(nil)
			if len(bFalse) != 1 {
				t.Errorf("enable=false wire size = %d bytes, want 1: % x", len(bFalse), bFalse)
			}
			if bFalse[0] != 0x00 {
				t.Errorf("enable=false byte[0] = 0x%02x, want 0x00", bFalse[0])
			}
		})
	}
}
