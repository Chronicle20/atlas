package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v72 ida=0x50c2c3
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v79 ida=0x51825e
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v83 ida=0x52cb7c
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v84 ida=0x538c80
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v87 ida=0x553526
// packet-audit:verify packet=field/serverbound/FieldSueCharacter version=gms_v95 ida=0x5413e5
// TestSueCharacterByteOutputV79 pins the gms_v79 SUE_CHARACTER (op 0x70)
// serverbound wire. IDA: CField::SendChatMsgSlash send-site @0x51825e
// (GMS_v79_1_DEVM.exe) —
//
//	COutPacket(0x70)                  @0x518263 → opcode 0x70 (matches registry).
//	COutPacket::Encode4([edi+1078h])  @0x518275 → accused character id (int32 LE).
//	COutPacket::Encode1(esi)          @0x51827e → flag byte.
//	COutPacket::EncodeStr             (String)  → reason string.
//
// v79 is GMS<95 so it uses the legacy int32-charId leading field (the v95
// sub-command-string form is gated MajorVersion>=95).
func TestSueCharacterByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewSueCharacterLegacy(0x01020304, 0x05, "hi")
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x05, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 sue_character golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSueCharacterByteOutputV72 pins the gms_v72 SUE_CHARACTER (op 0x71)
// serverbound wire. IDA: CField::SendChatMsgSlash send-site @0x50c2c3
// (GMS_v72.1_U_DEVM.exe) —
//
//	push 0x71; COutPacket ctor  @0x50c2cb → opcode 0x71 (matches registry).
//	COutPacket::Encode4         @0x50c2e3 → accused character id (int32 LE).
//	COutPacket::Encode1         @0x50c2f1 → flag byte.
//	COutPacket::EncodeStr       @0x50c30e → reason string.
//
// v72 is GMS<95 so it uses the legacy int32-charId leading field (the v95
// sub-command-string form is gated MajorVersion>=95); body == v79 legacy wire.
func TestSueCharacterByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewSueCharacterLegacy(0x01020304, 0x05, "hi")
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x05, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 sue_character golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSueCharacterGoldenLegacy(t *testing.T) {
	// v83/v84/v87 lead with the accused character id (int32).
	input := NewSueCharacterLegacy(0x01020304, 0x05, "hi")
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x05, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSueCharacterGoldenV95(t *testing.T) {
	// v95 leads with a sub-command string.
	input := NewSueCharacterV95("hi", 0x05, "ho")
	ctx := pt.CreateContext("GMS", 95, 1)
	expected := []byte{0x02, 0x00, 0x68, 0x69, 0x05, 0x02, 0x00, 0x68, 0x6f}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSueCharacterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			var input SueCharacter
			// Mirror the codec's version branch: string-lead from v95 onward
			// (jms is SUE-absent in practice; its branch choice is moot here).
			if v.MajorVersion >= 95 {
				input = NewSueCharacterV95("alice", 0x05, "spamming")
			} else {
				input = NewSueCharacterLegacy(0x01020304, 0x05, "spamming")
			}
			output := SueCharacter{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() || output.SubCommand() != input.SubCommand() ||
				output.Flag() != input.Flag() || output.Reason() != input.Reason() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
