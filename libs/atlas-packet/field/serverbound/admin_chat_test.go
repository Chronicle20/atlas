package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldAdminChat version=gms_v72 ida=0x50f8b8
// packet-audit:verify packet=field/serverbound/FieldAdminChat version=gms_v79 ida=0x5194ac
// packet-audit:verify packet=field/serverbound/FieldAdminChat version=gms_v83 ida=0x52de5a
// packet-audit:verify packet=field/serverbound/FieldAdminChat version=gms_v84 ida=0x539f6a
// packet-audit:verify packet=field/serverbound/FieldAdminChat version=gms_v87 ida=0x554e3b
// packet-audit:verify packet=field/serverbound/FieldAdminChat version=gms_v95 ida=0x541d57
// packet-audit:verify packet=field/serverbound/FieldAdminChat version=jms_v185 ida=0x5685b0
// TestAdminChatByteOutputV79 pins the gms_v79 ADMIN_CHAT (op 0x73) serverbound
// wire. IDA: CField::SendChatMsgSlash send-site @0x5194ac (GMS_v79_1_DEVM.exe) —
//
//	COutPacket(0x73)         @0x5194b1 → opcode 0x73 (matches registry).
//	COutPacket::Encode1(1)   @0x5194bf → chatType byte.
//	COutPacket::Encode1(1)   @0x5194c9 → flag byte.
//	COutPacket::EncodeStr    (String)  → message string.
//
// Body is version-uniform across send-sites; only the opcode shifts per version.
func TestAdminChatByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewAdminChat(0x01, 0x02, "hi")
	expected := []byte{0x01, 0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 admin_chat golden mismatch: got %v want %v", actual, expected)
	}
}

// TestAdminChatByteOutputV72 pins the gms_v72 ADMIN_CHAT (op 0x74) serverbound
// wire. IDA: CField::SendChatMsgSlash admin-chat send-site @0x50f8b8
// (GMS_v72.1_U_DEVM.exe) — one of many structurally-identical COutPacket(0x74)
// send-sites in the /-command parser, all with the same body shape:
//
//	push 0x74; COutPacket ctor  @0x50f8c0 → opcode 0x74 (matches registry).
//	COutPacket::Encode1         @0x50f8d1 → chatType byte.
//	COutPacket::Encode1         @0x50f8de → flag byte.
//	COutPacket::EncodeStr       @0x50f8f8 → message string.
//
// v72 is GMS<87 so the body matches the v79 legacy wire byte-for-byte.
func TestAdminChatByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewAdminChat(0x01, 0x02, "hi")
	expected := []byte{0x01, 0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 admin_chat golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAdminChatGolden(t *testing.T) {
	input := NewAdminChat(0x01, 0x02, "hi")
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAdminChatRoundTrip(t *testing.T) {
	input := NewAdminChat(0x01, 0x02, "hi")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := AdminChat{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChatType() != input.ChatType() || output.Flag() != input.Flag() || output.Message() != input.Message() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
