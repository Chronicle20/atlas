package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 cash serverbound fixtures. Each send-site was body-verified from its
// COutPacket(0x4E) call site in GMS_v72.1_U_DEVM.exe @port 13339 (task-123
// legacy phase 1; item/triple/TV + Megaphone update_time correction added by
// the legacy TV/item/triple gap-fill pass).

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
//
// CORRECTION (legacy TV/item/triple gap-fill pass): after the message tail,
// `mov eax,[arg_4]; cmp eax,ebx; jz loc_9055EC` then `cmp eax,[arg_8]; jz
// loc_905294` (arg_8 is the "attached commodity" pointer, nil in the normal
// no-attachment case) falls straight into jumptable case-33's shared tail
// @loc_905294: rate-check `sub_4DBE16(ctx,500)`, and on success
// `jnz loc_90911A` -> `call SetExclRequestSent (GetTickCount-style read of
// g_CWvsApp+0x18); push eax; call Encode4 @0x909123; call sub_513573
// (SendPacket)`. update_time IS present (trailing uint32) — the earlier
// "shared cleanup, no Encode4" reading stopped short of this shared tail.
// Wire (v72): message(str) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v72 ida=0x904fe2
func TestItemUseMegaphoneBytesV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Hello world!")) + "39300000"
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
//
// CORRECTION (legacy TV/item/triple gap-fill pass): same shared jumptable
// case-33 rate-check-and-send tail @loc_905294 -> loc_90911A as basic
// Megaphone — see that test's comment. update_time IS present (trailing
// uint32).
// Wire (v72): message(str) + whisper(bool) + updateTime(uint32 trailing).
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v72 ida=0x904fe2
func TestItemUseSuperMegaphoneBytesV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 54321
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "0c00" + hex.EncodeToString([]byte("Super hello!")) + "01" + "31d40000"
	if got != want {
		t.Errorf("v72 item use super megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseItemMegaphoneBytesV72 pins the v72 Item Megaphone (5076xxx)
// serverbound wire. IDA v72: SendConsumeCashItemUseRequest's jumptable
// case 14 @0x905609 only constructs/shows a dedicated dialog (0x5D0-byte
// ZAllocEx alloc + sub_5A6B00 ctor, itself CWnd-derived) — it does NOT
// encode any bytes inline. The real send lives in that dialog class's own
// OK-button path: OnCommand (vtable slot 8, sub_5A746D, size 0x2d — decoded
// `if (a2==1) { if (sub_5A7C42(this)) vtable[13](this,1); }`) calls the
// validate+send method sub_5A7C42 (size 0x1ea). Full decompile of
// sub_5A7C42 (0x5a7d1f-0x5a7dc5):
//
//	COutPacket ctor(78=0x4E)                              @0x5a7d1f
//	Encode2(*(WORD*)(this+120))         = slot             @0x5a7d35
//	Encode4(*(DWORD*)(this+124))        = itemId           @0x5a7d40
//	EncodeStr(CCtrlEdit::GetText())     = message          @0x5a7d61
//	Encode1(*(DWORD*)(*(DWORD*)(this+1444)+72)) = whisper  @0x5a7d72
//	Encode1(*(DWORD*)(this+140)!=0)     = hasItem          @0x5a7d84
//	  if hasItem: Encode4(this+128)=invType, Encode4(this+132)=slotPos
//	                                                         @0x5a7d9a/0x5a7da8
//	call SetExclRequestSent(); push eax; Encode4(eax) = updateTime @0x5a7db6
//	SendPacket()                                            @0x5a7dc5
//
// Wire (v72): message(str) + whisper(bool) + hasItem(bool) +
// [invType(int32)+slot(int32)] + updateTime(uint32 trailing) — matches
// ItemUseItemMegaphone.Encode(updateTimeFirst=false) exactly (this codec's
// unconditional `if !updateTimeFirst { WriteInt(updateTime) }` was already
// correct — only Megaphone/SuperMegaphone had the wrong hasUpdateTime gate).
// packet-audit:verify packet=cash/serverbound/CashItemUseItemMegaphone version=gms_v72 ida=0x5a7c42
func TestItemUseItemMegaphoneBytesV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Errorf("v72 item use item megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseTripleMegaphoneBytesV72 pins the v72 Triple Megaphone (5077xxx)
// serverbound wire. IDA v72: jumptable case 60 @0x905090 is SELF-CONTAINED
// (no separate dialog class, unlike item megaphone): it reads up to 3
// trimmed lines out of a VARIANTARG array (sub_71EA03 @0x90511c), computes a
// line count 0-3 by non-empty-string checks (@0x90520a-0x90522e), then:
//
//	Encode1(count)                       @0x905234 (arg_0 = 0..3)
//	loop count times: EncodeStr(line[i]) @0x90525a
//	Encode1(whisper)                     @0x905271 (arg_4)
//
// then falls through to the SAME shared jumptable case-33 rate-check-and-send
// tail as basic Megaphone (loc_905294 @0x905294 -> loc_90911A: `call
// SetExclRequestSent; Encode4(eax); SendPacket`) — confirmed by direct trace:
// case 60's body falls straight into 0x905294 at 0x905288.
// Wire (v72): count(byte) + count×line(str) + whisper(bool) +
// updateTime(uint32 trailing) — matches ItemUseTripleMegaphone.Encode
// (updateTimeFirst=false) exactly. The marker's ida= cites the shared
// dispatcher entry (0x904fe2), matching the evidence/report convention
// already used for the Megaphone/SuperMegaphone cells above (case 60 is
// inline in this same function, not a separate sender).
// packet-audit:verify packet=cash/serverbound/CashItemUseTripleMegaphone version=gms_v72 ida=0x904fe2
func TestItemUseTripleMegaphoneBytesV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewItemUseTripleMegaphone(false)
	input.lines = []string{"line one", "line two"}
	input.whisper = true
	input.updateTime = 12345
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	want := "02" + "0800" + hex.EncodeToString([]byte("line one")) + "0800" + hex.EncodeToString([]byte("line two")) + "01" + "39300000"
	if got != want {
		t.Errorf("v72 item use triple megaphone bytes: got %s, want %s", got, want)
	}
}

// TestItemUseMapleTVBytesV72 pins the v72 Maple TV (5075xxx) serverbound
// wire for tvType 0. IDA v72: jumptable case 46 @0x907702 (first of the
// SIX consecutive TV cases 46-51, same numbering as v87/v95) is
// SELF-CONTAINED (no separate dialog class, unlike item megaphone): it
// constructs 6 local ZXString slots (receiverName + 5 lines), then at the
// tail (@0x907a3e-0x907ade):
//
//	call sub_52319B (bool check) -> neg/sbb/and/add idiom producing a
//	  byte of 1 or 3                                          @0x907a3e-0x907a4b
//	Encode1(that byte)                    = pad               @0x907a54
//	EncodeStr(receiverName)                                    @0x907a6b
//	EncodeStr(line[0..4]) x5                                   @0x907a82-0x907ade
//
// then falls through to the shared jumptable case-33 rate-check-and-send
// tail (loc_905294 -> loc_90911A, same architecture as Triple Megaphone
// above): `call SetExclRequestSent; Encode4(eax); SendPacket`.
// Wire (v72, tvType 0): pad(byte) + receiverName(str) + 5×line(str) +
// updateTime(uint32 trailing) — matches ItemUseMapleTV.Encode(tvType=0,
// updateTimeFirst=false) exactly: tvType!=1 (true) -> tvType>=3 false ->
// tvType!=2 true -> WriteByte(pad); tvType!=4 true -> WriteAsciiString
// (receiverName); then 5 lines.
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleTV version=gms_v72 ida=0x904fe2
func TestItemUseMapleTVBytesV72(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Errorf("v72 item use maple tv bytes: got %s, want %s", got, want)
	}
}
