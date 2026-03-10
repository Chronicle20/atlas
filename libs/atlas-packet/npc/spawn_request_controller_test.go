package npc

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestNpcSpawnRequestController(t *testing.T) {
	input := NewNpcSpawnRequestController(100, 9010000, 150, -300, 0, 500, -50, 250, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
