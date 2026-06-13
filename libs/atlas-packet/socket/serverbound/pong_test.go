package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=socket/serverbound/Pong version=gms_v83 ida=0x4966c0
// packet-audit:verify packet=socket/serverbound/Pong version=gms_v87 ida=0x4a870a
// packet-audit:verify packet=socket/serverbound/Pong version=gms_v95 ida=0x4afc90
// packet-audit:verify packet=socket/serverbound/Pong version=jms_v185 ida=0x4b18e3
func TestPongRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Pong{}
			output := Pong{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
