package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestWhisperByteOutput pins the gms_v83 WHISPER (op 0x078) serverbound wire.
//
// IDA-verified send-sites (GMS v83 retail dump, port 13342):
//
//	CField::SendLocationWhisper @0x52f9c6 (find-friend send):
//	  COutPacket(0x78) @0x52fa50; Encode1((bTabFriend==0 ? 1 : 0x40)|4) @0x52fa6? — =5 for a
//	  non-tab find; EncodeStr(sWhisperTarget) @0x52fa83. No message, no get_update_time.
//	CField::SendChatMsgWhisper @0x52f185 (chat send, main path @0x52f8a8):
//	  COutPacket(0x78) @0x52f8a8; Encode1((!msgEmpty + 1)|4) @0x52f8cf — =6 when the
//	  message is non-empty (Chat); EncodeStr(sWhisperTarget) @0x52f8e7; then, gated by
//	  `*v11 && **v11` (non-empty message) @0x52f8f2, EncodeStr(message) @0x52f907.
//
// v83 has NO get_update_time prefix (whisperHasUpdateTime is false for GMS<87), so the
// flat-positional audit report (ChatWhisper) is FlatInvalid/🔍 — the wire shape is the
// mode-dependent layout proven here. WhisperMode: Find=5, Chat=6 (chat carries the msg).
// WriteAsciiString = uint16-LE length + ASCII bytes (see admin_chat_test golden "hi"=02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v83 ida=0x52f9c6
func TestWhisperByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)

	// Find mode (5): Encode1(mode) + EncodeStr("Bob") — SendLocationWhisper @0x52f9c6.
	// 0x05 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, targetName: "Bob"}
		expected := []byte{0x05, 0x03, 0x00, 0x42, 0x6F, 0x62}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("find golden mismatch: got %v want %v", actual, expected)
		}
	})

	// Chat mode (6): Encode1(mode) + EncodeStr("Bob") + EncodeStr("hi") — SendChatMsgWhisper @0x52f8a8.
	// 0x06 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, targetName: "Bob", msg: "hi"}
		expected := []byte{0x06, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("chat golden mismatch: got %v want %v", actual, expected)
		}
	})
}

// TestWhisperByteOutputV84 pins the gms_v84 WHISPER (op 0x07A) serverbound wire.
//
// IDA-verified send-sites (GMS_v84.1_U_DEVM, port 13337):
//
//	CField::SendLocationWhisper @0x53bb1c (find-friend send):
//	  COutPacket(122)=0x7A @0x53bba6; Encode1((a3==0 ? 1 : 64) | 4) @0x53bbb4 — =5
//	  (1|4) for a non-tab find; EncodeStr(target) @0x53bbd9. No message, no
//	  get_update_time (v84 < 87 so whisperHasUpdateTime is false).
//	CField::SendChatMsgWhisper @0x53b2db (chat send, main path @0x53b9fe):
//	  COutPacket(122)=0x7A @0x53b9fe; Encode1((!v47 + 1) | 4) @0x53ba25 — v47 is
//	  "message empty"; for a non-empty message !v47=1 so (1+1)|4 = 6 (Chat);
//	  EncodeStr(target=a3) @0x53ba3d; then, gated by `*v12 && *(_BYTE *)*v12`
//	  (non-empty message) @0x53ba48, EncodeStr(message=v12) @0x53ba5d.
//
// v84 has NO get_update_time prefix, so the flat-positional audit report
// (ChatWhisper @0x53b2db) is FlatInvalid/🔍 — the wire shape is the
// mode-dependent layout proven here. WhisperMode: Find=5, Chat=6 (chat carries
// the msg). WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden
// "hi"=02 00 68 69). Wire byte-identical to gms_v83 (only the opcode shifts
// 0x78→0x7A); the codec is opcode-agnostic so the encoded body is the same.
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v84 ida=0x53bb1c
func TestWhisperByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)

	// Find mode (5): Encode1(mode) + EncodeStr("Bob") — SendLocationWhisper @0x53bb1c.
	// 0x05 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, targetName: "Bob"}
		expected := []byte{0x05, 0x03, 0x00, 0x42, 0x6F, 0x62}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("find golden mismatch: got %v want %v", actual, expected)
		}
	})

	// Chat mode (6): Encode1(mode) + EncodeStr("Bob") + EncodeStr("hi") — SendChatMsgWhisper @0x53b9fe.
	// 0x06 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, targetName: "Bob", msg: "hi"}
		expected := []byte{0x06, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("chat golden mismatch: got %v want %v", actual, expected)
		}
	})
}

// TestWhisperByteOutputV87 pins the gms_v87 WHISPER (op 0x07E) serverbound wire.
//
// IDA-verified send-site (GMSv87_4GB.exe, port 13341):
//
//	CField::SendChatMsgWhisper @0x556385:
//	  COutPacket(0x7E) @0x556ab6 (main send path); Encode1((!v33 + 1) | 4) @0x556add —
//	  v33 is "message empty"; for a non-empty message !v33=1 so (1+1)|4 = 6 (Chat), and
//	  =5 (1|4) when empty (Find/no-msg). get_update_time() @0x556ae2 → Encode4(update_time)
//	  @0x556aeb — the 4-byte updateTime present from v87 (whisperHasUpdateTime true for
//	  GMS>=87). EncodeStr(target=arg4) @0x556b03; then, gated by `*v10 && **v10` (non-empty
//	  message) @0x556b0e, EncodeStr(message=v10) @0x556b23. (The find-loop branch @0x5569c5
//	  emits the identical mode+update_time+target+msg shape per recipient.)
//
// Unlike v83/v84, v87 carries the 4-byte get_update_time between the mode byte and the
// target name — so the fixture inserts updateTime (here 100 = 0x64 00 00 00 LE) at index
// 1. The flat-positional audit report (ChatWhisper @0x556385) is FlatInvalid/🔍 because
// it flattens both send paths (8 reads); the mode-dependent wire shape is proven here.
// WhisperMode: Find=5, Chat=6 (chat carries the msg). WriteInt = uint32-LE;
// WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden "hi"=02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v87 ida=0x556385
func TestWhisperByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)

	// Find mode (5): Encode1(mode) + Encode4(updateTime) + EncodeStr("Bob").
	// 0x05 | 0x64 0x00 0x00 0x00 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, updateTime: 100, targetName: "Bob"}
		expected := []byte{0x05, 0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("find golden mismatch: got %v want %v", actual, expected)
		}
	})

	// Chat mode (6): Encode1(mode) + Encode4(updateTime) + EncodeStr("Bob") + EncodeStr("hi").
	// 0x06 | 0x64 0x00 0x00 0x00 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, updateTime: 100, targetName: "Bob", msg: "hi"}
		expected := []byte{0x06, 0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("chat golden mismatch: got %v want %v", actual, expected)
		}
	})
}

// TestWhisperByteOutputV95 pins the gms_v95 WHISPER (op 0x08D) serverbound wire.
//
// IDA-verified send-site (GMS_v95.0_U_DEVM, port 13340):
//
//	CField::SendLocationWhisper @0x534150 (find-friend send, not-yet-sent branch):
//	  COutPacket::COutPacket(&oPacket, 141)=0x8D @0x53425d; Encode1((bTabFriend != 0 ? 64 : 1) | 4)
//	  @0x53427a — =5 (1|4) for a non-tab find; get_update_time() @0x534284 →
//	  Encode4(update_time) @0x53428e — the 4-byte updateTime present from v87/v95
//	  (whisperHasUpdateTime true for GMS>=87); EncodeStr(v15[0]=target) @0x5342a6.
//	  This find-path emits no message field. The chat-msg sibling
//	  CField::SendChatMsgWhisper emits the identical mode + update_time + target shape
//	  with a trailing EncodeStr(msg) gated on a non-empty message.
//
// v95 carries the 4-byte get_update_time between the mode byte and the target name
// (like v87, unlike v83/v84). The fixture inserts updateTime (here 100 = 0x64 00 00 00
// LE) at index 1. The flat-positional audit report (ChatWhisper @0x534150) is
// FlatInvalid/🔍 because the export captures only the find-path (3 reads:
// mode+updateTime+target) while the codec writes the optional Chat msg as a 4th field;
// the mode-dependent wire shape is proven here. WhisperMode: Find=5, Chat=6 (chat carries
// the msg). WriteInt = uint32-LE; WriteAsciiString = uint16-LE length + ASCII bytes
// (admin_chat golden "hi"=02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v95 ida=0x534150
func TestWhisperByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)

	// Find mode (5): Encode1(mode) + Encode4(updateTime) + EncodeStr("Bob").
	// 0x05 | 0x64 0x00 0x00 0x00 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, updateTime: 100, targetName: "Bob"}
		expected := []byte{0x05, 0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("find golden mismatch: got %v want %v", actual, expected)
		}
	})

	// Chat mode (6): Encode1(mode) + Encode4(updateTime) + EncodeStr("Bob") + EncodeStr("hi").
	// 0x06 | 0x64 0x00 0x00 0x00 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, updateTime: 100, targetName: "Bob", msg: "hi"}
		expected := []byte{0x06, 0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("chat golden mismatch: got %v want %v", actual, expected)
		}
	})
}

// TestWhisperByteOutputJMS pins the jms_v185 WHISPER (op 0x07A) serverbound wire.
//
// IDA-verified send-sites (MapleStory_dump_SCY.exe JMS v185, port 13339):
//
//	CField::SendLocationWhisper @0x56c73d (find-friend send):
//	  COutPacket(0x7A) @0x56c7c7; Encode1((a3 == 0 ? 1 : 64) | 4) @0x56c7d? — =5
//	  (1|4) for a non-tab find; get_update_time() @0x56c7e7 → Encode4(update_time)
//	  @0x56c7f0 — the 4-byte updateTime present in JMS (whisperHasUpdateTime true
//	  for region=="JMS"); EncodeStr(target) @0x56c808. No message field on this path.
//	CField::SendChatMsgWhisper @0x56bf11 (chat send, main path @0x56c60e):
//	  COutPacket(0x7A) @0x56c60e; Encode1((!v47 + 1) | 4) @0x56c635 — v47 is
//	  "message empty"; for a non-empty message !v47=1 so (1+1)|4 = 6 (Chat);
//	  get_update_time() @0x56c63a → Encode4(update_time) @0x56c643; EncodeStr(target=s)
//	  @0x56c65b; then, gated by `*m_pStr && **m_pStr` (non-empty message) @0x56c666,
//	  EncodeStr(message=m_pStr) @0x56c67b.
//
// Like v87/v95 (unlike v83/v84), jms carries the 4-byte get_update_time between the
// mode byte and the target name — the fixture inserts updateTime (here 100 =
// 0x64 00 00 00 LE) at index 1. The flat-positional audit report (ChatWhisper
// @0x56bf11) is FlatInvalid/🔍 because it flattens both send paths (8 reads); the
// mode-dependent wire shape is proven here. WhisperMode: Find=5, Chat=6 (chat carries
// the msg). WriteInt = uint32-LE; WriteAsciiString = uint16-LE length + ASCII bytes
// (admin_chat golden "hi"=02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=jms_v185 ida=0x56c73d
func TestWhisperByteOutputJMS(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)

	// Find mode (5): Encode1(mode) + Encode4(updateTime) + EncodeStr("Bob") — SendLocationWhisper @0x56c73d.
	// 0x05 | 0x64 0x00 0x00 0x00 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, updateTime: 100, targetName: "Bob"}
		expected := []byte{0x05, 0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("find golden mismatch: got %v want %v", actual, expected)
		}
	})

	// Chat mode (6): Encode1(mode) + Encode4(updateTime) + EncodeStr("Bob") + EncodeStr("hi") — SendChatMsgWhisper @0x56c60e.
	// 0x06 | 0x64 0x00 0x00 0x00 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, updateTime: 100, targetName: "Bob", msg: "hi"}
		expected := []byte{0x06, 0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("chat golden mismatch: got %v want %v", actual, expected)
		}
	})
}

// TestWhisperByteOutputV79 pins the gms_v79 WHISPER (op 0x075) serverbound wire.
//
// IDA-verified send-site (GMS_v79_1_DEVM.exe, port 13340) —
// CField::SendChatMsgWhisper @0x51a7bc, main send path @0x51aedf:
//
//	COutPacket::COutPacket(117)=0x75      @0x51aedf;
//	COutPacket::Encode1((!v24 + 1) | 4)   @0x51af06 — v24 is "message empty"; for a
//	  non-empty message !v24=1 so (1+1)|4 = 6 (Chat), and =5 (1|4) when empty (Find);
//	COutPacket::EncodeStr(a3=target)      @0x51af1e;
//	then, gated by `*v10 && **v10` (non-empty message) @0x51af29,
//	COutPacket::EncodeStr(v10=msg)        @0x51af3e.
//
// v79 is GMS<87 so whisperHasUpdateTime is false — there is NO get_update_time
// prefix (byte-identical to v83/v84; only the opcode shifts to 0x75). The audit
// report (ChatWhisper @0x51a7bc) is FlatInvalid/🔍; the mode-dependent wire shape
// is proven here. WhisperMode: Find=5, Chat=6 (chat carries the msg).
// WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden "hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v79 ida=0x51a7bc
func TestWhisperByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

	// Find mode (5): Encode1(mode) + EncodeStr("Bob") — NO updateTime.
	// 0x05 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, targetName: "Bob"}
		expected := []byte{0x05, 0x03, 0x00, 0x42, 0x6F, 0x62}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("find golden mismatch: got %v want %v", actual, expected)
		}
	})

	// Chat mode (6): Encode1(mode) + EncodeStr("Bob") + EncodeStr("hi") — NO updateTime.
	// 0x06 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, targetName: "Bob", msg: "hi"}
		expected := []byte{0x06, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("chat golden mismatch: got %v want %v", actual, expected)
		}
	})
}

// TestWhisperByteOutputV72 pins the gms_v72 WHISPER (op 0x076) serverbound wire.
//
// IDA-verified send-sites (GMS_v72.1_U_DEVM.exe, port 13339):
//
//	CField::SendChatMsgWhisper @0x513743 (chat send, main path @0x513e66):
//	  COutPacket::COutPacket(118)=0x76      @0x513e66;
//	  COutPacket::Encode1((!v24 + 1) | 4)   @0x513e8d — v24 is "message empty"; for a
//	    non-empty message !v24=1 so (1+1)|4 = 6 (Chat), and =5 (1|4) when empty;
//	  COutPacket::EncodeStr(a3=target)      @0x513ea5 (sub_4160CB(a3) @0x513e9d stages it);
//	  then, gated by `*v10 && **v10` (non-empty message) @0x513eb0,
//	  COutPacket::EncodeStr(v10=msg)        @0x513ec5.
//	CField::SendLocationWhisper @0x513f84 (find-friend send, not-yet-sent branch):
//	  COutPacket::COutPacket(118)=0x76      @0x51400e;
//	  COutPacket::Encode1((a3==0 ? 1 : 64) | 4) @0x514024 — =5 (1|4) for a non-tab find;
//	  COutPacket::EncodeStr(a2=target)      @0x514041. No message field on this path.
//
// v72 is GMS<87 so whisperHasUpdateTime is false — there is NO get_update_time
// prefix (byte-identical to v79/v83/v84; only the opcode shifts to 0x76). The audit
// report (ChatWhisper @0x513743) is FlatInvalid/🔍; the mode-dependent wire shape is
// proven here. WhisperMode: Find=5, Chat=6 (chat carries the msg).
// WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden "hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/serverbound/ChatWhisper version=gms_v72 ida=0x513743
func TestWhisperByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)

	// Find mode (5): Encode1(mode) + EncodeStr("Bob") — NO updateTime.
	// 0x05 | 0x03 0x00 'B' 'o' 'b'
	t.Run("find", func(t *testing.T) {
		input := Whisper{mode: WhisperModeFind, targetName: "Bob"}
		expected := []byte{0x05, 0x03, 0x00, 0x42, 0x6F, 0x62}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("find golden mismatch: got %v want %v", actual, expected)
		}
	})

	// Chat mode (6): Encode1(mode) + EncodeStr("Bob") + EncodeStr("hi") — NO updateTime.
	// 0x06 | 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	t.Run("chat", func(t *testing.T) {
		input := Whisper{mode: WhisperModeChat, targetName: "Bob", msg: "hi"}
		expected := []byte{0x06, 0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
		actual := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("chat golden mismatch: got %v want %v", actual, expected)
		}
	})
}

func TestWhisperFindRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Whisper{mode: WhisperModeFind, updateTime: 100, targetName: "SomePlayer"}
			output := Whisper{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
		})
	}
}

func TestWhisperChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Whisper{mode: WhisperModeChat, updateTime: 100, targetName: "SomePlayer", msg: "hello"}
			output := Whisper{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
			if output.Msg() != input.Msg() {
				t.Errorf("msg: got %v, want %v", output.Msg(), input.Msg())
			}
		})
	}
}
