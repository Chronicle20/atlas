package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// SnowballState read order re-derived from the live IDBs and found identical in
// every version (two m_aSnowBall SetPos entries + a first-gated three-short
// damage tail). The prior golden modelled one snowball and an unconditional
// tail — a false pass; corrected here across all versions (task-181).
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v79 ida=0x5525bf
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v83 ida=0x5750a3
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v84 ida=0x584a1c
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v87 ida=0x5a3328
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=gms_v95 ida=0x560ab0
// packet-audit:verify packet=field/clientbound/FieldSnowballState version=jms_v185 ida=0x5c959d
func TestSnowballStateGolden(t *testing.T) {
	// first == true -> the three damage shorts are appended (initial snapshot).
	input := NewSnowballState(0x01, 0x00000064, 0x00000032, 0x0003, 0x02, 0x0004, 0x05, true, 0x0006, 0x0007, 0x0008)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x01,                   // state
		0x64, 0x00, 0x00, 0x00, // leftSnowmanHp
		0x32, 0x00, 0x00, 0x00, // rightSnowmanHp
		0x03, 0x00, 0x02, // snowball0 x=3 y=2
		0x04, 0x00, 0x05, // snowball1 x=4 y=5
		0x06, 0x00, // damageSnowBall
		0x07, 0x00, // damageSnowMan0
		0x08, 0x00, // damageSnowMan1
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSnowballStateByteOutputV79 pins the gms_v79 SNOWBALL_STATE clientbound
// read. IDA: CField_SnowBall::OnSnowBallState @0x5525bf (GMS_v79_1_DEVM.exe) —
// Decode1(state) + Decode4(left) + Decode4(right) + 2x{Decode2 x, Decode1 y} +
// (first) 3x Decode2. Byte-identical to v83/v84/v87/v95/jms.
func TestSnowballStateByteOutputV79(t *testing.T) {
	input := NewSnowballState(0x01, 0x00000064, 0x00000032, 0x0003, 0x02, 0x0004, 0x05, true, 0x0006, 0x0007, 0x0008)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{
		0x01,
		0x64, 0x00, 0x00, 0x00,
		0x32, 0x00, 0x00, 0x00,
		0x03, 0x00, 0x02,
		0x04, 0x00, 0x05,
		0x06, 0x00,
		0x07, 0x00,
		0x08, 0x00,
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSnowballStateGoldenNotFirst confirms the damage tail is omitted when the
// packet is not the initial snapshot (the client gates it on its own state).
func TestSnowballStateGoldenNotFirst(t *testing.T) {
	input := NewSnowballState(0x01, 0x00000064, 0x00000032, 0x0003, 0x02, 0x0004, 0x05, false, 0, 0, 0)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x01,
		0x64, 0x00, 0x00, 0x00,
		0x32, 0x00, 0x00, 0x00,
		0x03, 0x00, 0x02,
		0x04, 0x00, 0x05,
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("not-first golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSnowballStateRoundTrip(t *testing.T) {
	for _, first := range []bool{true, false} {
		// When first is false the damage tail is not on the wire, so Decode
		// cannot recover it — zero those fields to keep the round-trip exact.
		var d0, d1, d2 uint16
		if first {
			d0, d1, d2 = 0x0006, 0x0007, 0x0008
		}
		input := NewSnowballState(0x01, 0x00000064, 0x00000032, 0x0003, 0x02, 0x0004, 0x05, first, d0, d1, d2)
		for _, v := range test.Variants {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		}
	}
}
