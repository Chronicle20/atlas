package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 MULTI_CHAT (serverbound op 89 / 0x59) wire verification —
// the send-site is UNNAMED in the v48 IDB: sub_65EB4F @0x65eb4f
// (GMS_v48_1_DEVM.exe, port 13337); the v61 twin is CUIStatusBar::SendGroupMessage.
// Send block (after the chatType demux 0=buddy/1=party/2=guild target lists):
//
//	COutPacket::COutPacket(v15, 89)    @0x65ed0e → opcode 0x59 (matches registry op 89
//	                                                and template handler 0x59).
//	COutPacket::Encode1(v21)           @0x65ed1d → chatType byte (0 buddy / 1 party / 2 guild).
//	COutPacket::Encode1(v4)            @0x65ed26 → recipient count.
//	for i in 0..v4: Encode4(memberIds[i]) @0x65ed3e → recipient ids (uint32 LE each).
//	COutPacket::EncodeStr(chatText)    @0x65ed5c → chatText.
//
// v48 is GMS<95 so there is NO leading get_update_time prefix — the codec's
// hasUpdateTime gate is GMS>=95, so v48 takes the no-prefix path. The wire is
// byte-identical to the IDA-verified gms_v61 SendGroupMessage @0x74467d
// (TestMultiV61Body); only the opcode shifts (0x59 vs 0x6B), which is not part
// of the encoded body. WriteInt = uint32-LE; WriteAsciiString = uint16-LE length
// + ASCII ("hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatMulti version=gms_v48 ida=0x65eb4f
func TestMultiV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)

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
		t.Errorf("v48 multi golden mismatch: got % x want % x", got, want)
	}
}

// v48 WHISPER (serverbound op 90 / 0x5A) wire verification —
// the whisper send-site is unnamed in the v48 IDB: sub_4C4F3B @0x4c4f3b
// (GMS_v48_1_DEVM.exe, port 13337); the v61 twin is sub_4E8635, the v72+ twin is
// CField::SendChatMsgWhisper. Send block:
//
//	COutPacket::COutPacket(v14, 90)    @0x4c4ff2 → opcode 0x5A (matches registry op 90
//	                                                and template handler 0x5A).
//	v11 = (*a2 == 0) || !**a2          @0x4c500e → "message empty" flag.
//	COutPacket::Encode1((!v11 + 1) | 4) @0x4c501c → mode byte: non-empty msg → (1+1)|4 = 6
//	                                                (Chat); empty → (0+1)|4 = 5 (Find).
//	COutPacket::EncodeStr(target)      @0x4c5034 → targetName.
//	if ( *v8 && **v8 ) {               @0x4c503f (non-empty message)
//	    COutPacket::EncodeStr(msg)     @0x4c5054 → msg.
//	}
//
// v48 is GMS<87 so there is NO get_update_time prefix — the codec's
// whisperHasUpdateTime gate is (GMS>=87 || JMS), so v48 takes the no-prefix
// path. WhisperMode: Find=5 (no msg), Chat=6 (carries msg). The wire is
// byte-identical to the IDA-verified gms_v61 whisper (TestWhisperV61Body); only
// the opcode differs (0x5A vs 0x6C), which is not part of the encoded body.
// WriteAsciiString = uint16-LE length + ASCII ("Bob"=03 00 42 6F 62, "hi"=02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v48 ida=0x4c4f3b
func TestWhisperV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)

	// Find mode (5): Encode1(mode) + EncodeStr("Bob") — empty message path.
	// 0x05 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, targetName: "Bob"}
		want := []byte{0x05, 0x03, 0x00, 0x42, 0x6F, 0x62}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v48 whisper find: got % x want % x", got, want)
		}
	})

	// Chat mode (6): Encode1(mode) + EncodeStr("Bob") + EncodeStr("hi").
	// 0x06 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, targetName: "Bob", msg: "hi"}
		want := []byte{0x06, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v48 whisper chat: got % x want % x", got, want)
		}
	})
}
