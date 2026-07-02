package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v61 NPC conversation detail byte fixtures. The v61 conversation dispatcher is
// CScriptMan::OnScriptMessage@0x63f9ca (GMS_v61.1_U_DEVM.exe, port 13338):
// Decode1 speakerTypeId, Decode4 speakerTemplateId, then a Decode1 msgType
// 15-case switch — exactly like v72, with NO param byte and NO param&4
// secondaryNpcTemplateId at the frame level (v61 is below the v79 param gate,
// same legacy branch as v72). Every v61 detail body read order is byte-identical
// to the verified v72 read order, INCLUDING the same 3-cycle IDA symbol rotation
// on {AskYesNo, AskBoxText, AskNumber}: symbol OnAskYesNo@0x63fddf reads the
// box-text body (str,str,short,short), symbol OnAskBoxText@0x640107 reads the
// number body (str,int,int,int), symbol OnAskNumber@0x640269 reads the yes/no
// body (str only). Each fixture cites the address of the handler that actually
// reads that body. The v61 msgType table matches v72 (0=Say,1=SayImage,
// 2=AskYesNo-sym,3=AskBoxText-sym,4=AskNumber-sym,5=Quiz,6=SpeedQuiz,7=AskMenu,
// 8=AskAvatar,9=MembershopAvatar,0xA=AskPet,0xC/D=SayImage,0xE=AskText), so the
// conversation message-type table is carried correctly from v72 — no divergence.
//
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v61 ida=0x63f9ca
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=gms_v61 ida=0x63fb44
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=gms_v61 ida=0x63fc8f
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v61 ida=0x640269
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=gms_v61 ida=0x63ff86
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v61 ida=0x640107
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=gms_v61 ida=0x63fddf
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=gms_v61 ida=0x6403bc
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v61 ida=0x640554
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=gms_v61 ida=0x84816e
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=gms_v61 ida=0x8482cb
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v61 ida=0x640961

// NpcNpcConversation frame: v61 reads Decode1 speakerTypeId, Decode4
// speakerTemplateId, Decode1 msgType, detail body — NO param byte / secondary
// (CScriptMan::OnScriptMessage @0x63f9ca).
func TestNpcConversationFrameV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)

	// param arg = 0: v61 frame is type + templateId + msgType + detail (no param).
	frame := NewNpcConversation(0, 2100, 0, 0, 0, []byte{0xAA, 0xBB})
	got := frame.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, le32(2100), []byte{0x00}, []byte{0xAA, 0xBB})
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 frame (param 0): got % x, want % x", got, want)
	}

	// param arg = 4 (would trigger the secondary in v79): v61 still omits BOTH
	// the param byte and the secondary int.
	frame2 := NewNpcConversation(1, 2100, 7, 4, 9999, []byte{0xCC})
	got2 := frame2.Encode(l, ctx)(nil)
	want2 := cat([]byte{0x01}, le32(2100), []byte{0x07}, []byte{0xCC})
	if !bytes.Equal(got2, want2) {
		t.Fatalf("v61 frame (param 4 omitted): got % x, want % x", got2, want2)
	}
}

// SayConversationDetail: DecodeStr message, Decode1 previous, Decode1 next
// (CScriptMan::OnSay @0x63fb44: DecodeStr, Decode1, Decode1).
func TestNpcConversationSayV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &SayConversationDetail{Message: "Hi", Previous: false, Next: true}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Hi"), []byte{0x00, 0x01})
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 Say: got % x, want % x", got, want)
	}
}

// SayImageConversationDetail: v61 OnSayImage @0x63fc8f reads a SINGLE DecodeStr
// (one image, no count) — matches the legacy branch (<83).
func TestNpcConversationSayImageV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &SayImageConversationDetail{Images: []string{"Effect/x.0", "Effect/x.1"}}
	got := detail.Encode(l, ctx)(nil)
	want := asciiBytes("Effect/x.0")
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 SayImage: got % x, want % x", got, want)
	}
}

// AskYesNoConversationDetail: DecodeStr message only. Read by the v61 msgType-4
// arm @0x640269 (IDA symbol OnAskNumber; 3-cycle rotation, same as v72).
func TestNpcConversationAskYesNoV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskYesNoConversationDetail{Message: "Proceed?"}
	got := detail.Encode(l, ctx)(nil)
	want := asciiBytes("Proceed?")
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskYesNo: got % x, want % x", got, want)
	}
}

// AskTextConversationDetail: DecodeStr message, DecodeStr default, Decode2 min,
// Decode2 max (CScriptMan::OnAskText @0x63ff86, msgType 0xE).
func TestNpcConversationAskTextV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskTextConversationDetail{Message: "Name?", Def: "abc", Min: 4, Max: 12}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Name?"), asciiBytes("abc"), le16(4), le16(12))
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskText: got % x, want % x", got, want)
	}
}

// AskNumberConversationDetail: DecodeStr message, Decode4 default, Decode4 min,
// Decode4 max. Read by the v61 msgType-3 arm @0x640107 (IDA symbol OnAskBoxText;
// 3-cycle rotation).
func TestNpcConversationAskNumberV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskNumberConversationDetail{Message: "How many?", Def: 1, Min: 0, Max: 99}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("How many?"), le32(1), le32(0), le32(99))
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskNumber: got % x, want % x", got, want)
	}
}

// AskBoxTextConversationDetail: DecodeStr message, DecodeStr default, Decode2
// col, Decode2 line. Read by the v61 msgType-2 arm @0x63fddf (IDA symbol
// OnAskYesNo; 3-cycle rotation).
func TestNpcConversationAskBoxTextV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskBoxTextConversationDetail{Message: "Note", Def: "hello", Col: 40, Line: 6}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Note"), asciiBytes("hello"), le16(40), le16(6))
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskBoxText: got % x, want % x", got, want)
	}
}

// AskMenuConversationDetail: DecodeStr message, Decode1 count, count x Decode4
// style (CScriptMan::OnAskMenu @0x6403bc, msgType 7). Atlas uses count 0 for
// the legacy branch (<83).
func TestNpcConversationAskMenuV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskMenuConversationDetail{Message: "Pick"}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pick"), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskMenu: got % x, want % x", got, want)
	}
}

// AskAvatarConversationDetail: DecodeStr message, Decode1 count, count x Decode4
// style id (CScriptMan::OnAskAvatar @0x640554, msgType 8).
func TestNpcConversationAskAvatarV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskAvatarConversationDetail{Message: "Style?", Styles: []uint32{0x11223344, 0x55667788}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Style?"), []byte{0x02}, le32(0x11223344), le32(0x55667788))
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskAvatar: got % x, want % x", got, want)
	}
}

// AskQuizConversationDetail: Decode1 fail; if !fail DecodeStr title, DecodeStr
// problem, DecodeStr hint, Decode4 min, Decode4 max, Decode4 timeRemaining
// (sub_84816E @0x84816e via dispatcher case 5 -> sub_640BE7).
func TestNpcConversationAskQuizV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskQuizConversationDetail{Fail: false, Title: "T", Problem: "P", Hint: "H", Min: 1, Max: 5, TimeRemaining: 30}
	got := detail.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, asciiBytes("T"), asciiBytes("P"), asciiBytes("H"), le32(1), le32(5), le32(30))
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskQuiz: got % x, want % x", got, want)
	}
}

// AskSpeedQuizConversationDetail: Decode1 fail; if !fail Decode4 type, answer,
// correct, remain, timeRemaining (sub_8482CB @0x8482cb via dispatcher case 6 ->
// sub_640BF9).
func TestNpcConversationAskSpeedQuizV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskSpeedQuizConversationDetail{Fail: false, Type: 2, Answer: 3, Correct: 4, Remain: 5, TimeRemaining: 60}
	got := detail.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, le32(2), le32(3), le32(4), le32(5), le32(60))
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskSpeedQuiz: got % x, want % x", got, want)
	}
}

// AskPetAllConversationDetail: DecodeStr message, Decode1 count, Decode1
// exceptionExists, count x (DecodeBuffer(8) cash sn + Decode1). The v61 pet
// handler CScriptMan::OnAskPet @0x640961 (dispatcher case 0xA) reads the
// exception byte, matching AskPetAll.
func TestNpcConversationAskPetAllV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	detail := &AskPetAllConversationDetail{Message: "Pet?", ExceptionExists: true, CashIds: []uint64{0x0102030405060708}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pet?"), []byte{0x01, 0x01}, le64(0x0102030405060708), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 AskPetAll: got % x, want % x", got, want)
	}
}
