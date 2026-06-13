package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/SelectWorld version=gms_v83 ida=0x5f82f4
// packet-audit:verify packet=login/clientbound/SelectWorld version=gms_v84 ida=0x60d26e
// packet-audit:verify packet=login/clientbound/SelectWorld version=gms_v87 ida=0x62fa8a
// packet-audit:verify packet=login/clientbound/SelectWorld version=gms_v95 ida=0x5d2200
func TestSelectWorldRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SelectWorld{worldId: 5}
			output := SelectWorld{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
		})
	}
}
