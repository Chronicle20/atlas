package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v72 cash serverbound fixtures. Each send-site was body-verified from its
// COutPacket(0x4E) call site in GMS_v72.1_U_DEVM.exe @port 13339 (task-123
// legacy phase 1).

// TestItemUseMegaphoneBytesV72 pins the v72 basic-Megaphone USE_CASH_ITEM
// sub-body. IDA v72 CWvsContext::SendConsumeCashItemUseRequest @0x904fe2:
// outer header (COutPacket opcode 0x4E, Encode2(slot), Encode4(itemId)
// @0x90504b-0x90506e) carries NO update_time. The type-dispatch here is
// keyed by is_select_npc_item(itemId), NOT the raw cash-slot-type byte used
// in v48/v61; jumptable label "cases 12,13,15" resolves to loc_9052D3.
// `cmp Str2,0Dh; jnz loc_90536E` — type 12 (this test) takes the jnz path.
// Message tail @0x9055ad-0x9055d7:
//
//	EncodeStr(message)          @0x9055c1
//	cmp Str2,0Dh; jnz skip      @0x9055c6 (type 12 SKIPS whisper)
//	[shared cleanup — NO Encode4]
//
// Wire (v72): message(str) ONLY. NO update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v72 ida=0x904fe2
func TestItemUseMegaphoneBytesV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345 // must NOT appear on the wire for v72
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Hello world!"))
	if got != want {
		t.Errorf("v72 item use megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseSuperMegaphoneBytesV72 pins the v72 SuperMegaphone USE_CASH_ITEM
// sub-body. Same case label loc_9052D3 (types 12,13,15); type 13 (this test)
// falls through the `cmp Str2,0Dh; jnz` at @0x9052d6 into the larger dialog
// path, then the SAME message tail:
//
//	EncodeStr(message)          @0x9055c1
//	cmp Str2,0Dh; jnz skip      @0x9055c6 (type 13 MATCHES -> whisper emitted)
//	Encode1(whisper)            @0x9055d2
//	[shared cleanup — NO Encode4]
//
// Wire (v72): message(str) + whisper(bool). NO update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v72 ida=0x904fe2
func TestItemUseSuperMegaphoneBytesV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 54321 // must NOT appear on the wire for v72
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Super hello!")) + "01"
	if got != want {
		t.Errorf("v72 item use super megaphone bytes: got %s, want %s", got, want)
	}
}
