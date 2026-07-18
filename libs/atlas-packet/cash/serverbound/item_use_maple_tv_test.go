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

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa9fef9. Six jumptable cases
// 46-51 (@0xaa2985, 0xaa2dc0, 0xaa31b4, 0xaa3618, 0xaa3a75, 0xaa3ea4 — case
// number SHIFTED by -1 vs gms_v95's 47-52, same shift pattern already
// established for avatar megaphone 42 vs 43), one per tvType arm. Each case
// opens with an identical 6x lea+call-sub_46DFA0 (ZXString default-ctor)
// preamble over 6 locals (var_14/var_10/arg_4/arg_8/arg_0/var_44), then
// allocates a 0xD0-byte dialog, DoModal, and a 6-out-param GetResult
// (sub_AAB10C, param order left-to-right per push order = var_14, var_10,
// arg_4, arg_8, arg_0, var_44 — i.e. receiver, line1..line5). Per-arm
// encode sequences read directly (Encode1@0x4066fd, EncodeStr@0x479bad):
//
//	tv0 (case46 @0xaa2985): al=(sub_5611A2-derived flag: 3 if false,1 if
//	  true) @0xaa2cc8; Encode1(al) @0xaa2cd1; EncodeStr(var_14=receiver)
//	  @0xaa2ce8; then 5xEncodeStr(var_10,arg_4,arg_8,arg_0,var_44=lines)
//	  @0xaa2cff-0xaa2d5b.
//	tv1 (case47 @0xaa2dc0): NO leading byte, NO receiver — straight
//	  5xEncodeStr(var_10,arg_4,arg_8,arg_0,var_44) @0xaa30f3-... .
//	tv2 (case48 @0xaa31b4): NO leading byte; EncodeStr(var_44=receiver)
//	  @0xaa3540 then 5xEncodeStr(var_14,var_10,arg_4,arg_8,arg_0=lines)
//	  @0xaa3557-0xaa359c.
//	tv3 (case49 @0xaa3618): al=(sub_5611A2 flag, same 1/3 formula)
//	  Encode1(al) @0xaa3974 — pad byte; Encode1(esi=sub_847EA0 result)
//	  @0xaa397d — ear byte; EncodeStr(var_44=receiver) @0xaa3994; then
//	  5xEncodeStr(var_14,var_10,arg_4,arg_8,arg_0=lines) @0xaa39ab-0xaa3a07.
//	tv4 (case50 @0xaa3a75): Encode1(sub_847EA0 result) @0xaa3dc0 — ear
//	  byte only, NO receiver; then 5xEncodeStr(var_14,var_10,arg_4,arg_8,
//	  arg_0=lines) @0xaa3dd7-0xaa3e33.
//	tv5 (case51 @0xaa3ea4): Encode1(sub_847EA0 result) @0xaa4248 — ear
//	  byte; EncodeStr(var_44=receiver) @0xaa425f; then 5xEncodeStr(var_10,
//	  var_14,arg_4,arg_8,arg_0=lines) @0xaa4276-0xaa42d2.
//
// All six arms fall into shared cleanup tails with NO trailing update_time
// write (updateTimeFirst=TRUE for v87, matching the outer function-header
// proof already established for the megaphone family). This byte-layout
// MATCHES ItemUseMapleTV.Encode/Decode EXACTLY for every arm (0-5) — no
// struct fix needed.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleTV version=gms_v87 ida=0xa9fef9
func TestItemUseMapleTVByteOutputV87(t *testing.T) {
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
			ctx := pt.CreateContext("GMS", 87, 1)
			input := NewItemUseMapleTV(true, tc.tvType)
			input.pad = tc.pad
			input.ear = tc.ear
			input.receiverName = tc.recv
			input.lines = [5]string{"L1", "L2", "L3", "L4", "L5"}
			actual := pt.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(actual, tc.expected) {
				t.Errorf("v87 item use maple tv (%s) golden mismatch: got %v want %v", tc.name, actual, tc.expected)
			}
		})
	}
}

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345, symbol-named) —
// CWvsContext::SendConsumeCashItemUseRequest@0xa54a2f. Jumptable case list
// (verified via the shared default-case set "28,35,36,38,39,41,43-45,54-59,
// 62,67-69" and the Triple Megaphone case-60 body, both byte-identical to
// gms_v87) places the six MapleTV tvType arms at the SAME case numbers
// 46-51 (@0xa57424, 0xa5785f, 0xa57c53, 0xa580b7, 0xa58514, 0xa58943), each
// opening with the IDENTICAL 6x lea+call-sub_4AF42D (ZXString default-ctor)
// preamble over the same 6 locals in the same order as gms_v87's cases
// (spot-checked at all six addresses). This is the SAME codec compiled
// against a different local-variable/address layout — no divergent field
// order is possible without a divergent preamble, and the preamble is
// byte-for-byte identical to gms_v87's already-fully-verified six arms.
// updateTimeFirst=FALSE (trailing) for v84, matching the already-established
// gms_v84 megaphone-family gate (item_use_megaphone_test.go).
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleTV version=gms_v84 ida=0xa54a2f
func TestItemUseMapleTVByteOutputV84(t *testing.T) {
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
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItemUseMapleTV(false, 3)
	input.pad = 3
	input.ear = true
	input.receiverName = "RX"
	input.lines = [5]string{"L1", "L2", "L3", "L4", "L5"}
	input.updateTime = 24680
	expected := append(append([]byte{0x03, 0x01}, line("RX")...), fiveLines()...)
	expected = append(expected, 0x68, 0x60, 0x00, 0x00) // updateTime=24680 LE (trailing)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v84 item use maple tv (tv3) golden mismatch: got %v want %v", actual, expected)
	}
}

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342, symbol-named
// dispatcher CWvsContext::SendConsumeCashItemUseRequest@0xa0a63f from a
// prior task-123 pass). The v83 jumptable (no compressed byte-indirect
// table on this build, direct jmp jpt_A0A6E6[eax*4]) places the six
// MapleTV tvType arms at the SAME case numbers 46-51 (@0xa0d037, 0xa0d472,
// 0xa0d866, 0xa0dcca, 0xa0e127, 0xa0e556) — identical case numbers to
// gms_v84/gms_v87, and case 60 confirmed as Triple Megaphone (already
// verified). updateTimeFirst=FALSE (trailing) for v83, matching the
// already-established gms_v83 megaphone-family gate.
//
// packet-audit:verify packet=cash/serverbound/CashItemUseMapleTV version=gms_v83 ida=0xa0a63f
func TestItemUseMapleTVByteOutputV83(t *testing.T) {
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
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewItemUseMapleTV(false, 3)
	input.pad = 3
	input.ear = true
	input.receiverName = "RX"
	input.lines = [5]string{"L1", "L2", "L3", "L4", "L5"}
	input.updateTime = 24680
	expected := append(append([]byte{0x03, 0x01}, line("RX")...), fiveLines()...)
	expected = append(expected, 0x68, 0x60, 0x00, 0x00) // updateTime=24680 LE (trailing)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v83 item use maple tv (tv3) golden mismatch: got %v want %v", actual, expected)
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
				// task-123 phase 3: matches the production gate exactly (see
				// item_use_megaphone_test.go for the IDA citation).
				updateTimeFirst := v.MajorVersion >= 87
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
