package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa0a63f, shared cases-12/13/15
// body @0xa0a930:
//
//	cmp type,0xD(13); jnz <type-12/15 path>       // type==13 -> SuperMegaphone
//	  (type==13 path collects message+whisper via a distinct dialog, then
//	   jumps to the SAME trim/EncodeStr tail as type 12/15)
//	@0xa0ac22: EncodeStr(message)
//	@0xa0ac27: cmp type,0xD; jnz skip
//	@0xa0ac33: Encode1(var_3C)      // whisper byte — ONLY emitted when type==13
//	then falls into the shared CanSendExclRequest -> Encode4(updateTime) ->
//	SendPacket tail (see ItemUseMegaphone evidence for the trailing-updateTime
//	proof).
//
// Wire (v83): message(str) + whisper(bool) + updateTime(uint32 trailing).
// Matches ItemUseSuperMegaphone.Encode exactly.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v83 ida=0xa0a63f
func TestItemUseSuperMegaphoneByteOutputV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 54321
	expected := []byte{
		0x0C, 0x00, 'S', 'u', 'p', 'e', 'r', ' ', 'h', 'e', 'l', 'l', 'o', '!', // message
		0x01,                   // whisper=true
		0x31, 0xD4, 0x00, 0x00, // updateTime=54321 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 item use super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::SendConsumeCashItemUseRequest@0x9eb3e0, shared cases-12/13/15/45
// body (entry @0x9eb811, message-write tail @loc_9EBC44):
//
//	EncodeStr(message)                                @0x9ebc59
//	cmp type,0Dh(13); jz whisper                       @0x9ebc62-0x9ebc65
//	cmp type,2Dh(45); jnz skip                          @0x9ebc67-0x9ebc6a
//	loc_9EBC6C: Encode1(s4 = bCheckWhisper out-param)   @0x9ebc6c-0x9ebc75
//	  (type==13 -> whisper Encode1 EMITTED)
//	[shared cleanup, NO trailing update_time — already written in the
//	 shared header before the type dispatch, confirming updateTimeFirst=TRUE]
//
// Wire (v95): message(str) + whisper(bool), NO trailing updateTime. Matches
// ItemUseSuperMegaphone.Encode(updateTimeFirst=true) exactly.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v95 ida=0x9eb3e0
func TestItemUseSuperMegaphoneByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItemUseSuperMegaphone(true)
	input.message = "Super hello!"
	input.whisper = true
	expected := []byte{
		0x0C, 0x00, 'S', 'u', 'p', 'e', 'r', ' ', 'h', 'e', 'l', 'l', 'o', '!', // message
		0x01, // whisper=true
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 item use super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa9fef9, jumptable label
// "cases 12,13,15" @0xaa01ff: for type 13 (SuperMegaphone), a bigger
// CUtilDlgEx confirm dialog (StringPool string 0x119, "will be seen by the
// whole world") is shown; on OK the shared tail does TrimRight/TrimLeft +
// curse-filter, then:
//
//	EncodeStr(message) @0xaa04f1
//	cmp type,0xD(13); jnz skip_whisper   @0xaa04f6-0xaa04fa (type==13, TAKEN)
//	Encode1(whisper) @0xaa0502   (pushes [var_3C], the checkbox state)
//	[shared cleanup, falls to loc_AA01C0 -> CanSendExclRequest -> 0xaa43a8 SendPacketThunk, NO trailing update_time]
//
// Wire (v87): message(str) + whisper(bool) — identical shape to gms_v95.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v87 ida=0xa9fef9
func TestItemUseSuperMegaphoneByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItemUseSuperMegaphone(true)
	input.message = "Super hello!"
	input.whisper = true
	expected := []byte{
		0x0C, 0x00, 'S', 'u', 'p', 'e', 'r', ' ', 'h', 'e', 'l', 'l', 'o', '!', // message
		0x01, // whisper=true
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 item use super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa54a2f, jumptable label
// "cases 12,13,15" @0xa54d27: for type 13 (SuperMegaphone), a bigger
// confirm dialog (StringPool 0x112) is shown; on OK the shared tail does
// TrimRight/TrimLeft, then:
//
//	EncodeStr(message) @0xa55019
//	cmp type,0xD(13); jnz skip_whisper   @0xa5501e-0xa55022 (type==13, TAKEN)
//	Encode1(whisper) @0xa5502a
//	[falls to loc_A54CE8 "cases 33,71,72" -> CanSendExclRequest -> loc_A58E47:
//	 get_update_time() -> Encode4(result) -> SendPacket]  (TRAILING update_time)
//
// Wire (v84): message(str) + whisper(bool) + updateTime(uint32 trailing) —
// same shape as gms_v83.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=gms_v84 ida=0xa54a2f
func TestItemUseSuperMegaphoneByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItemUseSuperMegaphone(false)
	input.message = "Super hello!"
	input.whisper = true
	input.updateTime = 12345
	expected := []byte{
		0x0C, 0x00, 'S', 'u', 'p', 'e', 'r', ' ', 'h', 'e', 'l', 'l', 'o', '!', // message
		0x01,                   // whisper=true
		0x39, 0x30, 0x00, 0x00, // updateTime=12345 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 item use super megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms185 MapleStory_dump_SCY.exe, port 13344) —
// CWvsContext::SendConsumeCashItemUseRequest@0xaef2f5, shared arm @0xaef5b9
// (comment "jumptable 00AEF3A8 cases 12,13,15,47,48"):
//
// get_cashslot_item_type@0x49a1ee: tier 3 (Heart, 5073000) -> type 47
// (case3@0x49a27c, push 2Fh); tier 4 (Skull, 5074000) -> type 48
// (case4@0x49a280, push 30h). Both land in the same shared arm as
// Cheap(type12)/basic-Megaphone(type13). Tail @0xaef987-0xaef9c8:
//
//	EncodeStr(message) @0xaef98a
//	cmp type,0x0D(13); jz whisper    @0xaef98f-0xaef993
//	cmp type,0x2F(47); jz whisper    @0xaef995-0xaef999   (Heart, TAKEN)
//	cmp type,0x30(48); jnz skip      @0xaef99b-0xaef99f   (Skull, TAKEN via jz)
//	loc_AEF9A1: Encode1(whisper) @0xaef9a7
//	[shared cleanup, NO trailing update_time — leading header, updateTimeFirst=TRUE]
//
// Wire (jms185, type 47/Heart and type 48/Skull): message(str) +
// whisper(bool) — matches ItemUseSuperMegaphone.Encode(updateTimeFirst=true)
// exactly, confirming the existing case-3 (Heart) and case-4 (Skull) super
// routing in character_cash_item_use_megaphone.go reuses the correct shape
// on JMS. (Note: get_cashslot_item_type's 507-family switch has NO arm for
// tier 2/Super Megaphone itself on jms185 — index 2 of jpt_49A26D resolves
// to def_49A22A/return-0 — so plain 5072000 has no JMS send path at all;
// out of scope for this task's Heart/Skull verification, flagged in the
// task-123 report.)
//
// packet-audit:verify packet=cash/serverbound/CashItemUseSuperMegaphone version=jms_v185 ida=0xaef2f5
func TestItemUseSuperMegaphoneByteOutputJMS(t *testing.T) {
	cases := []struct {
		name string
		typ  string
	}{
		{"heart_type47", "heart"},
		{"skull_type48", "skull"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext("JMS", 185, 1)
			input := NewItemUseSuperMegaphone(true)
			input.message = "Super hello!"
			input.whisper = true
			expected := []byte{
				0x0C, 0x00, 'S', 'u', 'p', 'e', 'r', ' ', 'h', 'e', 'l', 'l', 'o', '!', // message
				0x01, // whisper=true
			}
			actual := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(actual, expected) {
				t.Errorf("jms185 item use super megaphone (%s) golden mismatch: got %v want %v", tc.typ, actual, expected)
			}
		})
	}
}

func TestItemUseSuperMegaphoneRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		whisper bool
	}{
		{"whisper_false", false},
		{"whisper_true", true},
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				// task-123 phase 3: matches the production gate exactly (see
				// item_use_megaphone_test.go for the IDA citation).
				updateTimeFirst := v.MajorVersion >= 87
				// legacy TV/item/triple gap-fill pass: update_time is a
				// trailing Encode4 whenever updateTimeFirst is false on
				// EVERY GMS build (v48/61/72/79 included) — see
				// item_use_megaphone.go's doc comment for the IDA evidence
				// that disproved the earlier GMS<83-omits-update_time gate.
				input := NewItemUseSuperMegaphone(updateTimeFirst)
				input.message = "Super hello!"
				input.whisper = tc.whisper
				if !updateTimeFirst {
					input.updateTime = 54321
				}
				output := NewItemUseSuperMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Message() != input.Message() {
					t.Errorf("message: got %q, want %q", output.Message(), input.Message())
				}
				if output.Whisper() != input.Whisper() {
					t.Errorf("whisper: got %v, want %v", output.Whisper(), input.Whisper())
				}
				if !updateTimeFirst && output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
