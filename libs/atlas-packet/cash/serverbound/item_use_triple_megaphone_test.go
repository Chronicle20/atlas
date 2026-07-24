package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::SendConsumeCashItemUseRequest@0x9eb3e0. The Triple Megaphone
// case allocates a CSpeakerWorldDlgEx (ctor @0x786cc0), DoModal, then calls
// CSpeakerWorldDlgEx::GetResult(sResult1, sResult2, sResult3, &bCheckWhisper)
// @0x9eb5c8 — the 4th (by-ref int) GetResult out-param is the whisper
// checkbox value (NOT "count", correcting the controller's initial hint in
// serverbound-ida-findings.md). Validation (TrimRight/TrimLeft/IsEmpty per
// line, @0x9eb5cd-0x9eb62a) determines which of the 3 lines are non-empty;
// a trimmed line >60 chars aborts via Notice (SP idx 0x11E) with NO send.
// The actual encode sequence, past validation:
//
//	edi = 3 if !IsEmpty(sResult3), else 2 if !IsEmpty(sResult2), else 1  @0x9eb779-0x9eb7a2
//	Encode1(edi)                                          // count        @0x9eb7a7
//	loop edi times: EncodeStr(line[i])  (sResult1,2,3 in order)            @0x9eb7c5 (loop body)
//	Encode1(bCheckWhisper)  // read back from the GetResult out-param      @0x9eb7db
//	[falls into the shared cases-33/72/73 cleanup tail @0x9eb80c — NO
//	 trailing update_time write, confirming updateTimeFirst=TRUE for this
//	 arm too, consistent with the shared function-header proof]
//
// Wire (v95): count(byte) + count×line(str) + whisper(bool), no trailing
// updateTime. Matches ItemUseTripleMegaphone.Encode(updateTimeFirst=true)
// EXACTLY — no struct fix needed; this promotes the cell the v95 pass
// (commit fbb66ed75) left BLOCKED.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTripleMegaphone version=gms_v95 ida=0x9eb3e0
func TestItemUseTripleMegaphoneByteOutputV95(t *testing.T) {
	cases := []struct {
		name     string
		lines    []string
		whisper  bool
		expected []byte
	}{
		{
			name:    "lines_1",
			lines:   []string{"A1"},
			whisper: false,
			expected: []byte{
				0x01,                 // count=1
				0x02, 0x00, 'A', '1', // line[0]
				0x00, // whisper=false
			},
		},
		{
			name:    "lines_3",
			lines:   []string{"L1", "L2", "L3"},
			whisper: true,
			expected: []byte{
				0x03,                 // count=3
				0x02, 0x00, 'L', '1', // line[0]
				0x02, 0x00, 'L', '2', // line[1]
				0x02, 0x00, 'L', '3', // line[2]
				0x01, // whisper=true
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 95, 1)
			input := NewItemUseTripleMegaphone(true)
			input.lines = tc.lines
			input.whisper = tc.whisper
			actual := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(actual, tc.expected) {
				t.Errorf("v95 item use triple megaphone (%s) golden mismatch: got %v want %v", tc.name, actual, tc.expected)
			}
		})
	}
}

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa0a63f. The Triple Megaphone
// case sits near the function's own start: it calls the (unnamed, no PDB)
// GetResult-equivalent sub_7E0B5D@0xa0a779 with the SAME 4-arg shape as v95's
// CSpeakerWorldDlgEx::GetResult (3 line out-params + &whisper out-param,
// reusing the nItemID stack slot for the 4th by-ref int — same reuse pattern
// v95 uses for its "pItemInfo" slot). Validation
// (TrimRight/TrimLeft/IsEmpty per line @0xa0a783-0xa0a844, Notice SP idx
// 0x111 on a >60-char line) is structurally identical to v95's. The encode
// sequence past validation:
//
//	a2 = 3 if line3 non-empty, else 2 if line2 non-empty, else 1  @0xa0a867-0xa0a888
//	Encode1(a2)                                            // count   @0xa0a891
//	loop a2 times: EncodeStr(line[i])                              @0xa0a8ba (loop body)
//	Encode1(whisper)  // read back from the reused nItemID slot     @0xa0a8ce
//	[falls into the shared jumptable-cases-33/69 continuation @0xa0a8f1 ->
//	 CanSendExclRequest -> on success, loc_A0EA53: get_update_time();
//	 Encode4(result); SendPacket — the SAME shared trailing-updateTime tail
//	 already DEFINITIVELY established by the ItemUseMegaphone v83 evidence
//	 (item_use_megaphone_test.go) for cash-slot types 12/13/14/60. NO Encode4
//	 appears in the Triple Megaphone case body itself.]
//
// Wire (v83): count(byte) + count×line(str) + whisper(bool) +
// updateTime(uint32 trailing, appended by the shared send tail). Matches
// ItemUseTripleMegaphone.Encode(updateTimeFirst=false) EXACTLY — no struct
// fix needed. Retro-verifies the cell the v83 pass (task-123 phase 19) left
// unresolved.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTripleMegaphone version=gms_v83 ida=0xa0a63f
func TestItemUseTripleMegaphoneByteOutputV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewItemUseTripleMegaphone(false)
	input.lines = []string{"L1", "L2"}
	input.whisper = true
	input.updateTime = 24680
	expected := []byte{
		0x02,                 // count=2
		0x02, 0x00, 'L', '1', // line[0]
		0x02, 0x00, 'L', '2', // line[1]
		0x01,                   // whisper=true
		0x68, 0x60, 0x00, 0x00, // updateTime=24680 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 item use triple megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa54a2f. jumptable case 60
// (@0xa54ae4, alloc 0xC8h dialog, identical case NUMBER to gms_v87/gms_v83
// — the Triple Megaphone case is NOT part of the shared megaphone-case-33/
// 71/72 group). GetResult-equivalent call takes the same 4-out-param shape
// (3 line pointers + whisper) as the already-verified v83/v87/v95 cells;
// TrimRight/TrimLeft validation on all 3 lines matches. The encode sequence
// past validation:
//
//	count = 3 if !IsEmpty(line3), else 2 if !IsEmpty(line2), else 1  @0xa54c5e-0xa54c82
//	Encode1(count)                                          @0xa54c88
//	loop count times: EncodeStr(line[i])                            @0xa54cae (loop body)
//	Encode1(whisper)                                                @0xa54cc5
//	[falls into the shared cases-33/71/72 cleanup tail @0xa54ce8 ->
//	 CanSendExclRequest -> on failure shows a Notice + jmp default; on
//	 success falls through to the shared send tail with a TRAILING
//	 Encode4(get_update_time) — confirming updateTimeFirst=FALSE for v84,
//	 matching the already-established gms_v84 megaphone-family gate]
//
// Wire (v84): count(byte) + count×line(str) + whisper(bool) +
// updateTime(uint32 trailing). Matches ItemUseTripleMegaphone.Encode(
// updateTimeFirst=false) EXACTLY — no struct fix needed.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTripleMegaphone version=gms_v84 ida=0xa54a2f
func TestItemUseTripleMegaphoneByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItemUseTripleMegaphone(false)
	input.lines = []string{"L1", "L2"}
	input.whisper = true
	input.updateTime = 24680
	expected := []byte{
		0x02,                 // count=2
		0x02, 0x00, 'L', '1', // line[0]
		0x02, 0x00, 'L', '2', // line[1]
		0x01,                   // whisper=true
		0x68, 0x60, 0x00, 0x00, // updateTime=24680 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 item use triple megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa9fef9. jumptable case 60
// (@0xa9ffbc, alloc 0xD8h dialog). GetResult-equivalent call sub_8378B0
// @0xaa0048 takes the same 4-out-param shape (3 line pointers + whisper)
// established for v83/v84/v95; TrimRight/TrimLeft validation on all 3
// lines (>60 chars -> Notice SP idx 0x11B, no send) matches. The encode
// sequence past validation:
//
//	count = 3 if !IsEmpty(line3), else 2 if !IsEmpty(line2), else 1  @0xaa0136-0xaa0157
//	Encode1(count)                                          @0xaa0160
//	loop count times: EncodeStr(line[i])                            @0xaa0189 (loop body)
//	Encode1(whisper)                                                @0xaa019d
//	[falls into the shared cases-33/71/72 cleanup tail @0xaa01c0 ->
//	 CanSendExclRequest -> NO trailing update_time write, confirming
//	 updateTimeFirst=TRUE for v87, matching the outer function-header
//	 proof (get_update_time encoded FIRST, before Encode2/Encode4(itemId))
//	 and the already-established gms_v87 megaphone-family gate]
//
// Wire (v87): count(byte) + count×line(str) + whisper(bool), no trailing
// updateTime. Matches ItemUseTripleMegaphone.Encode(updateTimeFirst=true)
// EXACTLY — no struct fix needed.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTripleMegaphone version=gms_v87 ida=0xa9fef9
func TestItemUseTripleMegaphoneByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItemUseTripleMegaphone(true)
	input.lines = []string{"L1", "L2"}
	input.whisper = true
	expected := []byte{
		0x02,                 // count=2
		0x02, 0x00, 'L', '1', // line[0]
		0x02, 0x00, 'L', '2', // line[1]
		0x01, // whisper=true
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 item use triple megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xaef2f5. JMS's
// get_consume_cash_item_type case-numbering DIVERGES entirely from GMS:
// this function's jumptable (jpt_AEF3A8) places Triple Megaphone at case
// 38 (@0xaef3af, alloc 0xD8h — the SAME allocation size as gms_v87's case
// 60), NOT case 60/12/13/15. Confirmed by structural match (not just size):
// the case-38 body allocates the dialog, calls DoModal, then a
// GetResult-equivalent (sub_866935 @0xaef44d) taking the same 4-out-param
// shape (3 line pointers + whisper via nType/var_1C/var_20/s), followed by
// TrimRight/TrimLeft validation on all 3 lines (>60 chars -> Notice SP idx
// 0x10A, no send) — byte-identical control flow to the already-verified
// gms_v83/v84/v87/v95 Triple Megaphone cases. The encode sequence past
// validation:
//
//	count = 3 if !IsEmpty(line3), else 2 if !IsEmpty(line2), else 1  @0xaef52e-0xaef54b
//	Encode1(count)                                          @0xaef558
//	loop count times: EncodeStr(line[i])                            @0xaef57c (loop body)
//	Encode1(whisper)                                                @0xaef592
//	[falls into the shared cases-43/44/50 cleanup tail @0xaef5b4 -> no
//	 trailing update_time write, confirming updateTimeFirst=TRUE for jms,
//	 matching the outer function-header proof (get_update_time encoded
//	 FIRST @0xaef36c, before Encode2/Encode4(itemId))]
//
// Wire (jms185): count(byte) + count×line(str) + whisper(bool), no
// trailing updateTime. Matches ItemUseTripleMegaphone.Encode(
// updateTimeFirst=true) EXACTLY — no struct fix needed. The jms case-38
// dispatch target is a codec-external fact (which raw switch value routes
// to this body); it is NOT modeled in the struct, so no version-gate is
// required in Go — only the wire SHAPE (already TRUE for MajorVersion>=87)
// matters to the codec.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseTripleMegaphone version=jms_v185 ida=0xaef2f5
func TestItemUseTripleMegaphoneByteOutputJMS185(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := NewItemUseTripleMegaphone(true)
	input.lines = []string{"L1", "L2"}
	input.whisper = true
	expected := []byte{
		0x02,                 // count=2
		0x02, 0x00, 'L', '1', // line[0]
		0x02, 0x00, 'L', '2', // line[1]
		0x01, // whisper=true
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("jms185 item use triple megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

func TestItemUseTripleMegaphoneRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		lines   []string
		whisper bool
	}{
		{"lines_1", []string{"only line"}, false},
		{"lines_2", []string{"line one", "line two"}, true},
		{"lines_3", []string{"line one", "line two", "line three"}, false},
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				// task-123 phase 3: matches the production gate exactly (see
				// item_use_megaphone_test.go for the IDA citation).
				updateTimeFirst := v.MajorVersion >= 87
				input := NewItemUseTripleMegaphone(updateTimeFirst)
				input.lines = tc.lines
				input.whisper = tc.whisper
				if !updateTimeFirst {
					input.updateTime = 24680
				}
				output := NewItemUseTripleMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if len(output.Lines()) != len(input.Lines()) {
					t.Fatalf("lines count: got %d, want %d", len(output.Lines()), len(input.Lines()))
				}
				for i := range input.lines {
					if output.Lines()[i] != input.Lines()[i] {
						t.Errorf("line %d: got %q, want %q", i, output.Lines()[i], input.Lines()[i])
					}
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
