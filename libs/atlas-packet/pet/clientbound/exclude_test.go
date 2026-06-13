package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v83 ida=0x7061a5
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v87 ida=0x74a17a
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=gms_v95 ida=0x6a1510
// packet-audit:verify packet=pet/clientbound/PetExcludeResponse version=jms_v185 ida=0x76be76
func TestPetExcludeResponse(t *testing.T) {
	input := NewPetExcludeResponse(1234, 0, 999888777, []uint32{2000000, 2000001, 2000002})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
