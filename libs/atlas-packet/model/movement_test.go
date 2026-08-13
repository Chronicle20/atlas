package model

import (
	"bytes"
	"fmt"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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

// TestMovementHeaderVersionBoundary pins the CMovePath header across the v88
// rework. v83/v84/v87 and JMS write x,y,count; v92/v95 write x,y,vx,vy,count.
//
// Evidence (movement-types-derivation.md §5):
//
//	v83 CMovePath::Encode@0x68a563 — Encode2 @0x68a57c, @0x68a592, Encode1 @0x68a5c3
//	v87 @0x6c70fe                  — Encode2 @0x6c7118, @0x6c712e, Encode1 @0x6c715f
//	jms @0x70b6c4                  — Encode2 @0x70b6de, @0x70b6f4, Encode1 @0x70b725
//	v92 @0x65a260                  — Encode2 @0x65a284,@0x65a29f,@0x65a2ba,@0x65a2d5, Encode1 @0x65a306
//	v95 @0x666e20                  — Encode2 @0x666e44,@0x666e5f,@0x666e7a,@0x666e95, Encode1 @0x666ec6
//
// The four header bytes precede the element count, so with them unread numElems
// is parsed out of the low byte of vx and every subsequent read is garbage —
// no matter how correct the configured `types` table is.
func TestMovementHeaderVersionBoundary(t *testing.T) {
	build := func() *Movement {
		return &Movement{StartX: 100, StartY: 200, StartVx: 5, StartVy: -3}
	}
	encode := func(region string, major uint16) []byte {
		ctx := test.CreateContext(region, major, 1)
		return test.Encode(t, ctx, build().Encode, nil)
	}

	// Pre-rework: x(2) + y(2) + count(1) = 5 bytes. vx/vy are NOT on the wire.
	wantShort := []byte{0x64, 0x00, 0xC8, 0x00, 0x00}
	for _, v := range []struct {
		region string
		major  uint16
	}{
		{"GMS", 83}, {"GMS", 84}, {"GMS", 87}, {"JMS", 185},
	} {
		if got := encode(v.region, v.major); !bytes.Equal(got, wantShort) {
			t.Errorf("%s v%d header = % x, want % x (2-field header)", v.region, v.major, got, wantShort)
		}
	}

	// v88+: x(2) + y(2) + vx(2) + vy(2) + count(1) = 9 bytes, little-endian.
	wantLong := []byte{0x64, 0x00, 0xC8, 0x00, 0x05, 0x00, 0xFD, 0xFF, 0x00}
	for _, major := range []uint16{92, 95} {
		if got := encode("GMS", major); !bytes.Equal(got, wantLong) {
			t.Errorf("GMS v%d header = % x, want % x (4-field header)", major, got, wantLong)
		}
	}
}

// TestMovementHeaderRoundTrip proves Decode mirrors Encode exactly on both sides
// of the gate. A one-sided gate silently corrupts Atlas's own outbound packets.
func TestMovementHeaderRoundTrip(t *testing.T) {
	for _, v := range []struct {
		name   string
		region string
		major  uint16
		wantVx int16
		wantVy int16
	}{
		{"GMS v87 drops vx/vy", "GMS", 87, 0, 0},
		{"JMS v185 drops vx/vy", "JMS", 185, 0, 0},
		{"GMS v92 carries vx/vy", "GMS", 92, 5, -3},
		{"GMS v95 carries vx/vy", "GMS", 95, 5, -3},
	} {
		t.Run(v.name, func(t *testing.T) {
			ctx := test.CreateContext(v.region, v.major, 1)
			in := &Movement{StartX: 100, StartY: 200, StartVx: 5, StartVy: -3}
			out := &Movement{}
			// No unconsumed bytes proves the header is sized identically both ways.
			test.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
			if out.StartX != 100 || out.StartY != 200 {
				t.Errorf("start position lost: x=%d y=%d", out.StartX, out.StartY)
			}
			if out.StartVx != v.wantVx || out.StartVy != v.wantVy {
				t.Errorf("start velocity = (%d,%d), want (%d,%d)", out.StartVx, out.StartVy, v.wantVx, v.wantVy)
			}
		})
	}
}

// TestUnconfiguredMovementCodeLogsOnceWithFrame pins the diagnostic contract for
// a movement fragment whose type code is missing from the tenant's table.
//
// Two things were wrong with the old behaviour, and both cost real triage time
// on the GMS v87 occurrence (task-218 field reports #3/#5). The lookup logged
// from inside itself, and Decode asks it once per candidate element kind, so a
// single bad code produced six identical lines — which read as six problems and
// inflated the apparent rate by 6x. And the line carried only the code, so it
// could not distinguish "the client sent a fragment type we do not model" from
// "the reader is off a fragment boundary", which is what it actually means.
//
// The contract now: exactly ONE error per bad element, carrying the frame bytes
// and the reader offset.
func TestUnconfiguredMovementCodeLogsOnceWithFrame(t *testing.T) {
	ctx := test.CreateContext("GMS", 87, 1)
	// A types table with a single entry, so code 200 cannot resolve.
	options := map[string]interface{}{
		"types": []interface{}{
			map[string]interface{}{"Name": "NORMAL", "Type": "NORMAL"},
		},
	}

	l, hook := testlog.NewNullLogger()
	// startX, startY, one element, type code 200, then a short tail.
	raw := []byte{0x0A, 0x00, 0x14, 0x00, 0x01, 200, 0x01, 0x02, 0x03}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	(&Movement{}).Decode(l, ctx)(&reader, options)

	var errs []string
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel {
			errs = append(errs, e.Message)
		}
	}
	if len(errs) != 1 {
		t.Fatalf("got %d error lines for one unconfigured code, want exactly 1: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0], "Code [200]") {
		t.Errorf("error must name the offending code: %q", errs[0])
	}
	// The frame is the whole point — without it the report is unactionable.
	if !strings.Contains(errs[0], fmt.Sprintf("%x", raw)) {
		t.Errorf("error must carry the raw frame for diagnosis: %q", errs[0])
	}
	if !strings.Contains(errs[0], "misalignment") {
		t.Errorf("error must say what an out-of-range code actually means: %q", errs[0])
	}
}
