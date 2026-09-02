package model

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestNormalElementMovementVersionBoundary pins the OUTBOUND movement
// XOffset/YOffset boundary — what Atlas writes to the client.
//
// The pair is directional. GMS v87 sends it (CMovePath::Encode @0x6c70fe,
// `mov ax,[edi+14h]` / `[edi+16h]` at 6c720a/6c7218) but never reads it back
// (CMovePath::Decode @0x6c6e86 goes from fh straight to the bMoveAction/tElapse
// tail), so on the wire out of Atlas v87 must look like v83. v92 onward reads
// what it writes. The inbound side is pinned separately by
// TestNormalElementOffsetsAreDirectional.
func TestNormalElementMovementVersionBoundary(t *testing.T) {
	build := func() *NormalElement {
		return &NormalElement{Element{
			X: 1, Y: 2, Vx: 3, Vy: 4, Fh: 5,
			XOffset: 6, YOffset: 7,
			BMoveAction: 8, TElapse: 9, ElemType: 0,
		}}
	}
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		m := build()
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	// v84..v86 are on the pre-XOffset side. v84 is IDA-verified: its
	// CMovePath::Decode @0x6a0fd0 has no XOffset/YOffset read at all (the
	// per-element tail is Decode1 + Decode2 and nothing else).
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("NormalElement v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	// v87 is on the pre-XOffset side OUTBOUND, even though it is on the
	// post-XOffset side inbound. Writing the pair back to a v87 client makes it
	// read xOffset's low byte as bMoveAction and yOffset as tElapse, then read
	// the real bMoveAction/tElapse as the next fragment's attr and body — the
	// "NPC teleports instead of walking" symptom.
	if v87 := encode(87); !bytes.Equal(v87, v83) {
		t.Errorf("NormalElement v87 outbound encode is %d bytes, want the v83 layout (%d); v87's CMovePath::Decode @0x6c6e86 does not read XOffset/YOffset", len(v87), len(v83))
	}
	// v92 is the lowest version IDA-verified to READ the pair back
	// (CMovePath::Decode @0x65ad60), so it is the lowest that may be sent it.
	v92 := encode(92)
	if bytes.Equal(v92, v83) {
		t.Fatalf("NormalElement v92 must carry XOffset/YOffset (+4 bytes), got the v83 layout")
	}
	if len(v92) != len(v83)+4 {
		t.Errorf("v92 NormalElement is %d bytes, want v83 (%d) + 4 for XOffset/YOffset", len(v92), len(v83))
	}
	if v95 := encode(95); !bytes.Equal(v95, v92) {
		t.Errorf("NormalElement v95 must match v92 — both carry XOffset/YOffset")
	}

	// Decode must mirror Encode on the versions where the client is symmetric:
	// a v84 buffer (no XOffset) round-trips cleanly with no leftover bytes.
	ctx84 := pt.CreateContext("GMS", 84, 1)
	in := build()
	out := &NormalElement{}
	pt.RoundTrip(t, ctx84, in.Encode, out.Decode, nil)
	if out.X != in.X || out.Y != in.Y || out.Fh != in.Fh {
		t.Errorf("v84 NormalElement roundtrip mismatch: got X=%d Y=%d Fh=%d", out.X, out.Y, out.Fh)
	}
	// The fields AFTER the (absent) XOffset slot must survive. A stale >83 decode
	// consumes BMoveAction/TElapse as XOffset/YOffset and corrupts them.
	if out.BMoveAction != in.BMoveAction || out.TElapse != in.TElapse {
		t.Errorf("v84 NormalElement decode over-read: got BMoveAction=%d TElapse=%d, want %d/%d (stale >83 decode consumed them as XOffset)", out.BMoveAction, out.TElapse, in.BMoveAction, in.TElapse)
	}
}

// TestNormalElementOffsetsAreDirectional pins the one place in this codec where
// Decode and Encode MUST NOT match, and pins it from both sides at once so a
// future "the two gates should be identical" cleanup fails loudly.
//
// GMS v87 is the only version in the matrix whose CMovePath is asymmetric:
// Encode @0x6c70fe writes ELEM +0x14/+0x16 for the absolute-position attrs,
// Decode @0x6c6e86 never reads them. Atlas must therefore READ the pair from a
// v87 client and NOT WRITE it back. Every other version reads back what it
// writes, so encode and decode agree there.
func TestNormalElementOffsetsAreDirectional(t *testing.T) {
	for _, v := range []struct {
		name         string
		region       string
		major        uint16
		wantInbound  bool
		wantOutbound bool
	}{
		{"GMS v83 neither side", "GMS", 83, false, false},
		{"GMS v84 neither side", "GMS", 84, false, false},
		{"GMS v87 reads but does not write", "GMS", 87, true, false},
		{"GMS v92 both sides", "GMS", 92, true, true},
		{"GMS v95 both sides", "GMS", 95, true, true},
		{"JMS v185 both sides", "JMS", 185, true, true},
	} {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			in := &NormalElement{Element{
				X: 1, Y: 2, Vx: 3, Vy: 4, Fh: 5,
				XOffset: 6, YOffset: 7,
				BMoveAction: 8, TElapse: 9, ElemType: 0,
			}}

			// Encode width: base is attr-less body x,y,vx,vy,fh + bMoveAction +
			// tElapse == 13 bytes; the pair adds 4.
			const base = 5*2 + 1 + 2
			wantLen := base
			if v.wantOutbound {
				wantLen += 4
			}
			if got := pt.Encode(t, ctx, in.Encode, nil); len(got) != wantLen {
				t.Errorf("encode produced %d bytes, want %d (outbound offsets: %t)", len(got), wantLen, v.wantOutbound)
			}

			// Decode width: feed a buffer built to the INBOUND shape and require
			// the decoder to consume all of it and land the tail correctly. This
			// is the half that a symmetric gate would get wrong on v87.
			body := []byte{
				0x01, 0x00, // x
				0x02, 0x00, // y
				0x03, 0x00, // vx
				0x04, 0x00, // vy
				0x05, 0x00, // fh
			}
			if v.wantInbound {
				body = append(body, 0x06, 0x00, 0x07, 0x00) // xOffset, yOffset
			}
			body = append(body, 0x08, 0x0A, 0x00) // bMoveAction, tElapse

			req := request.Request(body)
			reader := request.NewRequestReader(&req, 0)
			l, _ := testlog.NewNullLogger()
			out := &NormalElement{Element{ElemType: 0}}
			out.Decode(l, ctx)(&reader, nil)

			if reader.Position() != len(body) {
				t.Errorf("decode consumed %d of %d bytes (inbound offsets: %t)", reader.Position(), len(body), v.wantInbound)
			}
			// A misaligned decode eats bMoveAction/tElapse as the offsets, which
			// is precisely how the client desyncs when Atlas gets this wrong.
			if out.BMoveAction != 8 || out.TElapse != 10 {
				t.Errorf("tail corrupted: BMoveAction=%d TElapse=%d, want 8/10", out.BMoveAction, out.TElapse)
			}
			wantXOffset, wantYOffset := int16(0), int16(0)
			if v.wantInbound {
				wantXOffset, wantYOffset = 6, 7
			}
			if out.XOffset != wantXOffset || out.YOffset != wantYOffset {
				t.Errorf("offsets = (%d,%d), want (%d,%d)", out.XOffset, out.YOffset, wantXOffset, wantYOffset)
			}
		})
	}
}

// TestMonsterModelVersionBoundary pins the corrected >83 -> >=87 boundary for
// the monster spawn `phase` int (delta §3.2): v84..86 encode byte-identically
// to v83 (no phase int). v87/v95 carry phase.
func TestMonsterModelVersionBoundary(t *testing.T) {
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		m := NewMonster(100, 200, 5, 7, MonsterAppearTypeNormal, 0)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("MonsterModel v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("MonsterModel v87 must carry the phase int, not equal v83")
	}
}
