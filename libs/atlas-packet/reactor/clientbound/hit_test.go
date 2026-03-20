package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestReactorHit(t *testing.T) {
	input := NewReactorHit(100, 2, 150, -300, 5)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
