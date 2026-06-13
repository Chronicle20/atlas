package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v83 ida=0x70476e
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v87 ida=0x74844b
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v95 ida=0x6a3860
// packet-audit:verify packet=pet/clientbound/PetChat version=jms_v185 ida=0x76a557
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v84 ida=0x720e91
func TestPetChat(t *testing.T) {
	input := NewPetChat(1234, 0, 1, 5, "Hello!", true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
