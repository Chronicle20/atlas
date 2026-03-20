package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Shop{op: 2}
			output := Shop{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Op() != input.Op() {
				t.Errorf("op: got %v, want %v", output.Op(), input.Op())
			}
		})
	}
}
