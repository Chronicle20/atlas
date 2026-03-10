package field

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestKiteDestroy(t *testing.T) {
	input := NewKiteDestroy(1, KiteDestroyAnimationType2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
