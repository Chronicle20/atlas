package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v79 ASK_MENU body (CScriptMan::OnAskMenu @0x6c8863, GMS_v79_1_DEVM.exe
// port 13340). The v79 client merged the avatar-style menu into ASK_MENU and
// reads:
//
//	CInPacket::DecodeStr(&v25)        -> message
//	v6 = CInPacket::Decode1(a4)       -> count
//	for count: CInPacket::Decode4(a4) -> avatar look id (SetUtilDlgEx_AVATAR)
//
// v83 (@0x746fad) and v95 (@0x6dce00) OnAskMenu read a plain single string with
// NO count. Atlas uses ASK_MENU only for plain #L#-token text menus, so the v79
// wire is DecodeStr(message) + Decode1(count=0) and carries no avatar styles.
// The msgType byte that routes to this handler is 7 in v79 (vs 4 in v83, 5 in
// v95) — see the corrected gms_v79 messageType table.
//
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=gms_v79 ida=0x6c8863
func TestNpcConversationAskMenuByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)

	detail := &AskMenuConversationDetail{Message: "#L0#Option 1#l\r\n#L1#Option 2#l"}
	got := detail.Encode(l, ctx)(nil)

	// DecodeStr(message) then Decode1(count=0): asciiBytes(message) + 0x00.
	want := append(asciiBytes("#L0#Option 1#l\r\n#L1#Option 2#l"), 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskMenu detail: got % x, want % x", got, want)
	}

	// Sanity: v83 (plain single string) carries NO trailing count byte.
	v83 := detail.Encode(l, test.CreateContext("GMS", 83, 1))(nil)
	if !bytes.Equal(v83, asciiBytes("#L0#Option 1#l\r\n#L1#Option 2#l")) {
		t.Fatalf("v83 AskMenu detail should be a plain string with no count: got % x", v83)
	}
}

// gms_v79 ASK_MEMBER_SHOP_AVATAR body (CScriptMan::OnAskMembershopAvatar
// @0x6c8bc8, GMS_v79_1_DEVM.exe port 13340, msgType 9). The v79 client reads:
//
//	CInPacket::DecodeStr(&v22)              -> message              (@0x6c8be2)
//	v4 = CInPacket::Decode1(a4)             -> candidate count      (@0x6c8bfb)
//	for count: DecodeBuffer(&v15, 8)        -> cash item SN (int64) (@0x6c8c45)
//	           CInPacket::Decode1(a4)       -> byte                 (@0x6c8c4d)
//
// The per-entry (int64 SN + byte) format is incompatible with the v83+ int32
// style-id list (v83 @0x74730b reads Decode4 style ids), and Atlas has no
// server-side SN data to drive it, so AskMemberShopAvatarConversationDetail
// gates count=0 for the legacy range: AsciiString(Message) + WriteByte(0),
// which the v79 client decodes as message + count=0 (no loop). Mirrors the
// ASK_MENU v79 handling.
//
// packet-audit:verify packet=npc/clientbound/NpcAskMemberShopAvatarConversationDetail version=gms_v79 ida=0x6c8bc8
func TestNpcConversationAskMemberShopAvatarByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)

	detail := &AskMemberShopAvatarConversationDetail{Message: "Select an avatar to preview."}
	got := detail.Encode(l, ctx)(nil)

	// DecodeStr(message) then Decode1(count=0): asciiBytes(message) + 0x00.
	want := append(asciiBytes("Select an avatar to preview."), 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 AskMemberShopAvatar detail: got % x, want % x", got, want)
	}

	// v83+ still emits the int32 style-id list: with one candidate the wire is
	// asciiBytes(message) + 0x01 + 4-byte style id (unchanged by the v79 gate).
	v83detail := &AskMemberShopAvatarConversationDetail{Message: "m", Candidates: []uint32{0x11223344}}
	v83 := v83detail.Encode(l, test.CreateContext("GMS", 83, 1))(nil)
	wantV83 := append(asciiBytes("m"), 0x01, 0x44, 0x33, 0x22, 0x11)
	if !bytes.Equal(v83, wantV83) {
		t.Fatalf("v83 AskMemberShopAvatar detail should keep int32 candidates: got % x, want % x", v83, wantV83)
	}
}
