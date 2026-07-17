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
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
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
