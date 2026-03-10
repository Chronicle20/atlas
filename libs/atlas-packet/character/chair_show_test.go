package character

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterChairShow(t *testing.T) {
	input := NewCharacterChairShow(1234, 3010000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
