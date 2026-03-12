package pet

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

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
