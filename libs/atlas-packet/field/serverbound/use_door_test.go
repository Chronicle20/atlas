package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v72 ida=0x51b86c
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v79 ida=0x522946
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v83 ida=0x5375ed
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v84 ida=0x5438eb
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v87 ida=0x55ef62
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v95 ida=0x52f970
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=jms_v185 ida=0x574826
func TestUseDoorGolden(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x05)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x05}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestUseDoorByteOutputV79 pins the gms_v79 USE_DOOR (op 0x82) serverbound wire.
// IDA: CField::TryEnterTownPortal @0x522946 (GMS_v79_1_DEVM.exe) builds
// COutPacket(130) + Encode4(portalFieldId) @0x522b1b + Encode1(1) @0x522b24.
func TestUseDoorByteOutputV79(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x01)
	ctx := pt.CreateContext("GMS", 79, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestUseDoorByteOutputV72 pins the gms_v72 USE_DOOR (op 0x83 / 131) serverbound
// wire. IDA: CField::TryEnterTownPortal @0x51b86c (GMS_v72.1_U_DEVM.exe) builds
// COutPacket(131) @0x51ba31 + Encode4(portalFieldId) @0x51ba41 + Encode1(1) @0x51ba4a
// (party-town-portal send path; the single-portal path @0x51b941 is byte-identical).
// Body identical to v79 (op 130); only the opcode shifts +1 (registry gms_v72 op 131).
func TestUseDoorByteOutputV72(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x01)
	ctx := pt.CreateContext("GMS", 72, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestUseDoorByteOutputV48 pins the gms_v48 USE_DOOR (op 0x67 = 103) serverbound
// wire. IDA: CField::TryEnterTownPortal = sub_5E3082 @0x5e3082 (GMS_v48_1_DEVM.exe)
// builds COutPacket(103) @0x5e3125 + Encode4(portalFieldId v17[2]) @0x5e3137 +
// Encode1(0=flag) @0x5e3140. Body = portalFieldId(4 LE) + flag(1) — identical to the
// v61 golden (op 121); only the opcode shifts.
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v48 ida=0x5e3082
func TestUseDoorByteOutputV48(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x00)
	ctx := pt.CreateContext("GMS", 48, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x00}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v48 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestUseDoorByteOutputV61 pins the gms_v61 USE_DOOR (op 0x79 = 121) serverbound
// wire. IDA: CField::TryEnterTownPortal @0x4ef641 (GMS_v61.1_U_DEVM.exe) builds
// COutPacket(121) + Encode4(portalFieldId) + Encode1(1) (both the party-portal and
// single-portal paths are byte-identical). Body = portalFieldId(4 LE) + flag(1) —
// identical to the v72 golden (op 131).
// packet-audit:verify packet=field/serverbound/FieldUseDoor version=gms_v61 ida=0x4ef641
func TestUseDoorByteOutputV61(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x01)
	ctx := pt.CreateContext("GMS", 61, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestUseDoorRoundTrip(t *testing.T) {
	input := NewUseDoor(0x01020304, 0x05)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := UseDoor{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PortalFieldId() != input.PortalFieldId() || output.Flag() != input.Flag() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
