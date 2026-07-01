package serverbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v72 serverbound NPC conversation/reply + action byte fixtures
// (GMS_v72.1_U_DEVM.exe, port 13339). These structs are version-invariant; the
// v72 client encode sites are the COutPacket(59) reply frame built by each
// per-msgType CScriptMan handler (OnSay@0x6a0d23 / OnAskText@0x6a1161 /
// OnAskMenu@0x6a15a6, all verified against the v72 decompile), and the
// NPC_ACTION send CNpc::GenerateMovePath@0x63fc49.
//
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v72 ida=0x70dd49
// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=gms_v72 ida=0x6a0d23
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=gms_v72 ida=0x6a1161
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=gms_v72 ida=0x6a15a6
// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=gms_v72 ida=0x63fc49

// StartConversation: v72 TalkToNpc (sub_70DD49@0x70dd49) sends ONLY Encode4(oid);
// the user-position x/y shorts were added after v79. Legacy GMS (<79) is oid-only.
func TestStartConversationByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := StartConversation{oid: 2100, x: -5, y: 200}.Encode(l, ctx)(nil)
	// oid only, no x/y.
	want := []byte{0x34, 0x08, 0x00, 0x00} // 2100 uint32-LE
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 StartConversation: got % x, want % x", got, want)
	}
}

func v72le16(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }

func v72ascii(s string) []byte {
	out := make([]byte, 2+len(s))
	binary.LittleEndian.PutUint16(out[:2], uint16(len(s)))
	copy(out[2:], s)
	return out
}

// ContinueConversation reply frame: Encode1 lastMessageType, Encode1 action
// (COutPacket(59) built by OnSay@0x6a0d23: Encode1(0)+Encode1(action)).
func TestContinueConversationByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ContinueConversation{lastMessageType: 0, action: 1}.Encode(l, ctx)(nil)
	want := []byte{0x00, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 ContinueConversation: got % x, want % x", got, want)
	}
}

// ContinueConversationText: EncodeStr text (OnAskText@0x6a1161 reply appends
// EncodeStr(text) on result==1).
func TestContinueConversationTextByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ContinueConversationText{text: "hi there"}.Encode(l, ctx)(nil)
	want := v72ascii("hi there")
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 ContinueConversationText: got % x, want % x", got, want)
	}
}

// ContinueConversationSelection (narrow): Encode1 selection index
// (OnAskMenu@0x6a15a6 reply Encode1(v18) — single byte in the legacy range).
func TestContinueConversationSelectionByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ContinueConversationSelection{selection: 3, wide: false}.Encode(l, ctx)(nil)
	want := []byte{0x03}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 ContinueConversationSelection: got % x, want % x", got, want)
	}
}

// ActionRequest (no movement): Encode4 objectId, Encode1 unk, Encode1 unk2
// (CNpc::GenerateMovePath@0x63fc49; NPC_ACTION op 187). Version-invariant wire.
func TestNPCActionRequestByteV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ActionRequest{objectId: 0x01020304, unk: 1, unk2: 2}.Encode(l, ctx)(nil)
	want := append(append([]byte{0x04, 0x03, 0x02, 0x01}, 0x01), 0x02)
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 ActionRequest: got % x, want % x", got, want)
	}
	_ = v72le16
}
