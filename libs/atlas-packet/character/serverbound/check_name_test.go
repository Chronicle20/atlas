package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/CheckName version=gms_v83 ida=0x7d75ab
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v84 ida=0x60cf5d
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v87 ida=0x62f779
// packet-audit:verify packet=character/serverbound/CheckName version=gms_v95 ida=0x5d5690
func TestCheckNameRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CheckName{name: "TestChar"}
			output := CheckName{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
