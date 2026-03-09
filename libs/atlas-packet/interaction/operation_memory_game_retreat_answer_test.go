package interaction

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationMemoryGameRetreatAnswerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameRetreatAnswer{response: true}
			output := OperationMemoryGameRetreatAnswer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Response() != input.Response() {
				t.Errorf("response: got %v, want %v", output.Response(), input.Response())
			}
		})
	}
}
