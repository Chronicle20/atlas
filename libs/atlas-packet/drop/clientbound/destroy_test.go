package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestDropDestroyPickUp(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypePickUp, 1234, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestDropDestroyExpire(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypeExpire, 0, -1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestDropDestroyExplode pins the v95 wire shape for destroyType == 4:
// byte(4) + int(dropId) + int16(tLeaveDelay) = 7 bytes. The legacy
// NewDropDestroy(dropId, 4, charId, -1) path emits the same shape with
// delay = 0 since callers historically passed characterId=0/petSlot=-1.
func TestDropDestroyExplode(t *testing.T) {
	input := NewDropDestroyExplode(9001, 500)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	bytes := input.Encode(l, ctx)(nil)
	if len(bytes) != 7 {
		t.Errorf("explode encode: got %d bytes, want 7 (byte type + uint32 dropId + int16 delay)", len(bytes))
	}
}

// TestDropDestroyPetPickUp pins the v95 wire shape for destroyType == 5:
// byte(5) + int(dropId) + int(pickupCharId) + int(petPickupExtra) = 13 bytes.
// Legacy NewDropDestroy with petSlot >= 0 widens the petSlot to the
// int4 v95 reads inside the case 5 body.
func TestDropDestroyPetPickUp(t *testing.T) {
	input := NewDropDestroy(9001, DropDestroyTypePetPickUp, 1234, 2)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	bytes := input.Encode(l, ctx)(nil)
	if len(bytes) != 13 {
		t.Errorf("pet pickup encode: got %d bytes, want 13 (byte type + uint32 dropId + uint32 charId + uint32 extra)", len(bytes))
	}
}
