package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v72 NPC conversation detail byte fixtures. The v72 conversation dispatcher is
// CScriptMan::OnScriptMessage@0x6a0ba9 (GMS_v72.1_U_DEVM.exe, port 13339):
// Decode1 speakerTypeId, Decode4 speakerTemplateId, Decode1 msgType, then a
// per-msgType handler reads the detail body. UNLIKE v79 there is NO param byte
// and NO param&4 secondaryNpcTemplateId at the frame level (verified from the
// dispatcher disasm: exactly Decode1/Decode4/Decode1 before the 15-case switch;
// the OnSay/OnAsk* `param` arg is passed uninitialised).
//
// The v72 detail bodies are byte-identical to the verified v79 read orders,
// including the SAME 3-cycle IDA symbol rotation on {AskYesNo, AskBoxText,
// AskNumber}: symbol OnAskYesNo@0x6a0fbb reads the box-text body (str,str,
// short,short), symbol OnAskBoxText@0x6a12e1 reads the number body (str,int,
// int,int), symbol OnAskNumber@0x6a1442 reads the yes/no body (str only). Each
// fixture cites the address of the handler that actually reads that body.
//
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v72 ida=0x6a0ba9
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=gms_v72 ida=0x6a0d23
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=gms_v72 ida=0x6a0e6d
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v72 ida=0x6a1442
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=gms_v72 ida=0x6a1161
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v72 ida=0x6a12e1
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=gms_v72 ida=0x6a0fbb
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=gms_v72 ida=0x6a15a6
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v72 ida=0x6a173d
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=gms_v72 ida=0x91ec5f
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=gms_v72 ida=0x91edbc
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v72 ida=0x6a1b47

// NpcNpcConversation frame: v72 reads Decode1 speakerTypeId, Decode4
// speakerTemplateId, Decode1 msgType, detail body — NO param byte / secondary
// (CScriptMan::OnScriptMessage @0x6a0ba9).
func TestNpcConversationFrameV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)

	// param arg = 0: v72 frame is type + templateId + msgType + detail (no param).
	frame := NewNpcConversation(0, 2100, 0, 0, 0, []byte{0xAA, 0xBB})
	got := frame.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, le32(2100), []byte{0x00}, []byte{0xAA, 0xBB})
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 frame (param 0): got % x, want % x", got, want)
	}

	// param arg = 4 (would trigger the secondary in v79): v72 still omits BOTH
	// the param byte and the secondary int.
	frame2 := NewNpcConversation(1, 2100, 7, 4, 9999, []byte{0xCC})
	got2 := frame2.Encode(l, ctx)(nil)
	want2 := cat([]byte{0x01}, le32(2100), []byte{0x07}, []byte{0xCC})
	if !bytes.Equal(got2, want2) {
		t.Fatalf("v72 frame (param 4 omitted): got % x, want % x", got2, want2)
	}
}

// SayConversationDetail: DecodeStr message, Decode1 previous, Decode1 next
// (CScriptMan::OnSay @0x6a0d23).
func TestNpcConversationSayV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &SayConversationDetail{Message: "Hi", Previous: false, Next: true}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Hi"), []byte{0x00, 0x01})
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 Say: got % x, want % x", got, want)
	}
}

// SayImageConversationDetail: v72 OnSayImage @0x6a0e6d reads a SINGLE DecodeStr
// (one image, no count) — matches the legacy branch (<83).
func TestNpcConversationSayImageV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &SayImageConversationDetail{Images: []string{"Effect/x.0", "Effect/x.1"}}
	got := detail.Encode(l, ctx)(nil)
	want := asciiBytes("Effect/x.0")
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 SayImage: got % x, want % x", got, want)
	}
}

// AskYesNoConversationDetail: DecodeStr message only. Read by the v72 msgType-4
// arm @0x6a1442 (IDA symbol OnAskNumber; 3-cycle rotation, same as v79).
func TestNpcConversationAskYesNoV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskYesNoConversationDetail{Message: "Proceed?"}
	got := detail.Encode(l, ctx)(nil)
	want := asciiBytes("Proceed?")
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskYesNo: got % x, want % x", got, want)
	}
}

// AskTextConversationDetail: DecodeStr message, DecodeStr default, Decode2 min,
// Decode2 max (CScriptMan::OnAskText @0x6a1161, msgType 0xE).
func TestNpcConversationAskTextV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskTextConversationDetail{Message: "Name?", Def: "abc", Min: 4, Max: 12}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Name?"), asciiBytes("abc"), le16(4), le16(12))
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskText: got % x, want % x", got, want)
	}
}

// AskNumberConversationDetail: DecodeStr message, Decode4 default, Decode4 min,
// Decode4 max. Read by the v72 msgType-3 arm @0x6a12e1 (IDA symbol OnAskBoxText;
// 3-cycle rotation).
func TestNpcConversationAskNumberV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskNumberConversationDetail{Message: "How many?", Def: 1, Min: 0, Max: 99}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("How many?"), le32(1), le32(0), le32(99))
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskNumber: got % x, want % x", got, want)
	}
}

// AskBoxTextConversationDetail: DecodeStr message, DecodeStr default, Decode2
// col, Decode2 line. Read by the v72 msgType-2 arm @0x6a0fbb (IDA symbol
// OnAskYesNo; 3-cycle rotation).
func TestNpcConversationAskBoxTextV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskBoxTextConversationDetail{Message: "Note", Def: "hello", Col: 40, Line: 6}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Note"), asciiBytes("hello"), le16(40), le16(6))
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskBoxText: got % x, want % x", got, want)
	}
}

// AskMenuConversationDetail: DecodeStr message, Decode1 count, count x Decode4
// style (CScriptMan::OnAskMenu @0x6a15a6, msgType 7). Atlas uses count 0 for
// the legacy branch (<83).
func TestNpcConversationAskMenuV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskMenuConversationDetail{Message: "Pick"}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pick"), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskMenu: got % x, want % x", got, want)
	}
}

// AskAvatarConversationDetail: DecodeStr message, Decode1 count, count x Decode4
// style id (CScriptMan::OnAskAvatar @0x6a173d, msgType 8).
func TestNpcConversationAskAvatarV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskAvatarConversationDetail{Message: "Style?", Styles: []uint32{0x11223344, 0x55667788}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Style?"), []byte{0x02}, le32(0x11223344), le32(0x55667788))
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskAvatar: got % x, want % x", got, want)
	}
}

// AskQuizConversationDetail: Decode1 fail; if !fail DecodeStr title, DecodeStr
// problem, DecodeStr hint, Decode4 min, Decode4 max, Decode4 timeRemaining
// (sub_91EC5F @0x91ec5f via dispatcher case 5).
func TestNpcConversationAskQuizV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskQuizConversationDetail{Fail: false, Title: "T", Problem: "P", Hint: "H", Min: 1, Max: 5, TimeRemaining: 30}
	got := detail.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, asciiBytes("T"), asciiBytes("P"), asciiBytes("H"), le32(1), le32(5), le32(30))
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskQuiz: got % x, want % x", got, want)
	}
}

// AskSpeedQuizConversationDetail: Decode1 fail; if !fail Decode4 type, answer,
// correct, remain, timeRemaining (sub_91EDBC @0x91edbc via dispatcher case 6).
func TestNpcConversationAskSpeedQuizV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskSpeedQuizConversationDetail{Fail: false, Type: 2, Answer: 3, Correct: 4, Remain: 5, TimeRemaining: 60}
	got := detail.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, le32(2), le32(3), le32(4), le32(5), le32(60))
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskSpeedQuiz: got % x, want % x", got, want)
	}
}

// AskPetAllConversationDetail: DecodeStr message, Decode1 count, Decode1
// exceptionExists, count x (DecodeBuffer(8) cash sn + Decode1). The v72 pet
// handler @0x6a1b47 (dispatcher case 0xA) reads the exception byte, matching
// AskPetAll.
func TestNpcConversationAskPetAllV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 72, 1)
	detail := &AskPetAllConversationDetail{Message: "Pet?", ExceptionExists: true, CashIds: []uint64{0x0102030405060708}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pet?"), []byte{0x01, 0x01}, le64(0x0102030405060708), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 AskPetAll: got % x, want % x", got, want)
	}
}
