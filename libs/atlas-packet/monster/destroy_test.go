package monster

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestMonsterDestroy(t *testing.T) {
	input := NewMonsterDestroy(5001, DestroyTypeFadeOut)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
