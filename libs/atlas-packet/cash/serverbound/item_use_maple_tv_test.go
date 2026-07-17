package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::SendConsumeCashItemUseRequest@0x9eb3e0. Six CUIMapleTV ctor
// call sites (@0x780fc0) exist in this dispatcher's jumptable, one per
// tvType arm (0x9eb50a jumptable cases 47,48,49,50,51,52). Each case calls
// CUIMapleTV::GetText@0x7949c0 to fetch (receiver, line1..line5); the LOCAL
// variable <-> GetText-out-param mapping is ROTATED differently per case by
// the compiler's register allocator (confirmed by reading each case's own
// GetText push-argument order) — reads below are normalized to the TRUE
// semantic role (receiver / line1..5), not IDA's inherited local names.
//
// Per-arm encode sequences read directly (COutPacket::Encode1@0x415360,
// EncodeStr@0x4841f0):
//
//	tvType0 (case 47, ctor@0x9ee820, nTVType=0): Encode1(hasReceiver flag,
//	  derived from !IsEmpty(receiver)) @0x9eeba4, EncodeStr(receiver)
//	  @0x9eebbe, then 5×EncodeStr(line) @0x9eebd8-0x9eec40.
//	tvType1 (case 48, ctor@0x9eecff, nTVType=1): NO leading byte, NO
//	  receiver — straight 5×EncodeStr(line) @0x9ef059-0x9ef0c1.
//	tvType2 (case 49, ctor@0x9ef180, nTVType=2): NO leading byte;
//	  EncodeStr(receiver) @0x9ef54b then 5×EncodeStr(line) @0x9ef565-0x9ef5cd.
//	  (This arm also gates on IsEmpty(line5) @0x9ef1f3 before encoding — a
//	  send-side validation, not a wire field.)
//	tvType3 (case 50, ctor@0x9ef6a2, nTVType=0 but bWhisper=1 — a distinct
//	  dialog flavor): TWO leading bytes — Encode1(flag = receiver-non-empty
//	  ? 3 : 1) @0x9efa20, Encode1(CUIMapleTV::IsCheckWhisper) @0x9efa2a,
//	  then EncodeStr(receiver) @0x9efa44 and 5×EncodeStr(line) @0x9efa5e-
//	  0x9efac6.
//	tvType4 (case 51, ctor@0x9efb97): ONE leading byte —
//	  Encode1(IsCheckWhisper) @0x9efefb, NO receiver, then 5×EncodeStr(line)
//	  @0x9eff15-... (receiver is fetched via GetText but never encoded).
//	tvType5 (case 52, ctor@0x9f0054): ONE leading byte —
//	  Encode1(IsCheckWhisper) @0x9f040b, EncodeStr(receiver) @0x9f0425,
//	  then 5×EncodeStr(line) @0x9f043f-0x9f04a7.
//
// All six arms fall into shared cleanup tails with NO trailing update_time
// write (updateTimeFirst=TRUE for v95, consistent with the shared function
// header proof). This byte-layout MATCHES ItemUseMapleTV.Encode/Decode
// EXACTLY as already modeled (tvType==3 -> pad+ear byte pair; tvType>=4 ->
// ear byte only; tvType!=1 && tvType!=2 && tvType<3 -> pad byte only;
// tvType!=4 -> receiver string) — no struct fix needed. This promotes the
// cell the v95 pass (commit fbb66ed75) left BLOCKED.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleTV version=gms_v95 ida=0x9eb3e0
func TestItemUseMapleTVByteOutputV95(t *testing.T) {
	line := func(s string) []byte {
		b := []byte{byte(len(s)), 0x00}
		return append(b, []byte(s)...)
	}
	fiveLines := func() []byte {
		var out []byte
		for _, s := range []string{"L1", "L2", "L3", "L4", "L5"} {
			out = append(out, line(s)...)
		}
		return out
	}

	cases := []struct {
		name     string
		tvType   byte
		pad      byte
		ear      bool
		recv     string
		expected []byte
	}{
		{
			name:   "tv0",
			tvType: 0, pad: 1, recv: "RX",
			expected: append(append([]byte{0x01}, line("RX")...), fiveLines()...),
		},
		{
			name:     "tv1",
			tvType:   1,
			expected: fiveLines(),
		},
		{
			name:   "tv2",
			tvType: 2, recv: "RX",
			expected: append(line("RX"), fiveLines()...),
		},
		{
			name:   "tv3",
			tvType: 3, pad: 3, ear: true, recv: "RX",
			expected: append(append([]byte{0x03, 0x01}, line("RX")...), fiveLines()...),
		},
		{
			name:   "tv4",
			tvType: 4, ear: true,
			expected: append([]byte{0x01}, fiveLines()...),
		},
		{
			name:   "tv5",
			tvType: 5, ear: false, recv: "RX",
			expected: append(append([]byte{0x00}, line("RX")...), fiveLines()...),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 95, 1)
			input := NewItemUseMapleTV(true, tc.tvType)
			input.pad = tc.pad
			input.ear = tc.ear
			input.receiverName = tc.recv
			input.lines = [5]string{"L1", "L2", "L3", "L4", "L5"}
			actual := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(actual, tc.expected) {
				t.Errorf("v95 item use maple tv (%s) golden mismatch: got %v want %v", tc.name, actual, tc.expected)
			}
		})
	}
}

func TestItemUseMapleTVRoundTrip(t *testing.T) {
	// tvType drives the conditional prefix; cover every arm.
	cases := []struct {
		name   string
		tvType byte
		ear    bool
		recv   string
	}{
		{"tv0_normal", 0, false, "PartnerA"},     // byte pad + receiver
		{"tv1_star", 1, false, ""},               // no prefix, no receiver
		{"tv2_heart", 2, false, "PartnerB"},      // receiver only
		{"tv3_megassenger", 3, true, "PartnerC"}, // byte + ear + receiver
		{"tv4_star_m", 4, true, ""},              // ear, NO receiver
		{"tv5_heart_m", 5, false, "PartnerD"},    // ear + receiver
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
				input := NewItemUseMapleTV(updateTimeFirst, tc.tvType)
				input.ear = tc.ear
				input.receiverName = tc.recv
				input.lines = [5]string{"l1", "l2", "l3", "l4", "l5"}
				if !updateTimeFirst {
					input.updateTime = 42
				}
				output := NewItemUseMapleTV(updateTimeFirst, tc.tvType)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Ear() != input.Ear() {
					t.Errorf("ear: got %v, want %v", output.Ear(), input.Ear())
				}
				if output.ReceiverName() != input.ReceiverName() {
					t.Errorf("receiverName: got %q, want %q", output.ReceiverName(), input.ReceiverName())
				}
				for i := range input.lines {
					if output.Lines()[i] != input.Lines()[i] {
						t.Errorf("line %d: got %q, want %q", i, output.Lines()[i], input.Lines()[i])
					}
				}
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
