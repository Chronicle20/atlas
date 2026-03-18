package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestPetSpawnActivated(t *testing.T) {
	input := NewPetSpawnActivated(1234, 0, 5000100, "Kitty", 999888777, 100, -200, 4, 300)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestPetDespawnActivated(t *testing.T) {
	input := NewPetDespawnActivated(1234, 1, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
