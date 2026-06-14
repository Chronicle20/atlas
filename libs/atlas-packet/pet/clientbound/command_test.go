package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v83 ida=0x7048ab
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v87 ida=0x74858a
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v95 ida=0x6a3930
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=jms_v185 ida=0x76a6ab
// packet-audit:verify packet=pet/clientbound/PetCommandResponse version=gms_v84 ida=0x720fd0
func TestPetCommandResponse(t *testing.T) {
	input := NewPetCommandResponse(1234, 0, 3, true, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestPetFoodResponse(t *testing.T) {
	input := NewPetFoodResponse(1234, 1, 5, false, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
