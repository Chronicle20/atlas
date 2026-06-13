package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v83 ida=0xa23cf3
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v87 ida=0xabb7c1
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v95 ida=0x9f2e90
// packet-audit:verify packet=character/serverbound/DistributeSp version=gms_v84 ida=0xa6f390
func TestDistributeSpRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DistributeSp{updateTime: 12345, skillId: 1001004}
			output := DistributeSp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
		})
	}
}
