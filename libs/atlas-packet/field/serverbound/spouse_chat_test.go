package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestCoupleMessageByteOutput pins the gms_v83 SPOUSE_CHAT (op 0x079) serverbound wire.
//
// IDA-verified send-site (GMS v83 retail dump, port 13342):
//
//	CUIStatusBar::SendCoupleMessage @0x8defff:
//	  GetCharacterData @0x8df051; marriage-record guard `if (*(v7 + 1367))` @0x8df062 (married flag);
//	  COutPacket(0x79) @0x8df08e; EncodeStr(spouseName) @0x8df0ab (partner name from the local
//	  marriage record buffer); EncodeStr(message=a1) @0x8df0c3. The not-married else-branch
//	  (@0x8df0e6) is a local StringPool::GetString(SP_159_...NOT_MARRIED...) + ChatLogAdd — it
//	  sends no packet. No leading mode byte, no get_update_time prefix.
//
// WriteAsciiString = uint16-LE length + ASCII bytes (see admin_chat_test golden "hi"=02 00 68 69).
//
// packet-audit:verify packet=field/serverbound/FieldCoupleMessage version=gms_v83 ida=0x8defff
func TestCoupleMessageByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)

	// EncodeStr("Bob") + EncodeStr("hi")
	// 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	input := CoupleMessage{spouseName: "Bob", message: "hi"}
	expected := []byte{0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestCoupleMessageByteOutputV79 pins the gms_v79 SPOUSE_CHAT (op 0x76) serverbound wire.
//
// IDA-verified send-site (GMS_v79_1_DEVM.exe, port 13340):
//
//	CUIStatusBar::SendCoupleMessage (sub_83CD67 @0x83cd67):
//	  GetCharacterData @0x83cdb9; marriage-record guard `if (*(v6 + 1291))` @0x83cdca
//	  (married flag); COutPacket(118)=0x76 @0x83cdf6; EncodeStr(spouseName) @0x83ce13
//	  (partner name from the local marriage record buffer); EncodeStr(message=a2)
//	  @0x83ce2b. The not-married else-branch (@0x83ce4e) is StringPool::GetString(159)
//	  + ChatLogAdd — it sends no packet. No leading mode byte, no get_update_time prefix.
//
// Wire byte-identical to v83 (only the opcode shifts 0x79→0x76); codec is opcode-agnostic.
//
// packet-audit:verify packet=field/serverbound/FieldCoupleMessage version=gms_v79 ida=0x83cd67
func TestCoupleMessageByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := CoupleMessage{spouseName: "Bob", message: "hi"}
	expected := []byte{0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 couple_message golden mismatch: got %v want %v", actual, expected)
	}
}

// TestCoupleMessageByteOutputV84 pins the gms_v84 SPOUSE_CHAT (op 0x07B) serverbound wire.
//
// IDA-verified send-site (GMS_v84.1_U_DEVM, port 13337):
//
//	CUIStatusBar::SendCoupleMessage @0x9145ee:
//	  GetCharacterData via sub_4267FF(v15) @0x914640 (v6 = *(ret+4)); marriage-record
//	  guard `if (*(v6 + 1367))` @0x914651 (married flag); COutPacket(123)=0x7B @0x91467d;
//	  EncodeStr(spouseName=v19) @0x91469a (partner name released from the local marriage
//	  record buffer @0x914673); EncodeStr(message=a2) @0x9146b2. The not-married
//	  else-branch (@0x9146d5) is StringPool::GetString(159) @0x9146f2 + ChatLogAdd
//	  (sub_90FE17) — it sends no packet. No leading mode byte, no get_update_time prefix.
//
// WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden "hi"=02 00 68 69).
// Wire byte-identical to gms_v83 (only the opcode shifts 0x79→0x7B); the codec is
// opcode-agnostic so the encoded body is the same.
//
// packet-audit:verify packet=field/serverbound/FieldCoupleMessage version=gms_v84 ida=0x9145ee
func TestCoupleMessageByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)

	// EncodeStr("Bob") + EncodeStr("hi")
	// 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	input := CoupleMessage{spouseName: "Bob", message: "hi"}
	expected := []byte{0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestCoupleMessageByteOutputV87 pins the gms_v87 SPOUSE_CHAT (op 0x07F) serverbound wire.
//
// IDA-verified send-site (GMSv87_4GB.exe, port 13341):
//
//	CUIStatusBar::SendCoupleMessage @0x953c15:
//	  GetCharacterData @0x953c67 (v6); marriage-record guard `if (*(v6 + 1497))` @0x953c78
//	  (married flag); COutPacket(0x7F) @0x953ca4; EncodeStr(spouseName=v20) @0x953cc1
//	  (partner name assigned from the local marriage record @0x953cb9); EncodeStr(message=a2)
//	  @0x953cd9. The not-married else-branch (@0x953cfc) is StringPool::GetString(159)
//	  @0x953d19 + ChatLogAdd (sub_94F43C) @0x953d32 — it sends no packet. No leading mode
//	  byte, no get_update_time prefix.
//
// WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden "hi"=02 00 68 69).
// Wire byte-identical to gms_v83/v84 (only the opcode shifts to 0x7F); the codec is
// opcode-agnostic so the encoded body is the same.
//
// packet-audit:verify packet=field/serverbound/FieldCoupleMessage version=gms_v87 ida=0x953c15
func TestCoupleMessageByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)

	// EncodeStr("Bob") + EncodeStr("hi")
	// 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	input := CoupleMessage{spouseName: "Bob", message: "hi"}
	expected := []byte{0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestCoupleMessageByteOutputV95 pins the gms_v95 SPOUSE_CHAT (op 0x08E) serverbound wire.
//
// IDA-verified send-site (GMS_v95.0_U_DEVM, port 13340):
//
//	CUIStatusBar::SendCoupleMessage @0x87b3e0:
//	  CWvsContext::GetCharacterData @0x87b47f (p); marriage-record guard
//	  `if ( p->lMarriageRecord._m_uCount )` @0x87b4bc (married flag); spouseName =
//	  m_pHead->sBrideName @0x87b4ce, or m_pHead->sGroomName @0x87b4d9 when m_nGender set
//	  (the partner name from the local marriage record); COutPacket::COutPacket(&oPacket,
//	  142)=0x8E @0x87b4ef; EncodeStr(sMarried=spouseName) @0x87b510; EncodeStr(sText=message)
//	  @0x87b52c. The not-married else-branch (@0x87b55e) is StringPool::GetString(0xA1)
//	  @0x87b590 + CUIStatusBar::ChatLogAdd @0x87b596 — it sends no packet. No leading mode
//	  byte, no get_update_time prefix.
//
// WriteAsciiString = uint16-LE length + ASCII bytes (admin_chat golden "hi"=02 00 68 69).
// Wire byte-identical to gms_v83/v84/v87 (only the opcode shifts to 0x8E); the codec is
// opcode-agnostic so the encoded body is the same.
//
// packet-audit:verify packet=field/serverbound/FieldCoupleMessage version=gms_v95 ida=0x87b3e0
func TestCoupleMessageByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)

	// EncodeStr("Bob") + EncodeStr("hi")
	// 0x03 0x00 'B' 'o' 'b' | 0x02 0x00 'h' 'i'
	input := CoupleMessage{spouseName: "Bob", message: "hi"}
	expected := []byte{0x03, 0x00, 0x42, 0x6F, 0x62, 0x02, 0x00, 0x68, 0x69}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestCoupleMessageRoundTrip(t *testing.T) {
	input := CoupleMessage{spouseName: "Bob", message: "hi"}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := CoupleMessage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SpouseName() != input.SpouseName() || output.Message() != input.Message() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
