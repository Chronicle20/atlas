package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v83 ida=0x94fa8e
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v87 ida=0x9c9270
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v95 ida=0x933920
// packet-audit:verify packet=character/serverbound/ChalkboardClose version=gms_v84 ida=0x987824
func TestChalkboardCloseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChalkboardClose{}
			output := ChalkboardClose{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
