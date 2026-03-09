package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestSetMemberTitleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetMemberTitle{targetId: 54321, newTitle: 2}
			output := SetMemberTitle{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.NewTitle() != input.NewTitle() {
				t.Errorf("newTitle: got %v, want %v", output.NewTitle(), input.NewTitle())
			}
		})
	}
}
