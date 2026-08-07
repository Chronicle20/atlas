package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa0a63f:
//
// The function opens with a shared header (opcode 0x4F, Encode2(txn),
// Encode4(itemId)) then classifies the item via get_consume_cash_item_type
// (cash-slot type 12 = Megaphone). Cash-slot types 12/13 share a body
// (jumptable case label "cases 12,13,15" @0xa0a930): when type != 13
// (Megaphone is type 12) the client shows a CUtilDlgEx message-input dialog,
// trims the result, and at @0xa0ac22 calls
//
//	EncodeStr(message)
//
// with NO Encode1(whisper) following (that Encode1 @0xa0ac33 is gated
// `cmp type,0xD(13); jnz skip` — dead for type 12). Both the type-12 and
// type-13 exit paths jump into the SHARED final-send tail: `cmp
// [Unknown],0; jz loc_A0A8F1` -> CanSendExclRequest -> on success,
// loc_A0EA53:
//
//	call get_update_time(); Encode4(result); call SendPacket.
//
// This is the DEFINITIVE resolution of the trailing-updateTime question for
// v83: updateTime is a TRAILING uint32 (Encode4) written immediately before
// SendPacket, appended by a body-shared tail used by every USE_CASH_ITEM
// megaphone sub-case traced (12, 13, 14/CItemSpeakerDlg, 60) — confirming
// updateTimeFirst=false for gms_v83 exactly as the existing version gate
// (MajorVersion < 95) already assumes.
//
// Wire (v83): message(str) + updateTime(uint32 trailing). No whisper byte
// for the plain Megaphone case.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v83 ida=0xa0a63f
func TestItemUseMegaphoneByteOutputV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345
	expected := []byte{
		0x0C, 0x00, 'H', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', '!', // message
		0x39, 0x30, 0x00, 0x00, // updateTime=12345 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 item use megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::SendConsumeCashItemUseRequest@0x9eb3e0:
//
// The function header (0x9eb4a4-0x9eb4e8) is, in order: COutPacket ctor
// (opcode 0x55) -> get_update_time() -> Encode4(update_time) -> Encode2(nPOS)
// -> Encode4(itemId) -> get_consume_cash_item_type(itemId) -> switch(type-12).
// This DEFINITIVELY resolves the trailing-updateTime question for v95:
// update_time is encoded IMMEDIATELY after the opcode, BEFORE nPOS/itemId —
// i.e. updateTimeFirst=TRUE for gms_v95, confirming the existing MajorVersion
// >=95 gate. The shared jumptable block for types 12/13/15/45 (@0x9eb811,
// entered via loc_9EBC44 after the CSpeakerWorldDlg/CUtilDlgEx message-input
// dialog returns) does:
//
//	EncodeStr(message) @0x9ebc59
//	cmp type,13; jz whisper; cmp type,45; jnz skip   @0x9ebc62-0x9ebc6a
//	  (type 12 is NEITHER 13 NOR 45 -> whisper Encode1 SKIPPED)
//	[shared cleanup, NO trailing update_time write]
//
// Wire (v95): message(str) ONLY — no whisper, no trailing updateTime (already
// written in the shared header). Matches ItemUseMegaphone.Encode(updateTimeFirst=true)
// exactly.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v95 ida=0x9eb3e0
func TestItemUseMegaphoneByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItemUseMegaphone(true)
	input.message = "Hello world!"
	expected := []byte{
		0x0C, 0x00, 'H', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', '!', // message
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 item use megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa9fef9:
//
// Function header (0xa9ff62-0xa9ff98): COutPacket ctor (opcode 0x52) ->
// get_update_time() -> Encode4(update_time) -> Encode2(slot) ->
// Encode4(itemId) -> get_consume_cash_item_type -> switch(type-12). update_time
// is LEADING, confirming updateTimeFirst=TRUE for gms_v87 (matches the
// production gate in character_cash_item_use.go: t.MajorVersion()>=87).
// Jumptable label "cases 12,13,15" @0xaa01ff: for type 12 (Megaphone), a
// small confirm dialog (CUtilDlgEx, no whisper checkbox) is shown; on OK the
// shared tail @0xaa0390-0xaa04f1 does TrimRight/TrimLeft + curse-filter, then
//
//	EncodeStr(message) @0xaa04f1
//	cmp type,0xD(13); jnz skip_whisper   @0xaa04f6-0xaa04fa (type 12 != 13, SKIPPED)
//	[shared cleanup, falls to loc_AA01C0 -> CanSendExclRequest -> 0xaa43a8 SendPacketThunk, NO trailing update_time]
//
// Wire (v87): message(str) ONLY — identical shape to gms_v95.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v87 ida=0xa9fef9
func TestItemUseMegaphoneByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItemUseMegaphone(true)
	input.message = "Hello world!"
	expected := []byte{
		0x0C, 0x00, 'H', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', '!', // message
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 item use megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345, partially-symbol-named
// — COutPacket helpers named, dialog/cleanup helpers are sub_XXXXXX) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa54a2f:
//
// Function header (0xa54a98-0xa54ac1): COutPacket ctor (opcode 0x4F) ->
// Encode2(slot) -> Encode4(itemId) -> get_consume_cash_item_type -> switch
// (type-12). NO Encode4(update_time) call anywhere in the header — matches
// megaphoneHasUpdateTime's v83 finding exactly (v84 is the OTHER
// updateTimeFirst=false GMS build). Jumptable label "cases 12,13,15"
// @0xa54d27 (byte-identical case-numbering to gms_v87/v95): for type 12, a
// small confirm dialog is shown; on OK, the shared tail does
// TrimRight/TrimLeft, then:
//
//	EncodeStr(message) @0xa55019
//	cmp type,0xD(13); jnz skip_whisper   @0xa5501e-0xa55022 (type 12 != 13, SKIPPED)
//	[falls to loc_A54CE8 "cases 33,71,72" -> CanSendExclRequest -> loc_A58E47:
//	 get_update_time() -> Encode4(result) -> SendPacket]  (TRAILING update_time)
//
// Wire (v84): message(str) + updateTime(uint32 trailing) — same shape as gms_v83.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=gms_v84 ida=0xa54a2f
func TestItemUseMegaphoneByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItemUseMegaphone(false)
	input.message = "Hello world!"
	input.updateTime = 12345
	expected := []byte{
		0x0C, 0x00, 'H', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', '!', // message
		0x39, 0x30, 0x00, 0x00, // updateTime=12345 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 item use megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms185 MapleStory_dump_SCY.exe, port 13344) —
// CWvsContext::SendConsumeCashItemUseRequest@0xaef2f5:
//
// Function header (0xaef35e-0xaef393): COutPacket ctor (opcode 0x47) ->
// get_update_time() -> Encode4(update_time) -> Encode2(nType/slot) ->
// Encode4(String2/itemId) -> get_consume_cash_item_type -> switch(type-12).
// update_time is LEADING, confirming updateTimeFirst=TRUE for jms185
// (matches the production gate: t.MajorVersion()>=87).
//
// get_cashslot_item_type@0x49a1ee (507-family switch on tier =
// nItemID%10000/1000): tier 0 (Cheap, 5070000) -> type 12 (case0@0x49a274,
// push 0Ch). The dispatch jumptable (jpt_AEF3A8, base 0xaf2b6a) routes type
// 12 to the shared arm @0xaef5b9 (comment "jumptable 00AEF3A8 cases
// 12,13,15,47,48") along with the basic-Megaphone(tier1/type13),
// Heart(tier3/type47), and Skull(tier4/type48) items. Inside that arm the
// tail @0xaef987-0xaef9c8 does:
//
//	EncodeStr(message) @0xaef98a
//	cmp type,0x0D(13); jz whisper    @0xaef98f-0xaef993
//	cmp type,0x2F(47); jz whisper    @0xaef995-0xaef999
//	cmp type,0x30(48); jnz skip_whisper (type 12 is none of 13/47/48, SKIPPED)
//	[shared cleanup, NO trailing update_time — already written in the leading
//	 header]
//
// Wire (jms185, type 12/Cheap): message(str) ONLY — matches
// ItemUseMegaphone.Encode(updateTimeFirst=true) exactly, confirming the
// existing case-0 (Cheap) routing in character_cash_item_use_megaphone.go
// reuses the correct shape on JMS.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMegaphone version=jms_v185 ida=0xaef2f5
func TestItemUseMegaphoneByteOutputJMS(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := NewItemUseMegaphone(true)
	input.message = "Hello world!"
	expected := []byte{
		0x0C, 0x00, 'H', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', '!', // message
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("jms185 item use megaphone (type 12/Cheap) golden mismatch: got %v want %v", actual, expected)
	}
}

func TestItemUseMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// task-123 phase 3: matches the production gate exactly
			// (character_cash_item_use.go: t.MajorVersion() >= 87, no region
			// check) — GMS v87 IDA-confirmed leading update_time (CItemSpeakerDlg
			// @0x623728 gms_v87, CWvsContext::SendConsumeCashItemUseRequest
			// @0xa9fef9 gms_v87), so the old >=95 threshold under-tested v87.
			updateTimeFirst := v.MajorVersion >= 87
			// legacy TV/item/triple gap-fill pass: update_time is a trailing
			// Encode4 whenever updateTimeFirst is false on EVERY GMS build
			// (v48/61/72/79 included) — see this file's type doc comment for
			// the IDA evidence that disproved the earlier
			// GMS<83-omits-update_time gate.
			input := NewItemUseMegaphone(updateTimeFirst)
			input.message = "Hello world!"
			if !updateTimeFirst {
				input.updateTime = 12345
			}
			output := NewItemUseMegaphone(updateTimeFirst)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Message() != input.Message() {
				t.Errorf("message: got %q, want %q", output.Message(), input.Message())
			}
			if !updateTimeFirst && output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
