package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v48 NPC conversation detail byte fixtures (GMS_v48_1_DEVM.exe, port 13337).
// The v48 conversation dispatcher is sub_5B0AE4@0x5b0ae4: Decode1 speakerTypeId,
// Decode4 speakerTemplateId, Decode1 msgType, then a msgType switch with ONLY
// cases 0..9 (narrower than v61's 0..0xE). Each arm is a discrete handler and
// its reply COutPacket(47) echoes the case index, so the v48 messageType table
// is a distinct, older layout (re-derived, NOT carried from v61):
//
//   0 Say            sub_5B0C11@0x5b0c11  DecodeStr, Decode1 prev, Decode1 next
//   1 AskYesNo       sub_5B0D5C@0x5b0d5c  DecodeStr (dialog type 1, yes/no)
//   2 (yes/no variant, a6=1, sub_5B0D5C) — no distinct Atlas struct
//   3 AskText        sub_5B0E90@0x5b0e90  DecodeStr, DecodeStr, Decode2, Decode2
//   4 AskNumber      sub_5B1037@0x5b1037  DecodeStr, Decode4, Decode4, Decode4
//   5 AskMenu        sub_5B1195@0x5b1195  DecodeStr ONLY (menu #L-tokens; NO count)
//   6 AskAvatar      sub_5B12E8@0x5b12e8  DecodeStr, Decode1 count, Decode4 x count
//   7 MembershopAvtr sub_5B1494@0x5b1494  DecodeStr, Decode1 count, Decode4 x count
//   8 AskPet         sub_5B1640@0x5b1640  DecodeStr, Decode1 count, (Buf8+Decode1) x count
//   9 AskPetAll      sub_5B18B5@0x5b18b5  DecodeStr, Decode1 count, Decode1 exc, (Buf8+Decode1)
//
// Genuinely absent in v48 (switch handles only 0..9; default is a no-op):
// SayImage (no image-dialog arm), AskQuiz, AskSpeedQuiz, AskBoxText (only the
// single-line GetText dialog at case 3 exists), AskSlideMenu — dispositioned n-a.
//
// Divergences from the v61 anchor: (a) AskMenu (case 5) reads a single string
// with NO count byte, because in v48 the plain #L menu (dialog type 4) and the
// count-prefixed avatar list (dialog type 5) are SEPARATE arms; the count byte
// is gated to the v61..v82 merged-menu range. (b) The msgType numbering is a
// distinct, older layout, not v61's rotated table.
//
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v48 ida=0x5b0ae4
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=gms_v48 ida=0x5b0c11
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v48 ida=0x5b0d5c
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=gms_v48 ida=0x5b0e90
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v48 ida=0x5b1037
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=gms_v48 ida=0x5b1195
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v48 ida=0x5b12e8
// packet-audit:verify packet=npc/clientbound/NpcAskMemberShopAvatarConversationDetail version=gms_v48 ida=0x5b1494
// packet-audit:verify packet=npc/clientbound/NpcAskPetConversationDetail version=gms_v48 ida=0x5b1640
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v48 ida=0x5b18b5

// NpcNpcConversation frame: v48 sub_5B0AE4 reads Decode1 speakerTypeId, Decode4
// speakerTemplateId, Decode1 msgType, then the detail body — NO param byte / no
// secondary (v48 is below the v79 param gate).
func TestNpcConversationFrameV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)

	frame := NewNpcConversation(0, 2100, 0, 0, 0, []byte{0xAA, 0xBB})
	got := frame.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, le32(2100), []byte{0x00}, []byte{0xAA, 0xBB})
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 frame (param 0): got % x, want % x", got, want)
	}

	// param arg = 4 (would trigger the secondary in v79): v48 still omits BOTH
	// the param byte and the secondary int.
	frame2 := NewNpcConversation(1, 2100, 5, 4, 9999, []byte{0xCC})
	got2 := frame2.Encode(l, ctx)(nil)
	want2 := cat([]byte{0x01}, le32(2100), []byte{0x05}, []byte{0xCC})
	if !bytes.Equal(got2, want2) {
		t.Fatalf("v48 frame (param 4 omitted): got % x, want % x", got2, want2)
	}
}

// Say (case 0, sub_5B0C11): DecodeStr message, Decode1 previous, Decode1 next.
func TestNpcConversationSayV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &SayConversationDetail{Message: "Hi", Previous: false, Next: true}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Hi"), []byte{0x00, 0x01})
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 Say: got % x, want % x", got, want)
	}
}

// AskYesNo (case 1, sub_5B0D5C): DecodeStr message only (yes/no dialog type 1).
func TestNpcConversationAskYesNoV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskYesNoConversationDetail{Message: "Proceed?"}
	got := detail.Encode(l, ctx)(nil)
	want := asciiBytes("Proceed?")
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskYesNo: got % x, want % x", got, want)
	}
}

// AskText (case 3, sub_5B0E90): DecodeStr message, DecodeStr default, Decode2
// min, Decode2 max (GetText dialog type 3).
func TestNpcConversationAskTextV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskTextConversationDetail{Message: "Name?", Def: "abc", Min: 4, Max: 12}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Name?"), asciiBytes("abc"), le16(4), le16(12))
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskText: got % x, want % x", got, want)
	}
}

// AskNumber (case 4, sub_5B1037): DecodeStr message, Decode4 default, Decode4
// min, Decode4 max (GetNumber dialog type 2).
func TestNpcConversationAskNumberV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskNumberConversationDetail{Message: "How many?", Def: 1, Min: 0, Max: 99}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("How many?"), le32(1), le32(0), le32(99))
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskNumber: got % x, want % x", got, want)
	}
}

// AskMenu (case 5, sub_5B1195): DecodeStr message ONLY — v48 emits no count
// byte (menu items are #L-tokens embedded in the string; the count-prefixed
// avatar list is the separate ASK_AVATAR arm).
func TestNpcConversationAskMenuV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskMenuConversationDetail{Message: "Pick"}
	got := detail.Encode(l, ctx)(nil)
	want := asciiBytes("Pick")
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskMenu: got % x, want % x", got, want)
	}
}

// AskAvatar (case 6, sub_5B12E8): DecodeStr message, Decode1 count, count x
// Decode4 style id (SetUtilDlgEx_AVATAR, dialog type 5).
func TestNpcConversationAskAvatarV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskAvatarConversationDetail{Message: "Style?", Styles: []uint32{0x11223344, 0x55667788}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Style?"), []byte{0x02}, le32(0x11223344), le32(0x55667788))
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskAvatar: got % x, want % x", got, want)
	}
}

// AskMemberShopAvatar (case 7, sub_5B1494): DecodeStr message, Decode1 count,
// count x Decode4 style id. Atlas drives no candidates in the legacy range, so
// the encoder emits AsciiString(message) + Byte(0) (count 0). This is the
// representative writer for the v48 NPC_TALK_MORE op row.
func TestNpcConversationAskMemberShopAvatarV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskMemberShopAvatarConversationDetail{Message: "Pick avatar"}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pick avatar"), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskMemberShopAvatar: got % x, want % x", got, want)
	}
}

// AskPet (case 8, sub_5B1640): DecodeStr message, Decode1 count, count x
// (DecodeBuffer(8) cash sn + Decode1) — no exception byte.
func TestNpcConversationAskPetV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskPetConversationDetail{Message: "Pet?", CashId: []uint64{0x0102030405060708}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pet?"), []byte{0x01}, le64(0x0102030405060708), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskPet: got % x, want % x", got, want)
	}
}

// AskPetAll (case 9, sub_5B18B5): DecodeStr message, Decode1 count, Decode1
// exceptionExists, count x (DecodeBuffer(8) cash sn + Decode1).
func TestNpcConversationAskPetAllV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 48, 1)
	detail := &AskPetAllConversationDetail{Message: "Pet?", ExceptionExists: true, CashIds: []uint64{0x0102030405060708}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pet?"), []byte{0x01, 0x01}, le64(0x0102030405060708), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 AskPetAll: got % x, want % x", got, want)
	}
}
