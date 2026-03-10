package pet

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestPetChatW(t *testing.T) {
	input := NewPetChatW(1234, 0, 1, 5, "Hello!", true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
