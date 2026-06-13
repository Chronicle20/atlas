package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v83 ida=0xa0f9e2
// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v87 ida=0xa9ef5b
// packet-audit:verify packet=character/serverbound/ChairPortable version=gms_v95 ida=0x9da100
func TestChairPortableRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChairPortable{itemId: 3010000}
			output := ChairPortable{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
