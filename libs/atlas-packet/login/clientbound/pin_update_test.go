package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/PinUpdate version=gms_v83 ida=0x5fcbc1
// packet-audit:verify packet=login/clientbound/PinUpdate version=gms_v84 ida=0x611c99
// packet-audit:verify packet=login/clientbound/PinUpdate version=gms_v87 ida=0x6345d4
// packet-audit:verify packet=login/clientbound/PinUpdate version=gms_v95 ida=0x5d2420
func TestPinUpdateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := PinUpdate{mode: 1}
			output := PinUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
