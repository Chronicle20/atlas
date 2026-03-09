package interaction

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationMemoryGameFlipCardRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameFlipCard{first: true, index: 7}
			output := OperationMemoryGameFlipCard{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.First() != input.First() {
				t.Errorf("first: got %v, want %v", output.First(), input.First())
			}
			if output.Index() != input.Index() {
				t.Errorf("index: got %v, want %v", output.Index(), input.Index())
			}
		})
	}
}
