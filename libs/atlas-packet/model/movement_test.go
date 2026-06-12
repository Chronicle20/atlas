package model

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// movementTypesV84 returns a GMS v84 move-action "types" table (indices 0..23).
// Index 23 is the FLYING_BLOCK action added in v84; index 0 is NORMAL. The rest
// are filler so index 23 is in range.
func movementTypesV84() map[string]interface{} {
	types := make([]interface{}, 24)
	for i := range types {
		types[i] = map[string]interface{}{"Name": "UNKNOWN", "Type": "DEFAULT"}
	}
	types[0] = map[string]interface{}{"Name": "NORMAL", "Type": "NORMAL"}
	types[23] = map[string]interface{}{"Name": "FLYING_BLOCK", "Type": "FLYING_BLOCK"}
	return map[string]interface{}{"types": types}
}

// TestMovementFlyingBlockType23 pins GMS v84 move action 23. v84's
// CMovePath::Decode (client sub_6A0FD0) added a case 23 that reads x,y,vx,vy plus
// the common (bMoveAction, tElapse) tail — the FLYING_BLOCK shape (v83 has no such
// case). With index 23 absent from the configured types table the decoder treats
// the element as a 3-byte stub and desyncs the rest of the packet, producing the
// live "Code [255] not configured for use in movement" flood and a client crash.
func TestMovementFlyingBlockType23(t *testing.T) {
	options := movementTypesV84()
	ctx := test.CreateContext("GMS", 84, 1)

	m := &Movement{
		StartX: 10,
		StartY: 20,
		Elements: []MovementCodec{
			&FlyingBlockElement{Element{ElemType: 23, X: 100, Y: 200, Vx: 5, Vy: -3, BMoveAction: 7, TElapse: 50}},
		},
	}

	// Alignment: a clean round-trip with no unconsumed bytes proves the type-23
	// element is sized correctly (type + x,y,vx,vy + bMoveAction + tElapse).
	test.RoundTrip(t, ctx, m.Encode, (&Movement{}).Decode, options)

	// Categorization: type 23 must decode back as a FlyingBlockElement with fields
	// preserved (not silently downgraded to a bare Element).
	encoded := test.Encode(t, ctx, m.Encode, options)
	req := request.Request(encoded)
	reader := request.NewRequestReader(&req, 0)
	out := &Movement{}
	out.Decode(logrus.New(), ctx)(&reader, options)

	if len(out.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(out.Elements))
	}
	fb, ok := out.Elements[0].(*FlyingBlockElement)
	if !ok {
		t.Fatalf("type 23 decoded as %T, want *FlyingBlockElement", out.Elements[0])
	}
	if fb.X != 100 || fb.Y != 200 || fb.Vx != 5 || fb.Vy != -3 || fb.BMoveAction != 7 || fb.TElapse != 50 {
		t.Errorf("flying-block fields not preserved: x=%d y=%d vx=%d vy=%d move=%d elapse=%d",
			fb.X, fb.Y, fb.Vx, fb.Vy, fb.BMoveAction, fb.TElapse)
	}
}
