package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{targetId: 12345, mode: 1}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
