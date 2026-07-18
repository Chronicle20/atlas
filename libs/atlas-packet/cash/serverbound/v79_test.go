package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// v79 cash serverbound fixtures. Each send-site was body-verified from its
// COutPacket(0x4D) call site in GMS_v79_1_DEVM.exe @port 13340 (task-123
// legacy phase 1). Note: the client's local `get_update_time()` call at the
// top of SendConsumeCashItemUseRequest @0x956381 is a client-side send-rate
// throttle check (result gates a "please wait" Notice), NOT a wire field —
// it is unrelated to the packet-level update_time discussed for GMS>=83; the
// outer header still carries no update_time (confirmed below).

// TestItemUseMegaphoneBytesV79 pins the v79 basic-Megaphone USE_CASH_ITEM
// sub-body. IDA v79 CWvsContext::SendConsumeCashItemUseRequest @0x95634a:
// outer header (COutPacket opcode 0x4D, Encode2(slot), Encode4(itemId)
// @0x9563b3-0x9563d6) carries NO update_time. Type-dispatch via
// is_select_npc_item(itemId) (same helper as v72); jumptable label "cases
// 12,13,15" resolves to loc_95663B. `cmp Str2,0Dh; jnz loc_9566DD` — type 12
// (this test) takes the jnz path. Message tail @0x956919-0x956943:
//
//	EncodeStr(message)          @0x95692d
//	cmp Str2,0Dh; jnz skip      @0x956932 (type 12 SKIPS whisper)
//	[shared cleanup — NO Encode4]
//
// Wire (v79): message(str) ONLY. NO update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v79 ida=0x95634a
func TestItemUseMegaphoneBytesV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345 // must NOT appear on the wire for v79
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Hello world!"))
	if got != want {
		t.Errorf("v79 item use megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseSuperMegaphoneBytesV79 pins the v79 SuperMegaphone USE_CASH_ITEM
// sub-body. Same case label loc_95663B (types 12,13,15); type 13 (this test)
// falls through the `cmp Str2,0Dh; jnz` at @0x95663e into the larger dialog
// path, then the SAME message tail:
//
//	EncodeStr(message)          @0x95692d
//	cmp Str2,0Dh; jnz skip      @0x956932 (type 13 MATCHES -> whisper emitted)
//	Encode1(whisper)            @0x95693e
//	[shared cleanup — NO Encode4]
//
// Wire (v79): message(str) + whisper(bool). NO update_time.
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v79 ida=0x95634a
func TestItemUseSuperMegaphoneBytesV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 54321 // must NOT appear on the wire for v79
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Super hello!")) + "01"
	if got != want {
		t.Errorf("v79 item use super megaphone bytes: got %s, want %s", got, want)
	}
}
