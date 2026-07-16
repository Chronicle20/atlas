package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v48 serverbound NPC conversation-reply + NPC_TALK byte fixtures
// (GMS_v48_1_DEVM.exe, port 13337). The reply builders are the CScriptMan
// dialog handlers, all sending COutPacket(47) + Encode1(msgType) +
// Encode1(action) + [type payload]:
//   - Say reply       sub_5B0C11@0x5b0c11: Encode1(0=msgType) + Encode1(action)
//   - AskText reply    sub_5B0E90@0x5b0e90: on accept, EncodeStr(text)
//   - AskMenu reply    sub_5B1195@0x5b1195: on accept, Encode4(selection) — WIDE
//     int32 selection in v48 (v61 OnAskMenu@0x6403bc replies with a narrow
//     Encode1 selection; the menu selection widened to int32 in the v48 layout).
// NPC_TALK (op 46) send site sub_568A2A@0x568a2a: Encode4(npcOid) +
// Encode2(userX) + Encode2(userY) — v48 carries the user position (oid+x+y),
// NOT oid-only.
//
// packet-audit:verify packet=npc/serverbound/NpcContinueConversation version=gms_v48 ida=0x5b0c11
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationText version=gms_v48 ida=0x5b0e90
// packet-audit:verify packet=npc/serverbound/NpcContinueConversationSelection version=gms_v48 ida=0x5b1195
// packet-audit:verify packet=npc/serverbound/NpcStartConversation version=gms_v48 ida=0x568a2a

// ContinueConversation reply frame: Encode1 lastMessageType, Encode1 action
// (sub_5B0C11: Encode1(0)+Encode1(action)).
func TestContinueConversationByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ContinueConversation{lastMessageType: 0, action: 1}.Encode(l, ctx)(nil)
	want := []byte{0x00, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ContinueConversation: got % x, want % x", got, want)
	}
}

// ContinueConversationText: EncodeStr text (sub_5B0E90 reply appends
// EncodeStr(text) on accept).
func TestContinueConversationTextByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ContinueConversationText{text: "hi there"}.Encode(l, ctx)(nil)
	want := v72ascii("hi there")
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ContinueConversationText: got % x, want % x", got, want)
	}
}

// ContinueConversationSelection (wide): Encode4 selection index (sub_5B1195
// menu reply Encode4(v9) — 4-byte selection in v48).
func TestContinueConversationSelectionByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ContinueConversationSelection{selection: 3, wide: true}.Encode(l, ctx)(nil)
	want := []byte{0x03, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ContinueConversationSelection: got % x, want % x", got, want)
	}
}

// NPC_TALK (v48 send op 46) — sub_568A2A builds COutPacket(46) then
// Encode4(npcOid) + Encode2(userX) + Encode2(userY). v48 includes the user
// position x/y. oid=42, x=100, y=-50 (0xFFCE LE).
func TestStartConversationByteV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := StartConversation{oid: 42, x: 100, y: -50}.Encode(l, ctx)(nil)
	want := []byte{
		0x2A, 0x00, 0x00, 0x00, // npcOid = 42 (Encode4 @0x569297/0x569380)
		0x64, 0x00, // userX = 100 (Encode2 @0x5692b0/0x569399)
		0xCE, 0xFF, // userY = -50 int16 LE (Encode2 @0x5692ca/0x5693b3)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 StartConversation: got % x, want % x", got, want)
	}
}
