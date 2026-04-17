package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationMemoryGameTieAnswerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameTieAnswer{response: true}
			output := OperationMemoryGameTieAnswer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Response() != input.Response() {
				t.Errorf("response: got %v, want %v", output.Response(), input.Response())
			}
		})
	}
}
