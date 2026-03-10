package character

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestChalkboardUse(t *testing.T) {
	input := NewChalkboardUse(1234, "Selling scrolls!")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestChalkboardClear(t *testing.T) {
	input := NewChalkboardClear(1234)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
