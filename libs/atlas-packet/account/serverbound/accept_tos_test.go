package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=account/serverbound/AcceptTos version=gms_v87 ida=0x633e1a
// packet-audit:verify packet=account/serverbound/AcceptTos version=gms_v95 ida=0x5d4540
func TestAcceptTosRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AcceptTos{accepted: true}
			output := AcceptTos{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Accepted() != input.Accepted() {
				t.Errorf("accepted: got %v, want %v", output.Accepted(), input.Accepted())
			}
		})
	}
}
