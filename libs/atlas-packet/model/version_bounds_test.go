package model

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNormalElementMovementVersionBoundary pins the corrected movement
// XOffset/YOffset boundary (delta §3.1.8). The encode side already gated >87;
// the decode side was the stale >83. Both must be identical (>87) or Atlas
// corrupts its own movement packets. v84..86 == v83 (5 Int16, no XOffset);
// v87 also == v83 (XOffset is >87, i.e. v88+); v95 carries XOffset.
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
	// v84..87 are all on the pre-XOffset side (encode gates >87 == v88+).
	for _, major := range []uint16{84, 85, 86, 87} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("NormalElement v%d encode differs from v83 (len %d vs %d); v84..87 must match v83 (XOffset is >87)", major, len(got), len(v83))
		}
	}
	if v95 := encode(95); bytes.Equal(v95, v83) {
		t.Errorf("NormalElement v95 must carry XOffset/YOffset, not equal v83")
	}

	// Decode must mirror Encode exactly: a v84 buffer (no XOffset) round-trips
	// cleanly with no leftover bytes. A stale >83 decode would over-read.
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
