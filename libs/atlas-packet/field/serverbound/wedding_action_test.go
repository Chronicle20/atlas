package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v72 ida=0x548c50
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v79 ida=0x55dfbb
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v83 ida=0x58153d
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v84 ida=0x5911e6
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v87 ida=0x5b012e
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v95 ida=0x5640f0
func TestWeddingActionGolden(t *testing.T) {
	input := NewWeddingAction(0x02)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x02}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWeddingActionByteOutputV79 pins the gms_v79 WEDDING_ACTION (op 0x88)
// serverbound wire. IDA: CField_Wedding::OnWeddingProgress @0x55dfbb
// (GMS_v79_1_DEVM.exe) builds COutPacket(136) @0x55e547 + Encode1(readyByte)
// @0x55e55c.
func TestWeddingActionByteOutputV79(t *testing.T) {
	input := NewWeddingAction(0x02)
	ctx := pt.CreateContext("GMS", 79, 1)
	expected := []byte{0x02}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWeddingActionByteOutputV72 pins the gms_v72 WEDDING_ACTION (op 0x89 / 137)
// serverbound wire. IDA: CField_Wedding::OnWeddingProgress @0x548c50
// (GMS_v72.1_U_DEVM.exe) Action arm builds COutPacket(137) @0x5491dc +
// Encode1(readyByte this+1920) @0x5491f1 — single-byte body. Body identical to
// v79 (op 136); only the opcode shifts +1 (registry gms_v72 op 137).
func TestWeddingActionByteOutputV72(t *testing.T) {
	input := NewWeddingAction(0x02)
	ctx := pt.CreateContext("GMS", 72, 1)
	expected := []byte{0x02}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWeddingActionByteOutputV61 pins the gms_v61 WEDDING_ACTION (op 0x7F = 127)
// serverbound wire. IDA: CField_Wedding::OnWeddingProgress @0x513473
// (GMS_v61.1_U_DEVM.exe) Action arm builds COutPacket(127) + Encode1(readyByte
// this+1832) — single-byte body. Identical to the v72 golden (op 137).
// packet-audit:verify packet=field/serverbound/FieldWeddingAction version=gms_v61 ida=0x513473
func TestWeddingActionByteOutputV61(t *testing.T) {
	input := NewWeddingAction(0x02)
	ctx := pt.CreateContext("GMS", 61, 1)
	expected := []byte{0x02}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestWeddingActionRoundTrip(t *testing.T) {
	input := NewWeddingAction(0x02)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := WeddingAction{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Step() != input.Step() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
