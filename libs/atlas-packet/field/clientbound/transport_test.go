package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestFieldTransport(t *testing.T) {
	input := NewFieldTransport(TransportStateMove1, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
