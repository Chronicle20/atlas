package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 serverbound NPC conversation-reply + action byte fixtures
// (GMS_v61.1_U_DEVM.exe, port 13338). These structs are version-invariant in
// the legacy range; the v61 client encode sites are the COutPacket(56) reply
// frame built by each per-msgType CScriptMan handler (OnSay@0x63fb44 /
// OnAskText@0x63ff86 / OnAskMenu@0x6403bc, all verified against the v61
// decompile), and the NPC_ACTION send CNpc::GenerateMovePath@0x5ea07a
// (COutPacket(164): Encode4 objectId, Encode1 unk, Encode1 unk2). Byte-identical
// to the verified gms_v72 read orders.
//
// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=gms_v61 ida=0x63fb44
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=gms_v61 ida=0x63ff86
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=gms_v61 ida=0x6403bc
// packet-audit:verify packet=npc/serverbound/NpcActionRequest version=gms_v61 ida=0x5ea07a

// ContinueConversation reply frame: Encode1 lastMessageType, Encode1 action
// (COutPacket(56) built by OnSay@0x63fb44: Encode1(0)+Encode1(action)).
func TestContinueConversationByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ContinueConversation{lastMessageType: 0, action: 1}.Encode(l, ctx)(nil)
	want := []byte{0x00, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 ContinueConversation: got % x, want % x", got, want)
	}
}

// ContinueConversationText: EncodeStr text (OnAskText@0x63ff86 reply appends
// EncodeStr(text) on result==1).
func TestContinueConversationTextByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ContinueConversationText{text: "hi there"}.Encode(l, ctx)(nil)
	want := v72ascii("hi there")
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 ContinueConversationText: got % x, want % x", got, want)
	}
}

// ContinueConversationSelection (narrow): Encode1 selection index
// (OnAskMenu@0x6403bc reply Encode1(v19) — single byte in the legacy range).
func TestContinueConversationSelectionByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ContinueConversationSelection{selection: 3, wide: false}.Encode(l, ctx)(nil)
	want := []byte{0x03}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 ContinueConversationSelection: got % x, want % x", got, want)
	}
}

// ActionRequest (no movement): Encode4 objectId, Encode1 unk, Encode1 unk2
// (CNpc::GenerateMovePath@0x5ea07a; COutPacket(164)). Version-invariant wire.
func TestNPCActionRequestByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ActionRequest{objectId: 0x01020304, unk: 1, unk2: 2}.Encode(l, ctx)(nil)
	want := append(append([]byte{0x04, 0x03, 0x02, 0x01}, 0x01), 0x02)
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 ActionRequest: got % x, want % x", got, want)
	}
}

// NPC_TALK (v61 send op 54) — sub_7B1403@0x7b1403 builds COutPacket(54) then
// Encode4(npcOid) + Encode2(userX) + Encode2(userY) (@0x7b143a / 0x7b1450 /
// 0x7b1461). v61 includes the user-position x/y (verified; v72 twin sub_63FD91
// op57 has the same layout). oid=42, x=100, y=-50 (0xFFCE LE).
//
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v61 ida=0x7b1403
func TestStartConversationByteV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 61, 1)
	got := StartConversation{oid: 42, x: 100, y: -50}.Encode(l, ctx)(nil)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // npcOid = 42 (Encode4 @0x7b143a)
		0x64, 0x00, // userX = 100 (Encode2 @0x7b1450)
		0xCE, 0xFF, // userY = -50 int16 LE (Encode2 @0x7b1461)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 StartConversation: got % x, want % x", got, want)
	}
}
