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
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
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
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
