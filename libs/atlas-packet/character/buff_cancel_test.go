package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestBuffCancelRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BuffCancel{skillId: 1001003}
			output := BuffCancel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
		})
	}
}
