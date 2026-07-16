package serverbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v79 serverbound NPC conversation/reply byte fixtures (GMS_v79_1_DEVM.exe,
// port 13340). These structs are version-invariant; the v79 client encode sites
// are CUserLocal::TalkToNpc@0x8b7e10 (start) and the COutPacket(58) reply frame
// built by each per-msgType CScriptMan handler.
//
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v79 ida=0x8b7e10
// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=gms_v79 ida=0x6c7ed1
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=gms_v79 ida=0x6c836c
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=gms_v79 ida=0x6c8863

func sle16(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }
func sle32(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }

func sascii(s string) []byte {
	out := make([]byte, 2+len(s))
	binary.LittleEndian.PutUint16(out[:2], uint16(len(s)))
	copy(out[2:], s)
	return out
}

// StartConversation: Encode4 oid, Encode2 x, Encode2 y (CUserLocal::TalkToNpc
// @0x8b7e10).
func TestStartConversationByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	got := StartConversation{oid: 2100, x: -5, y: 200}.Encode(l, ctx)(nil)
	// x=-5 as int16 little-endian = 0xFFFB.
	want := append(append(sle32(2100), 0xFB, 0xFF), sle16(200)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 StartConversation: got % x, want % x", got, want)
	}
}

// ContinueConversation reply frame: Encode1 lastMessageType, Encode1 action
// (COutPacket(58) built by e.g. OnSay@0x6c7ed1).
func TestContinueConversationByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ContinueConversation{lastMessageType: 0, action: 1}.Encode(l, ctx)(nil)
	want := []byte{0x00, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 ContinueConversation: got % x, want % x", got, want)
	}
}

// ContinueConversationText: EncodeStr text (OnAskText@0x6c836c reply).
func TestContinueConversationTextByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ContinueConversationText{text: "hi there"}.Encode(l, ctx)(nil)
	want := sascii("hi there")
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 ContinueConversationText: got % x, want % x", got, want)
	}
}

// ContinueConversationSelection (narrow): Encode1 selection index
// (OnAskMenu@0x6c8863 reply Encode1(v18)).
func TestContinueConversationSelectionByteV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ContinueConversationSelection{selection: 3, wide: false}.Encode(l, ctx)(nil)
	want := []byte{0x03}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 ContinueConversationSelection: got % x, want % x", got, want)
	}
}
