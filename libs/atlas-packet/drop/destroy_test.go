package drop

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestDropDestroyPickUp(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypePickUp, 1234, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestDropDestroyExpire(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypeExpire, 0, -1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
