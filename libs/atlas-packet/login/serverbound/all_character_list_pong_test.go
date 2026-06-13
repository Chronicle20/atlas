package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/AllCharacterListPong version=gms_v83 ida=0x5fb0e1
// packet-audit:verify packet=login/serverbound/AllCharacterListPong version=gms_v87 ida=0x632d3a
// packet-audit:verify packet=login/serverbound/AllCharacterListPong version=gms_v95 ida=0x5d44a0
func TestAllCharacterListPongRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListPong{render: true}
			output := AllCharacterListPong{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Render() != input.Render() {
				t.Errorf("render: got %v, want %v", output.Render(), input.Render())
			}
		})
	}
}
