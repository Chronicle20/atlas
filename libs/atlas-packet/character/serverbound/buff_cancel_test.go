package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v83 ida=0x96d873
// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v87 ida=0x9f22b8
// packet-audit:verify packet=character/serverbound/BuffCancelRequest version=gms_v95 ida=0x93d730
func TestBuffCancelRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BuffCancelRequest{skillId: 1001003}
			output := BuffCancelRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
		})
	}
}
