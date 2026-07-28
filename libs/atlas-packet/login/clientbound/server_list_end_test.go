package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61: WORLD_INFORMATION handler sub_56663F @0x56663f (GMS_v61.1_U_DEVM.exe,
// port 13338) reads (char)Decode1(worldId)@0x566660; when < 0 (0xFF = -1) it takes
// the else branch (end-of-world-list → DrawWorldItems) instead of decoding a world
// record. atlas ServerListEnd.Encode writes the single 0xFF terminator byte.
//
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v61 ida=0x56663f
func TestServerListEndV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := ServerListEnd{}
	if got := pt.Encode(t, ctx, input.Encode, nil); !bytes.Equal(got, []byte{0xFF}) {
		t.Errorf("v61 ServerListEnd body: got % x, want ff", got)
	}
}

// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v83 ida=0x5f95b7
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v87 ida=0x630e7c
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v95 ida=0x5da7f0
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v84 ida=0x60e5b3
// packet-audit:verify packet=login/clientbound/ServerListEnd version=jms_v185 ida=0x66f107
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v79 ida=0x5ce269
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v72 ida=0x5b33f8
func TestServerListEndRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerListEnd{}
			output := ServerListEnd{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
