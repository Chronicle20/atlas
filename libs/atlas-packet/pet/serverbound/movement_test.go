package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v83 ida=0x9c4e41
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v87 ida=0xa558b6
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v95 ida=0x99f5a0
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=jms_v185 ida=0xaa25ab
// packet-audit:verify packet=pet/serverbound/PetMovementRequest version=gms_v84 ida=0xa0c600
func TestPetMovement(t *testing.T) {
	p := MovementRequest{}
	p.petId = 5000001

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.PetId() != 5000001 {
				t.Errorf("expected petId 5000001, got %d", p.PetId())
			}
			if p.PetIdAsUint32() != 5000001 {
				t.Errorf("expected petIdAsUint32 5000001, got %d", p.PetIdAsUint32())
			}
		})
	}
}

func TestPetMovementOperationString(t *testing.T) {
	p := MovementRequest{}
	if p.Operation() != PetMovementHandle {
		t.Errorf("expected operation %s, got %s", PetMovementHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}
