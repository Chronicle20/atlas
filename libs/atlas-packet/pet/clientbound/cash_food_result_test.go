package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetCashFoodResult version=gms_v83 ida=0xa29049
// packet-audit:verify packet=pet/clientbound/PetCashFoodResult version=gms_v87 ida=0xac0cbf
// packet-audit:verify packet=pet/clientbound/PetCashFoodResult version=gms_v95 ida=0x9f7180
// packet-audit:verify packet=pet/clientbound/PetCashFoodResult version=jms_v185 ida=0xb102d5
// packet-audit:verify packet=pet/clientbound/PetCashFoodResult version=gms_v84 ida=0xa7480c
func TestPetCashFoodResult(t *testing.T) {
	input := NewPetCashFoodResult(2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestPetCashFoodResultError(t *testing.T) {
	input := NewPetCashFoodResultError()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
