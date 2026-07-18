package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestMultiByteOutputV79 pins the gms_v79 MULTI_CHAT (op 0x074) serverbound wire.
//
// IDA-verified send-site (GMS_v79_1_DEVM.exe, port 13340) —
// CUIStatusBar::SendGroupMessage @0x83cebd, send block at LABEL_25:
//
//	COutPacket::COutPacket(116)        @0x83d183 → opcode 0x74 (matches registry).
//	COutPacket::Encode1(v47)           @0x83d192 → chatType byte (group kind: friend=0,
//	                                                friend-group=0, multi=1, expedition=2,
//	                                                family=3 per the per-branch v47 sets).
//	COutPacket::Encode1((u8)v5)        @0x83d19b → recipient count (v5 = member-id array len).
//	for i in 0..v5: Encode4(memberIds[i]) @0x83d1b3 → recipient ids (uint32 LE each).
//	COutPacket::EncodeStr(a3)           @0x83d1d1 → chatText string.
//
// v79 is GMS<95 so there is NO leading get_update_time (the v95 prefix); the
// audit report (ChatMulti @0x83cebd) is FlatInvalid/🔍 because it flattens the
// branchy send. The wire shape is proven here. WriteInt = uint32-LE;
// WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden "hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatMulti version=gms_v79 ida=0x83cebd
func TestMultiByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

	// chatType=1, recipients=[100,200,300], chatText="hi".
	// 0x01 | 0x03 | 64 00 00 00 | C8 00 00 00 | 2C 01 00 00 | 02 00 'h' 'i'
	input := Multi{chatType: 1, recipients: []uint32{100, 200, 300}, chatText: "hi"}
	expected := []byte{
		0x01,
		0x03,
		0x64, 0x00, 0x00, 0x00,
		0xC8, 0x00, 0x00, 0x00,
		0x2C, 0x01, 0x00, 0x00,
		0x02, 0x00, 0x68, 0x69,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 multi golden mismatch: got %v want %v", actual, expected)
	}
}

// TestMultiByteOutputV72 pins the gms_v72 MULTI_CHAT (op 0x075) serverbound wire.
//
// IDA-verified send-site (GMS_v72.1_U_DEVM.exe, port 13339) —
// CUIStatusBar::SendGroupMessage @0x7f47a7, send block at LABEL_25:
//
//	COutPacket::COutPacket(117)        @0x7f4a6d → opcode 0x75 (matches registry/template 0x75).
//	COutPacket::Encode1(v47)           @0x7f4a7c → chatType byte (group kind: friend-group/guild=0,
//	                                                friend=1, expedition=2, family=3 per the
//	                                                per-branch v47 sets @0x7f49f3/0x7f490a/0x7f4897/0x7f47f1).
//	COutPacket::Encode1((u8)v5)        @0x7f4a85 → recipient count (v5 = member-id array len).
//	for i in 0..v5: Encode4(memberIds[i]) @0x7f4a9d → recipient ids (uint32 LE each).
//	COutPacket::EncodeStr(chatText)    @0x7f4abb (sub_4160CB(a3) @0x7f4ab3 stages the ZXString) → chatText.
//
// v72 is GMS<95 so there is NO leading get_update_time (the v95 prefix); the codec's
// hasUpdateTime gate is GMS>=95, so v72 takes the no-prefix path — byte-identical to
// v79 (only the opcode shifts 0x74→0x75). WriteInt = uint32-LE; WriteAsciiString =
// uint16-LE length + ASCII bytes (admin_chat golden "hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatMulti version=gms_v72 ida=0x7f47a7
func TestMultiByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)

	// chatType=1, recipients=[100,200,300], chatText="hi".
	// 0x01 | 0x03 | 64 00 00 00 | C8 00 00 00 | 2C 01 00 00 | 02 00 'h' 'i'
	input := Multi{chatType: 1, recipients: []uint32{100, 200, 300}, chatText: "hi"}
	expected := []byte{
		0x01,
		0x03,
		0x64, 0x00, 0x00, 0x00,
		0xC8, 0x00, 0x00, 0x00,
		0x2C, 0x01, 0x00, 0x00,
		0x02, 0x00, 0x68, 0x69,
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 multi golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMultiRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Multi{chatType: 1, recipients: []uint32{100, 200, 300}, chatText: "party chat"}
			output := Multi{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ChatType() != input.ChatType() {
				t.Errorf("chatType: got %v, want %v", output.ChatType(), input.ChatType())
			}
			if len(output.Recipients()) != len(input.Recipients()) {
				t.Fatalf("recipients length: got %v, want %v", len(output.Recipients()), len(input.Recipients()))
			}
			for i, r := range output.Recipients() {
				if r != input.Recipients()[i] {
					t.Errorf("recipients[%d]: got %v, want %v", i, r, input.Recipients()[i])
				}
			}
			if output.ChatText() != input.ChatText() {
				t.Errorf("chatText: got %v, want %v", output.ChatText(), input.ChatText())
			}
		})
	}
}

func TestMultiUpdateTimeGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := Multi{updateTime: 0x11223344, chatType: 1, recipients: []uint32{7}, chatText: "hi"}
	// GMS v95: leading 4-byte updateTime little-endian.
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if !bytes.Equal(b95[:4], []byte{0x44, 0x33, 0x22, 0x11}) {
		t.Errorf("v95 leading updateTime = % x, want 44 33 22 11", b95[:4])
	}
	// GMS v87: NO updateTime → first byte is chatType.
	b87 := in.Encode(l, pt.CreateContext("GMS", 87, 1))(nil)
	if b87[0] != 0x01 {
		t.Errorf("v87 first byte = 0x%02x, want chatType 0x01", b87[0])
	}
	// GMS v83: NO updateTime.
	b83 := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if b83[0] != 0x01 {
		t.Errorf("v83 first byte = 0x%02x, want chatType 0x01", b83[0])
	}
	// JMS185: NO updateTime.
	bj := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	if bj[0] != 0x01 {
		t.Errorf("JMS first byte = 0x%02x, want chatType 0x01", bj[0])
	}
	// Round-trip every variant.
	for _, v := range pt.Variants {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		out := Multi{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	}
}
