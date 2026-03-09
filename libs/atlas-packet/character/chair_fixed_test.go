package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

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
