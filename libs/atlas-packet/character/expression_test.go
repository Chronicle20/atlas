package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestExpressionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Expression{emote: 42}
			output := Expression{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Emote() != input.Emote() {
				t.Errorf("emote: got %v, want %v", output.Emote(), input.Emote())
			}
		})
	}
}
