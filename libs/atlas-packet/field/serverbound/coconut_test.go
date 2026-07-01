package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v79 ida=0x5333c8
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v83 ida=0x549902
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v84 ida=0x556075
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v87 ida=0x5735b7
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v95 ida=0x54a5e0
// packet-audit:verify packet=field/serverbound/FieldCoconut version=jms_v185 ida=0x589bd1
func TestCoconutGolden(t *testing.T) {
	input := NewCoconut(0x0102, 0x0304)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x02, 0x01, 0x04, 0x03}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestCoconutByteOutputV79 pins the gms_v79 COCONUT (op 0xCD) serverbound wire.
// IDA: CField_Coconut::BasicActionAttack (was sub_5333C8) @0x5333c8
// (GMS_v79_1_DEVM.exe) — COutPacket(205) @0x533483, Encode2(attack v7) @0x533490,
// Encode2(x v13) @0x53349b. Body = attack(2 LE) + x(2 LE).
func TestCoconutByteOutputV79(t *testing.T) {
	input := NewCoconut(0x0102, 0x0304)
	ctx := pt.CreateContext("GMS", 79, 1)
	expected := []byte{0x02, 0x01, 0x04, 0x03}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 coconut golden mismatch: got %v want %v", actual, expected)
	}
}

// TestCoconutByteOutputV72 pins the gms_v72 COCONUT (op 0xCB = 203) serverbound
// wire. IDA: CField_Coconut::BasicActionAttack (was sub_52699C, renamed to its
// v79 twin) @0x52699c (GMS_v72.1_U_DEVM.exe) — COutPacket(203) @0x526a57,
// Encode2(attack v7) @0x526a64, Encode2(x v13) @0x526a6f, then SendPacket.
// Body = attack(2 LE) + x(2 LE) — identical to the v79 golden (op 205).
// packet-audit:verify packet=field/serverbound/FieldCoconut version=gms_v72 ida=0x52699c
func TestCoconutByteOutputV72(t *testing.T) {
	input := NewCoconut(0x0102, 0x0304)
	ctx := pt.CreateContext("GMS", 72, 1)
	expected := []byte{0x02, 0x01, 0x04, 0x03}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 coconut golden mismatch: got %v want %v", actual, expected)
	}
}

func TestCoconutRoundTrip(t *testing.T) {
	input := NewCoconut(0x0102, 0x0304)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Coconut{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Attack() != input.Attack() || output.X() != input.X() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
