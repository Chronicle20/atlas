package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v83 ida=0x94e45f
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v87 ida=0x9c9270
// packet-audit:verify packet=character/serverbound/ChairFixed version=gms_v95 ida=0x90f6d0
func TestChairFixedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChairFixed{chairId: 42}
			output := ChairFixed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChairId() != input.ChairId() {
				t.Errorf("chairId: got %v, want %v", output.ChairId(), input.ChairId())
			}
		})
	}
}
