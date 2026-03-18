package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestPetExcludeResponse(t *testing.T) {
	input := NewPetExcludeResponse(1234, 0, 999888777, []uint32{2000000, 2000001, 2000002})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
