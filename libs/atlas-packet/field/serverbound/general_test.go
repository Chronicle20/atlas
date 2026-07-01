package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v79 ida=0x517a02
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v83 ida=0x52c315
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v87 ida=0x552b67
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v95 ida=0x534000
// packet-audit:verify packet=field/serverbound/FieldGeneral version=jms_v185 ida=0x564a0a
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v84 ida=0x5382d7
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v72 ida=0x50b7dc
// TestGeneralByteOutputV79 pins the gms_v79 GENERAL_CHAT (op 0x2F) serverbound
// wire. IDA: CField::SendChatMsg (sub_517A02 @0x517a02, GMS_v79_1_DEVM.exe) —
//
//	COutPacket(47)              @0x517aa1 → opcode 0x2F (matches registry).
//	COutPacket::EncodeStr(v8)   @0x517abe → sText string.
//	COutPacket::Encode1(a2)     @0x517ac9 → bOnlyBalloon byte.
//
// v79 is GMS<87 so there is NO leading get_update_time (the v87+ prefix);
// the codec's MajorAtLeast(87) gate excludes it. WriteAsciiString = uint16-LE
// len + ShiftJIS bytes ("hi" = 02 00 68 69); WriteBool(false) = 00.
func TestGeneralByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := General{msg: "hi", bOnlyBalloon: false}
	expected := []byte{
		0x02, 0x00, 0x68, 0x69, // EncodeStr("hi") @0x517abe
		0x00, // Encode1(bOnlyBalloon=false) @0x517ac9
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 general golden mismatch: got %v want %v", actual, expected)
	}
}

// TestGeneralByteOutputV72 pins the gms_v72 GENERAL_CHAT (op 0x30) serverbound
// wire. IDA: CField::SendChatMsg (sub_50B7DC @0x50b7dc, GMS_v72.1_U_DEVM.exe) —
// the non-slash general-chat send (sibling of CField::SendChatMsgSlash) —
//
//	COutPacket(48)             @0x50b87b → opcode 0x30 (matches registry).
//	COutPacket::EncodeStr(msg) @0x50b898 → message string.
//	COutPacket::Encode1(a2)    @0x50b8a3 → bOnlyBalloon byte.
//
// v72 is GMS<87 so there is NO leading get_update_time (the v87+ prefix); the
// codec's MajorAtLeast(87) gate excludes it — same body shape as v79.
// WriteAsciiString = uint16-LE len + bytes ("hi" = 02 00 68 69); WriteBool(false) = 00.
func TestGeneralByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := General{msg: "hi", bOnlyBalloon: false}
	expected := []byte{
		0x02, 0x00, 0x68, 0x69, // EncodeStr("hi") @0x50b898
		0x00, // Encode1(bOnlyBalloon=false) @0x50b8a3
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 general golden mismatch: got %v want %v", actual, expected)
	}
}

func TestGeneralRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := General{updateTime: 100, msg: "hello world", bOnlyBalloon: true}
			output := General{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Msg() != input.Msg() {
				t.Errorf("msg: got %v, want %v", output.Msg(), input.Msg())
			}
			if output.BOnlyBalloon() != input.BOnlyBalloon() {
				t.Errorf("bOnlyBalloon: got %v, want %v", output.BOnlyBalloon(), input.BOnlyBalloon())
			}
		})
	}
}
