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
				0x01,                             // count=1
				0x02, 0x00, 'A', '1',              // line[0]
				0x00,                              // whisper=false
			},
		},
		{
			name:    "lines_3",
			lines:   []string{"L1", "L2", "L3"},
			whisper: true,
			expected: []byte{
				0x03,                             // count=3
				0x02, 0x00, 'L', '1',              // line[0]
				0x02, 0x00, 'L', '2',              // line[1]
				0x02, 0x00, 'L', '3',              // line[2]
				0x01,                              // whisper=true
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
		0x02,                             // count=2
		0x02, 0x00, 'L', '1',              // line[0]
		0x02, 0x00, 'L', '2',              // line[1]
		0x01,                              // whisper=true
		0x68, 0x60, 0x00, 0x00,            // updateTime=24680 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 item use triple megaphone golden mismatch: got %v want %v", actual, expected)
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
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
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
