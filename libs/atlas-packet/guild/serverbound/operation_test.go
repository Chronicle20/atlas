package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Operation{op: 7}
			output := Operation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Op() != input.Op() {
				t.Errorf("op: got %v, want %v", output.Op(), input.Op())
			}
		})
	}
}
