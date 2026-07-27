package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 MULTI_CHAT (serverbound op 0x6B / 107) wire verification —
// CUIStatusBar::SendGroupMessage @0x74467d (GMS_v61.1_U_DEVM.exe, port 13338),
// send block at LABEL_21:
//
//	COutPacket::COutPacket(v24, 107)   @0x7448bc → opcode 0x6B (matches registry op 107
//	                                                and template handler 0x6B).
//	COutPacket::Encode1(v31)           @0x7448cb → chatType byte (group kind: v31 set per
//	                                                branch — friend-group/guild=0 @0x744852,
//	                                                friend=1 @0x7447d5, expedition=2 @0x744764,
//	                                                family=3 @0x7446c0).
//	COutPacket::Encode1((u8)v5)        @0x7448d4 → recipient count (v5 = member-id array len).
//	for i in 0..v5: Encode4(memberIds[i]) @0x7448ec → recipient ids (uint32 LE each).
//	sub_414F76(a3) @0x744902 stages the chat text; COutPacket::EncodeStr @0x74490a → chatText.
//
// v61 is GMS<95 so there is NO leading get_update_time prefix — the codec's
// hasUpdateTime gate is GMS>=95, so v61 takes the no-prefix path. The wire is
// byte-identical to the IDA-verified gms_v72 SendGroupMessage @0x7f47a7
// (TestMultiByteOutputV72); only the opcode shifts (0x6B vs 0x75), which is not
// part of the encoded body. WriteInt = uint32-LE; WriteAsciiString = uint16-LE
// length + ASCII ("hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatMulti version=gms_v61 ida=0x74467d
func TestMultiV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)

	// chatType=1, recipients=[100,200,300], chatText="hi".
	// 0x01 | 0x03 | 64 00 00 00 | C8 00 00 00 | 2C 01 00 00 | 02 00 'h' 'i'
	input := Multi{chatType: 1, recipients: []uint32{100, 200, 300}, chatText: "hi"}
	want := []byte{
		0x01,
		0x03,
		0x64, 0x00, 0x00, 0x00,
		0xC8, 0x00, 0x00, 0x00,
		0x2C, 0x01, 0x00, 0x00,
		0x02, 0x00, 0x68, 0x69,
	}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 multi golden mismatch: got % x want % x", got, want)
	}
}

// v61 WHISPER (serverbound op 0x6C / 108) wire verification —
// the whisper send-site is unnamed in the v61 IDB: sub_4E8635 @0x4e8635
// (GMS_v61.1_U_DEVM.exe, port 13338); the v72 twin is CField::SendChatMsgWhisper.
// Send block:
//
//	COutPacket::COutPacket(v12, 108)   @0x4e86e0 → opcode 0x6C (matches registry op 108
//	                                                and template handler 0x6C).
//	v9 = (*a2 == 0) || !**a2           @0x4e86fc → "message empty" flag.
//	COutPacket::Encode1((!v9 + 1) | 4) @0x4e870a → mode byte: non-empty msg → (1+1)|4 = 6
//	                                                (Chat); empty → (0+1)|4 = 5 (Find).
//	sub_414F76(a3) stages target; COutPacket::EncodeStr @0x4e8722 → targetName.
//	if ( *a2 && **a2 ) {               @0x4e872d (non-empty message)
//	    sub_414F76(a2); COutPacket::EncodeStr @0x4e8742 → msg.
//	}
//
// v61 is GMS<87 so there is NO get_update_time prefix — the codec's
// whisperHasUpdateTime gate is (GMS>=87 || JMS), so v61 takes the no-prefix
// path. WhisperMode: Find=5 (no msg), Chat=6 (carries msg). The wire is
// byte-identical to the IDA-verified gms_v83 whisper (TestWhisperByteOutput);
// only the opcode differs (0x6C vs 0x78), which is not part of the encoded body.
// WriteAsciiString = uint16-LE length + ASCII ("Bob"=03 00 42 6F 62, "hi"=02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v61 ida=0x4e8635
func TestWhisperV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)

	// Find mode (5): Encode1(mode) + EncodeStr("Bob") — empty message path.
	// 0x05 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, targetName: "Bob"}
		want := []byte{0x05, 0x03, 0x00, 0x42, 0x6F, 0x62}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v61 whisper find: got % x want % x", got, want)
		}
	})

	// Chat mode (6): Encode1(mode) + EncodeStr("Bob") + EncodeStr("hi").
	// 0x06 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, targetName: "Bob", msg: "hi"}
		want := []byte{0x06, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v61 whisper chat: got % x want % x", got, want)
		}
	})
}
