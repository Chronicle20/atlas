package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetActivated version=gms_v83 ida=0x983aff
// packet-audit:verify packet=pet/clientbound/PetActivated version=gms_v95 ida=0x9547d0
// packet-audit:verify packet=pet/clientbound/PetActivated version=jms_v185 ida=0xa576d3
// packet-audit:verify packet=pet/clientbound/PetActivated version=gms_v84 ida=0x9c3e9d
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
