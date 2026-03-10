package inventory

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestCompartmentSortW(t *testing.T) {
	input := NewCompartmentSortW(2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
