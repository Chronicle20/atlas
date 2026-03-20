package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestNPCActionWithoutMovement(t *testing.T) {
	p := ActionRequest{}
	p.objectId = 12345
	p.unk = 1
	p.unk2 = 2
	p.hasMovement = false

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.ObjectId() != 12345 {
				t.Errorf("expected objectId 12345, got %d", p.ObjectId())
			}
			if p.Unk() != 1 {
				t.Errorf("expected unk 1, got %d", p.Unk())
			}
			if p.Unk2() != 2 {
				t.Errorf("expected unk2 2, got %d", p.Unk2())
			}
			if p.HasMovement() {
				t.Error("expected hasMovement false")
			}
		})
	}
}

func TestNPCActionWithMovement(t *testing.T) {
	p := ActionRequest{}
	p.objectId = 99999
	p.unk = 3
	p.unk2 = 4
	p.hasMovement = true
	// movement with 0 elements (startX=10, startY=20)
	p.movement.StartX = 10
	p.movement.StartY = 20

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.ObjectId() != 99999 {
				t.Errorf("expected objectId 99999, got %d", p.ObjectId())
			}
			if !p.HasMovement() {
				t.Error("expected hasMovement true")
			}
			if p.MovementData().StartX != 10 {
				t.Errorf("expected startX 10, got %d", p.MovementData().StartX)
			}
			if p.MovementData().StartY != 20 {
				t.Errorf("expected startY 20, got %d", p.MovementData().StartY)
			}
		})
	}
}

func TestNPCActionOperationString(t *testing.T) {
	p := ActionRequest{}
	if p.Operation() != NPCActionHandle {
		t.Errorf("expected operation %s, got %s", NPCActionHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}
