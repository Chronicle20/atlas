package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v87 ida=0xabb60b
// packet-audit:verify packet=character/serverbound/DistributeAp version=gms_v95 ida=0x9f61c0
func TestDistributeApRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DistributeAp{updateTime: 12345, dwFlag: 64}
			output := DistributeAp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.DwFlag() != input.DwFlag() {
				t.Errorf("dwFlag: got %v, want %v", output.DwFlag(), input.DwFlag())
			}
		})
	}
}
