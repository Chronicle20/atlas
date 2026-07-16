package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v79 NPC conversation detail byte fixtures. The v79 conversation dispatcher is
// CScriptMan::OnScriptMessage@0x6c7d3e (GMS_v79_1_DEVM.exe, port 13340): Decode1
// speakerTypeId, Decode4 speakerTemplateId, Decode1 msgType, Decode1 param, then
// a per-msgType handler reads the detail body ([+ Decode4 secondaryNpcTemplateId
// at the head of that handler when param&4]).
//
// IMPORTANT: the v79 {AskYesNo, AskBoxText, AskNumber} handler IDA symbols are a
// clean 3-cycle rotation (symbol OnAskYesNo@0x6c81b1 reads the box-text body,
// OnAskBoxText@0x6c8525 reads the number body, OnAskNumber@0x6c86d3 reads the
// yes/no body). The decompiled BODY is ground truth; each fixture below cites the
// address of the handler that actually reads that body.
//
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v79 ida=0x6c7d3e
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=gms_v79 ida=0x6c7ed1
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=gms_v79 ida=0x6c8052
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v79 ida=0x6c86d3
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=gms_v79 ida=0x6c836c
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v79 ida=0x6c8525
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=gms_v79 ida=0x6c81b1
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=gms_v79 ida=0x970baa
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=gms_v79 ida=0x970d07
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v79 ida=0x6c8a31
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v79 ida=0x6c8e82

func le16(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }
func le32(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }
func le64(v uint64) []byte { b := make([]byte, 8); binary.LittleEndian.PutUint64(b, v); return b }

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// NpcNpcConversation frame: Decode1 speakerTypeId, Decode4 speakerTemplateId,
// Decode1 msgType, Decode1 param, [Decode4 secondaryNpcTemplateId if param&4],
// detail body (CScriptMan::OnScriptMessage @0x6c7d3e, refs OnSay@0x6c7ef3 for the
// param&4 secondary read).
func TestNpcConversationFrameV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)

	// param=0: no secondary. speakerType=0, templateId=2100, msgType=0, param=0.
	frame := NewNpcConversation(0, 2100, 0, 0, 0, []byte{0xAA, 0xBB})
	got := frame.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, le32(2100), []byte{0x00, 0x00}, []byte{0xAA, 0xBB})
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 frame (param 0): got % x, want % x", got, want)
	}

	// param=4: secondaryNpcTemplateId int is inserted before the detail.
	frame2 := NewNpcConversation(1, 2100, 7, 4, 9999, []byte{0xCC})
	got2 := frame2.Encode(l, ctx)(nil)
	want2 := cat([]byte{0x01}, le32(2100), []byte{0x07, 0x04}, le32(9999), []byte{0xCC})
	if !bytes.Equal(got2, want2) {
		t.Fatalf("v79 frame (param 4): got % x, want % x", got2, want2)
	}
}

// SayConversationDetail: DecodeStr message, Decode1 previous, Decode1 next
// (CScriptMan::OnSay @0x6c7ed1).
func TestNpcConversationSayV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &SayConversationDetail{Message: "Hi", Previous: false, Next: true}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Hi"), []byte{0x00, 0x01})
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 Say: got % x, want % x", got, want)
	}
}

// SayImageConversationDetail: v79 OnSayImage @0x6c8052 reads a SINGLE DecodeStr
// (one image, no count). v83+/JMS read Decode1 count + count x DecodeStr.
func TestNpcConversationSayImageV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &SayImageConversationDetail{Images: []string{"Effect/x.0", "Effect/x.1"}}
	got := detail.Encode(l, ctx)(nil)
	// legacy: first image only, no count byte.
	want := asciiBytes("Effect/x.0")
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 SayImage: got % x, want % x", got, want)
	}
	// v83 still emits count + list: 0x02 + str + str.
	v83 := detail.Encode(l, test.CreateContext("GMS", 83, 1))(nil)
	wantV83 := cat([]byte{0x02}, asciiBytes("Effect/x.0"), asciiBytes("Effect/x.1"))
	if !bytes.Equal(v83, wantV83) {
		t.Fatalf("v83 SayImage should be count+list: got % x, want % x", v83, wantV83)
	}
}

// AskYesNoConversationDetail: DecodeStr message only. Read by the v79 msgType-4
// arm @0x6c86d3 (IDA symbol OnAskNumber; 3-cycle rotation).
func TestNpcConversationAskYesNoV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskYesNoConversationDetail{Message: "Proceed?"}
	got := detail.Encode(l, ctx)(nil)
	want := asciiBytes("Proceed?")
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskYesNo: got % x, want % x", got, want)
	}
}

// AskTextConversationDetail: DecodeStr message, DecodeStr default, Decode2 min,
// Decode2 max (CScriptMan::OnAskText @0x6c836c).
func TestNpcConversationAskTextV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskTextConversationDetail{Message: "Name?", Def: "abc", Min: 4, Max: 12}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Name?"), asciiBytes("abc"), le16(4), le16(12))
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskText: got % x, want % x", got, want)
	}
}

// AskNumberConversationDetail: DecodeStr message, Decode4 default, Decode4 min,
// Decode4 max. Read by the v79 msgType-3 arm @0x6c8525 (IDA symbol OnAskBoxText;
// 3-cycle rotation).
func TestNpcConversationAskNumberV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskNumberConversationDetail{Message: "How many?", Def: 1, Min: 0, Max: 99}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("How many?"), le32(1), le32(0), le32(99))
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskNumber: got % x, want % x", got, want)
	}
}

// AskBoxTextConversationDetail: DecodeStr message, DecodeStr default, Decode2
// col, Decode2 line. Read by the v79 msgType-2 arm @0x6c81b1 (IDA symbol
// OnAskYesNo; 3-cycle rotation).
func TestNpcConversationAskBoxTextV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskBoxTextConversationDetail{Message: "Note", Def: "hello", Col: 40, Line: 6}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Note"), asciiBytes("hello"), le16(40), le16(6))
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskBoxText: got % x, want % x", got, want)
	}
}

// AskQuizConversationDetail: Decode1 fail; if !fail DecodeStr title, DecodeStr
// problem, DecodeStr hint, Decode4 min, Decode4 max, Decode4 timeRemaining
// (sub_970BAA @0x970baa via dispatcher case 5).
func TestNpcConversationAskQuizV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskQuizConversationDetail{Fail: false, Title: "T", Problem: "P", Hint: "H", Min: 1, Max: 5, TimeRemaining: 30}
	got := detail.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, asciiBytes("T"), asciiBytes("P"), asciiBytes("H"), le32(1), le32(5), le32(30))
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskQuiz: got % x, want % x", got, want)
	}
}

// AskSpeedQuizConversationDetail: Decode1 fail; if !fail Decode4 type, answer,
// correct, remain, timeRemaining (sub_970D07 @0x970d07 via dispatcher case 6).
func TestNpcConversationAskSpeedQuizV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskSpeedQuizConversationDetail{Fail: false, Type: 2, Answer: 3, Correct: 4, Remain: 5, TimeRemaining: 60}
	got := detail.Encode(l, ctx)(nil)
	want := cat([]byte{0x00}, le32(2), le32(3), le32(4), le32(5), le32(60))
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskSpeedQuiz: got % x, want % x", got, want)
	}
}

// AskAvatarConversationDetail: DecodeStr message, Decode1 count, count x Decode4
// style id (CScriptMan::OnAskAvatar @0x6c8a31).
func TestNpcConversationAskAvatarV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskAvatarConversationDetail{Message: "Style?", Styles: []uint32{0x11223344, 0x55667788}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Style?"), []byte{0x02}, le32(0x11223344), le32(0x55667788))
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskAvatar: got % x, want % x", got, want)
	}
}

// AskPetAllConversationDetail: DecodeStr message, Decode1 count, Decode1
// exceptionExists, count x (DecodeBuffer(8) cash sn + Decode1). The v79 pet
// handler @0x6c8e82 (dispatcher case 10) is unified and always reads the
// exception byte, matching AskPetAll.
func TestNpcConversationAskPetAllV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	detail := &AskPetAllConversationDetail{Message: "Pet?", ExceptionExists: true, CashIds: []uint64{0x0102030405060708}}
	got := detail.Encode(l, ctx)(nil)
	want := cat(asciiBytes("Pet?"), []byte{0x01, 0x01}, le64(0x0102030405060708), []byte{0x00})
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskPetAll: got % x, want % x", got, want)
	}
}
