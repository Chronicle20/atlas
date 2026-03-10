package field

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestKiteError(t *testing.T) {
	input := NewKiteError()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
