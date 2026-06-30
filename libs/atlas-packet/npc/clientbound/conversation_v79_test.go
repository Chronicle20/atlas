package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
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
