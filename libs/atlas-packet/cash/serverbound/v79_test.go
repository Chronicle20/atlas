package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 cash serverbound fixtures. Each send-site was body-verified from its
// COutPacket(0x4D) call site in GMS_v79_1_DEVM.exe @port 13340 (task-123
// legacy phase 1; item/triple/TV + Megaphone update_time correction added by
// the legacy TV/item/triple gap-fill pass). Note: the client's local
// `get_update_time()` call at the top of SendConsumeCashItemUseRequest
// @0x956381 is a client-side send-rate throttle check (result gates a
// "please wait" Notice), NOT a wire field — it is unrelated to the
// packet-level update_time discussed below; the outer header still carries
// no update_time (confirmed below).

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
//
// CORRECTION (legacy TV/item/triple gap-fill pass): v79's jumptable is
// byte-identical in structure to v72's (case 60 @0x9563f8 opens with the
// SAME `push 0C0h; Alloc; ctor` sequence as v72's case 60; case 14 @0x956975
// opens with the SAME dialog-alloc pattern as v72's case 14). The Megaphone/
// SuperMegaphone case body falls through into the shared jumptable
// case-33-equivalent rate-check-and-send tail (same architecture as v72's
// loc_905294 -> loc_90911A: `call SetExclRequestSent; Encode4(eax);
// SendPacket`) — update_time IS present (trailing uint32).
// Wire (v79): message(str) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v79 ida=0x95634a
func TestItemUseMegaphoneBytesV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Hello world!")) + "39300000"
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
//
// CORRECTION (legacy TV/item/triple gap-fill pass): same shared
// rate-check-and-send tail as basic Megaphone — see that test's comment.
// update_time IS present (trailing uint32).
// Wire (v79): message(str) + whisper(bool) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v79 ida=0x95634a
func TestItemUseSuperMegaphoneBytesV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 54321
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Super hello!")) + "01" + "31d40000"
	if got != want {
		t.Errorf("v79 item use super megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseItemMegaphoneBytesV79 pins the v79 Item Megaphone (5076xxx)
// serverbound wire. IDA v79: SendConsumeCashItemUseRequest's jumptable
// case 14 @0x956975 only constructs/shows a dedicated dialog (0x5D4-byte
// ZAllocEx alloc + sub_5C11F3 ctor — byte-identical layout to v72's
// sub_5A6B00, same function sizes: ctor 0x116, OnCreate 0x62f, OnCommand
// sub_5C1B61 0x2d, validate+send sub_5C2336 0x1ea). Full decompile of
// sub_5C2336 (0x5c2413-0x5c24b9):
//
//	COutPacket ctor(77=0x4D)                              @0x5c2413
//	Encode2(*(WORD*)(this+120))         = slot             @0x5c2429
//	Encode4(*(DWORD*)(this+124))        = itemId           @0x5c2434
//	EncodeStr(CCtrlEdit::GetText())     = message          @0x5c2455
//	Encode1(*(DWORD*)(*(DWORD*)(this+1448)+72)) = whisper  @0x5c2466
//	Encode1(*(DWORD*)(this+140)!=0)     = hasItem          @0x5c2478
//	  if hasItem: Encode4(this+128)=invType, Encode4(this+132)=slotPos
//	                                                         @0x5c248e/0x5c249c
//	call SetExclRequestSent(); push eax; Encode4(eax) = updateTime @0x5c24aa
//	SendPacket()                                            @0x5c24b9
//
// Wire (v79): message(str) + whisper(bool) + hasItem(bool) +
// [invType(int32)+slot(int32)] + updateTime(uint32 trailing) — matches
// ItemUseItemMegaphone.Encode(updateTimeFirst=false) exactly.
// packet-audit:verify packet=cash/serverbound/CashItemUseItemMegaphone version=gms_v79 ida=0x5c2336
func TestItemUseItemMegaphoneBytesV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewItemUseItemMegaphone(false)
	input.message = "Item hello!"
	input.whisper = true
	input.hasItem = true
	input.invType = 2
	input.slot = 5
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0b00" + hex.EncodeToString([]byte("Item hello!")) + "01" + "01" + "02000000" + "05000000" + "39300000"
	if got != want {
		t.Errorf("v79 item use item megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseTripleMegaphoneBytesV79 pins the v79 Triple Megaphone (5077xxx)
// serverbound wire. IDA v79: jumptable case 60 @0x9563f8 opens with the SAME
// `push 0C0h; Alloc; ctor(sub_95A904)` sequence as v72's case 60 (v72's
// analogous ctor was sub_9094CD) — byte-identical VARIANTARG-array/count/
// trim architecture. Same shape as v72's Triple Megaphone: Encode1(count 0-3)
// + count×EncodeStr(line) + Encode1(whisper), then falls into the shared
// rate-check-and-send tail (`call SetExclRequestSent; Encode4(eax);
// SendPacket`).
// Wire (v79): count(byte) + count×line(str) + whisper(bool) +
// updateTime(uint32 trailing) — matches ItemUseTripleMegaphone.Encode
// (updateTimeFirst=false) exactly.
// packet-audit:verify packet=cash/serverbound/CashItemUseTripleMegaphone version=gms_v79 ida=0x95634a
func TestItemUseTripleMegaphoneBytesV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewItemUseTripleMegaphone(false)
	input.lines = []string{"line one", "line two"}
	input.whisper = true
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "02" + "0800" + hex.EncodeToString([]byte("line one")) + "0800" + hex.EncodeToString([]byte("line two")) + "01" + "39300000"
	if got != want {
		t.Errorf("v79 item use triple megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseMapleTVBytesV79 pins the v79 Maple TV (5075xxx) serverbound
// wire for tvType 0. IDA v79: jumptable case 46 @0x958b2a (first of the SIX
// consecutive TV cases 46-51, same numbering as v72/v87/v95). Directly
// traced the encode tail @0x958e6d-0x958f00 — BYTE-IDENTICAL structure to
// v72's case 46 tail:
//
//	call sub_52319B-equivalent -> neg/sbb/and/add idiom -> byte of 1 or 3
//	                                                          @0x958e5d-0x958e6d
//	Encode1(that byte)                    = pad               @0x958e76
//	EncodeStr(receiverName)                                    @0x958e8d
//	EncodeStr(line[0..4]) x5                                   @0x958ea4-0x958f00
//
// then falls through to the shared rate-check-and-send tail.
// Wire (v79, tvType 0): pad(byte) + receiverName(str) + 5×line(str) +
// updateTime(uint32 trailing) — matches ItemUseMapleTV.Encode(tvType=0,
// updateTimeFirst=false) exactly.
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleTV version=gms_v79 ida=0x95634a
func TestItemUseMapleTVBytesV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewItemUseMapleTV(false, 0)
	input.pad = 3
	input.receiverName = "Receiver"
	input.lines = [5]string{"line0", "line1", "line2", "line3", "line4"}
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "03" +
		"0800" + hex.EncodeToString([]byte("Receiver")) +
		"0500" + hex.EncodeToString([]byte("line0")) +
		"0500" + hex.EncodeToString([]byte("line1")) +
		"0500" + hex.EncodeToString([]byte("line2")) +
		"0500" + hex.EncodeToString([]byte("line3")) +
		"0500" + hex.EncodeToString([]byte("line4")) +
		"39300000"
	if got != want {
		t.Errorf("v79 item use maple tv bytes: got %s, want %s", got, want)
	}
}
