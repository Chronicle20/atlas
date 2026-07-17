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

func TestItemUseMegaphoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
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
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
