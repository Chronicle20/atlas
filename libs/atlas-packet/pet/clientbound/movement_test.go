package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v83 ida=0x70474d
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v87 ida=0x74842a
// packet-audit:verify packet=pet/clientbound/PetMovement version=gms_v95 ida=0x69fb60
// packet-audit:verify packet=pet/clientbound/PetMovement version=jms_v185 ida=0x76a534
func TestPetMovementRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			input := NewPetMovement(2001, 0, model.Movement{})
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
