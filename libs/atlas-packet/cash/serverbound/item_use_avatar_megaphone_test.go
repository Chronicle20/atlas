package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::SendConsumeCashItemUseRequest@0x9eb3e0, jumptable case 43
// (@0x9ebd1d, constructs CUIAvatarMegaphone via CAvatarMegaphone-family item
// classification 539xxxx -> get_cashslot_item_type type 43). After the
// CUIAvatarMegaphone dialog returns (CUIAvatarMegaphone::GetText populates
// s1..s4), the encode sequence at @0x9ec1a9-0x9ec213 is:
//
//	EncodeStr(s4) @0x9ec1ad
//	EncodeStr(s3) @0x9ec1c7
//	EncodeStr(s2) @0x9ec1e1
//	EncodeStr(s1) @0x9ec1fb
//	n = CUIAvatarMegaphone::IsCheckWhisper(pIncDlg) @0x9ec204
//	Encode1(n)    @0x9ec20e
//	[falls into the shared cases-33/72/73 cleanup tail — NO trailing
//	 update_time write, confirming updateTimeFirst=TRUE for this arm too]
//
// This DEFINITIVELY cracks the cell the gms_v83 pass left BLOCKED (only
// case identity was narrowed there, not field layout). Four EncodeStr
// calls (message lines) followed by exactly one Encode1 (whisper) — no
// other fields. Wire (v95): line[0..3](str x4) + whisper(bool), no
// trailing updateTime (consistent with the shared function-header
// updateTime-first proof — see ItemUseMegaphone v95 evidence).
//
// packet-audit:verify packet=cash/serverbound/CashItemUseAvatarMegaphone version=gms_v95 ida=0x9eb3e0
func TestItemUseAvatarMegaphoneByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItemUseAvatarMegaphone(true)
	input.lines = [4]string{"a1", "a2", "a3", "a4"}
	input.whisper = true
	expected := []byte{
		0x02, 0x00, 'a', '1', // line[0]
		0x02, 0x00, 'a', '2', // line[1]
		0x02, 0x00, 'a', '3', // line[2]
		0x02, 0x00, 'a', '4', // line[3]
		0x01, // whisper=true
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 item use avatar megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa9fef9, jumptable case 42
// (@0xaa05a4 — case NUMBER SHIFTED vs gms_v95's 43, same shift pattern
// already established for gms_v83; the "cmp esi,offset off_523EB9" head at
// 0xaa05a4 is an unrelated alternate-caller arm sending 4 empty strings +
// Encode1(1), not exercised by the normal dialog flow — the real dialog path
// is the "jnz loc_AA064A" fallthrough). After the CUIAvatarMegaphone-family
// dialog (0xB8-byte alloc @0xaa0659, DoModal @0xaa06ae) returns OK, GetResult
// (sub_848382@0xaa0736) populates 4 line vars, which are curse-filtered via a
// concatenated "%s%s%s%s" Format (0xaa074a) in declaration order, then
// individually re-extracted and encoded in sequence:
//
//	EncodeStr(line1) @0xaa0a0c
//	EncodeStr(line2) @0xaa0a25
//	EncodeStr(line3) @0xaa0a3e
//	EncodeStr(line4) @0xaa0a57
//	Encode1(whisper) @0xaa0a68   (sub_848555, the checkbox getter)
//	[falls to loc_AA01C0 -> CanSendExclRequest -> 0xaa43a8 SendPacketThunk, NO trailing update_time]
//
// Wire (v87): line[0..3](str x4) + whisper(bool) — identical shape to gms_v95.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseAvatarMegaphone version=gms_v87 ida=0xa9fef9
func TestItemUseAvatarMegaphoneByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItemUseAvatarMegaphone(true)
	input.lines = [4]string{"a1", "a2", "a3", "a4"}
	input.whisper = true
	expected := []byte{
		0x02, 0x00, 'a', '1', // line[0]
		0x02, 0x00, 'a', '2', // line[1]
		0x02, 0x00, 'a', '3', // line[2]
		0x02, 0x00, 'a', '4', // line[3]
		0x01, // whisper=true
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 item use avatar megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa54a2f, jumptable case 42
// (@0xa550cc — same case-number shift vs gms_v95's 43 as gms_v83/gms_v87).
// After the avatar-megaphone dialog (0xA8-byte alloc @0xa550db, DoModal
// @0xa55130) returns OK, GetResult (sub_81780F@0xa551b8) populates 4 line
// vars, curse-filtered via a concatenated "%s%s%s%s" Format (0xa551cc) in
// declaration order, then individually re-extracted and encoded in sequence:
//
//	EncodeStr(line1) @0xa5548e
//	EncodeStr(line2) @0xa554a7
//	EncodeStr(line3) @0xa554c0
//	EncodeStr(line4) @0xa554d9
//	Encode1(whisper) @0xa554ea   (sub_8179E2, the checkbox getter)
//	[falls to loc_A5504B -> loc_A54CE8 "cases 33,71,72" -> CanSendExclRequest
//	 -> loc_A58E47: get_update_time() -> Encode4(result) -> SendPacket]
//	 (TRAILING update_time, same shared tail as Megaphone/SuperMegaphone v84)
//
// Wire (v84): line[0..3](str x4) + whisper(bool) + updateTime(uint32
// trailing) — same shape as gms_v83, matches ItemUseAvatarMegaphone exactly.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseAvatarMegaphone version=gms_v84 ida=0xa54a2f
func TestItemUseAvatarMegaphoneByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItemUseAvatarMegaphone(false)
	input.lines = [4]string{"a1", "a2", "a3", "a4"}
	input.whisper = true
	input.updateTime = 12345
	expected := []byte{
		0x02, 0x00, 'a', '1', // line[0]
		0x02, 0x00, 'a', '2', // line[1]
		0x02, 0x00, 'a', '3', // line[2]
		0x02, 0x00, 'a', '4', // line[3]
		0x01,                   // whisper=true
		0x39, 0x30, 0x00, 0x00, // updateTime=12345 LE (trailing)
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 item use avatar megaphone golden mismatch: got %v want %v", actual, expected)
	}
}

func TestItemUseAvatarMegaphoneRoundTrip(t *testing.T) {
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
				input := NewItemUseAvatarMegaphone(updateTimeFirst)
				input.lines = [4]string{"a1", "a2", "a3", "a4"}
				input.whisper = tc.whisper
				if !updateTimeFirst {
					input.updateTime = 13579
				}
				output := NewItemUseAvatarMegaphone(updateTimeFirst)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
