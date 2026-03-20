package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestNpcSpawn(t *testing.T) {
	input := NewNpcSpawn(100, 9010000, 150, -300, 0, 500, -50, 250)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
