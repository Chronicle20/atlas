package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldSnowball version=gms_v79 ida=0x5528a3
// packet-audit:verify packet=field/serverbound/FieldSnowball version=gms_v83 ida=0x575387
// packet-audit:verify packet=field/serverbound/FieldSnowball version=gms_v84 ida=0x584d00
// packet-audit:verify packet=field/serverbound/FieldSnowball version=gms_v87 ida=0x5a360c
// packet-audit:verify packet=field/serverbound/FieldSnowball version=gms_v95 ida=0x5617b0
// packet-audit:verify packet=field/serverbound/FieldSnowball version=jms_v185 ida=0x5c9881
func TestSnowballGolden(t *testing.T) {
	input := NewSnowball(0x01, 0x0203, 0x0405)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x03, 0x02, 0x05, 0x04}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSnowballByteOutputV79 pins the gms_v79 SNOWBALL (op 0xCB) serverbound wire.
// IDA: CField_SnowBall::BasicActionAttack @0x5528a3 (GMS_v79_1_DEVM.exe) —
// COutPacket(203) @0x552988, Encode1(attack v8) @0x552995, Encode2(damage v9)
// @0x55299e, Encode2(x v13) @0x5529a9. Body = attack(1) + damage(2 LE) + x(2 LE).
func TestSnowballByteOutputV79(t *testing.T) {
	input := NewSnowball(0x01, 0x0203, 0x0405)
	ctx := pt.CreateContext("GMS", 79, 1)
	expected := []byte{0x01, 0x03, 0x02, 0x05, 0x04}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 snowball golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSnowballByteOutputV72 pins the gms_v72 SNOWBALL (op 0xC9 = 201)
// serverbound wire. IDA: CField_SnowBall::BasicActionAttack @0x5404cc
// (GMS_v72.1_U_DEVM.exe) — COutPacket(201) @0x5405b1, Encode1(attack v8)
// @0x5405be, Encode2(damage v9) @0x5405c7, Encode2(x v13) @0x5405d2, then
// SendPacket. Body = attack(1) + damage(2 LE) + x(2 LE) — identical to the v79
// golden (op 203).
// packet-audit:verify packet=field/serverbound/FieldSnowball version=gms_v72 ida=0x5404cc
func TestSnowballByteOutputV72(t *testing.T) {
	input := NewSnowball(0x01, 0x0203, 0x0405)
	ctx := pt.CreateContext("GMS", 72, 1)
	expected := []byte{0x01, 0x03, 0x02, 0x05, 0x04}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 snowball golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSnowballRoundTrip(t *testing.T) {
	input := NewSnowball(0x01, 0x0203, 0x0405)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Snowball{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Attack() != input.Attack() || output.Damage() != input.Damage() || output.X() != input.X() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
