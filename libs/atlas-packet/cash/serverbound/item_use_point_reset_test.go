package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUsePointResetRoundTrip(t *testing.T) {
	for _, utf := range []bool{true, false} {
		name := "trailingUpdateTime"
		if utf {
			name = "updateTimeFirst"
		}
		t.Run(name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			input := ItemUsePointReset{to: 2048, from: 64, updateTime: 12345, updateTimeFirst: utf}
			output := ItemUsePointReset{updateTimeFirst: utf}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.To() != input.To() {
				t.Errorf("To: got %d want %d", output.To(), input.To())
			}
			if output.From() != input.From() {
				t.Errorf("From: got %d want %d", output.From(), input.From())
			}
			if !utf && output.UpdateTime() != input.UpdateTime() {
				t.Errorf("UpdateTime: got %d want %d", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}

// TestItemUsePointResetBytesV83 pins the AP/SP-reset sub-body wire for gms_v83.
//
// IDA (live MapleStory_dump.exe v83, port 13342):
// CWvsContext::SendConsumeCashItemUseRequest @0xa0a63f. The sender resolves the
// item type via get_consume_cash_item_type (@0x4863d5), which delegates to
// get_cashslot_item_type (@0x48645b): 5050000 (%10==0) -> type 23 = AP reset;
// 5050001-5050004 (%10 in 1..4) -> type 24 = SP reset. Both branches encode the
// sub-body as exactly two Encode4 (case 23 @s[42783]: Encode4(p_p_pvargDest)
// then Encode4(Unknown); case 24 @s[45981]: Encode4(v567) then Encode4(Unknown))
// followed by the COMMON send tail (LABEL_41): update_time = get_update_time();
// Encode4(update_time); SendPacket. So the point-reset sub-body is:
//
//	Encode4(to) + Encode4(from) + Encode4(update_time)   [updateTime TRAILING]
//
// => updateTimeFirst == false for v83: To-then-From with trailing updateTime,
// CONFIRMED by the IDA decompile above.
// packet-audit:verify packet=cash/serverbound/CashItemUsePointReset version=gms_v83 ida=0xa0a63f
func TestItemUsePointResetBytesV83(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// to=2048 (0x00000800), from=64 (0x00000040), updateTime=0x12345678.
	input := ItemUsePointReset{to: 2048, from: 64, updateTime: 0x12345678, updateTimeFirst: false}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	// 00080000 (to) | 40000000 (from) | 78563412 (updateTime) = 12 bytes.
	if want := "00080000" + "40000000" + "78563412"; got != want {
		t.Errorf("v83 bytes: got %s want %s", got, want)
	}
}

// TestItemUsePointResetBytesV95 pins the AP/SP-reset sub-body wire for gms_v95.
//
// IDA (live GMS_v95.0_U_DEVM.exe, port 13341):
// CWvsContext::SendConsumeCashItemUseRequest @0x9eb3e0, opcode 0x55. Unlike v83,
// the header encodes update_time FIRST:
//
//	COutPacket(0x55); Encode4(get_update_time()); Encode2(nPOS); Encode4(nItemID)
//
// then switches on get_consume_cash_item_type. case 0x17 (AP reset) @s[53432]
// encodes Encode4(s3) then Encode4(s4); case 0x18 (SP reset) @s[57347] encodes
// Encode4(s5) then Encode4(pItemInfo). No trailing update_time in the sub-body
// (it was written in the header). So the v95 point-reset sub-body is exactly:
//
//	Encode4(to) + Encode4(from)                          [updateTime is in header]
//
// => updateTimeFirst == true for v95. Confirms the updateTime-first hypothesis.
//
// NOTE (no packet-audit:verify marker): the shared ItemUsePointReset codec gates
// the trailing updateTime write on the runtime bool updateTimeFirst, which the
// packet-audit analyzer (version-based, not value-based) cannot evaluate, so it
// statically counts three writes and grades the generated gms_v95 report
// FlatInvalid ("atlas extra field"). The read order below is nonetheless
// IDA-verified; the report/marker path for the updateTime-first versions is a
// known tooling gap (see task-4 report), not a codec defect.
func TestItemUsePointResetBytesV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ItemUsePointReset{to: 2048, from: 64, updateTime: 0x12345678, updateTimeFirst: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil))
	// 00080000 (to) | 40000000 (from) = 8 bytes; no trailing updateTime.
	if want := "00080000" + "40000000"; got != want {
		t.Errorf("v95 bytes: got %s want %s", got, want)
	}
}

// TestItemUsePointResetBytesV87 pins the AP/SP-reset sub-body wire for gms_v87.
//
// IDA (live GMSv87_4GB.exe, port 13343):
// CWvsContext::SendConsumeCashItemUseRequest @0xa9fef9, opcode 0x52. Unlike v83
// (and identical to v95), the header encodes update_time FIRST:
//
//	COutPacket(0x52); Encode4(get_update_time()); Encode2(nPOS); Encode4(nItemID)
//
// then switches on get_consume_cash_item_type. case 0x17 (AP reset) @line 1375
// encodes Encode4(to) then Encode4(from); case 0x18 (SP reset) @line 1545 encodes
// Encode4(to) then Encode4(from). The send tail (LABEL_41 @0x...; decompiler line
// 2965) contains NO trailing Encode4(update_time). So the v87 point-reset sub-body
// is exactly:
//
//	Encode4(to) + Encode4(from)                          [updateTime is in header]
//
// => updateTimeFirst == true for v87. This CONTRADICTS the original task-126
// hypothesis (which assumed v87 was trailing, like v83). The header gate was
// therefore corrected from MajorVersion()>=95 to MajorVersion()>=87 in item_use.go
// and character_cash_item_use.go so v87 reads update_time from the header.
//
// NOTE (no packet-audit:verify marker): like the v95 case above, the shared codec
// gates the trailing updateTime on the runtime bool updateTimeFirst, which the
// version-based (not value-based) packet-audit analyzer cannot evaluate; it
// statically counts three writes and grades the generated gms_v87 report
// FlatInvalid ("atlas extra field"). The read order below is nonetheless
// IDA-verified; the marker path for updateTime-first versions is the same known
// tooling gap documented on the v95 fixture, not a codec defect.
func TestItemUsePointResetBytesV87(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ItemUsePointReset{to: 2048, from: 64, updateTime: 0x12345678, updateTimeFirst: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 87, 1))(nil))
	// 00080000 (to) | 40000000 (from) = 8 bytes; no trailing updateTime.
	if want := "00080000" + "40000000"; got != want {
		t.Errorf("v87 bytes: got %s want %s", got, want)
	}
}

// TestItemUsePointResetBytesJMS185 pins the AP/SP-reset sub-body wire for jms_v185.
//
// IDA (live MapleStory_dump_SCY.exe, port 13344):
// CWvsContext::SendConsumeCashItemUseRequest @0xaef2f5, opcode 0x47 (71). As with
// gms_v87/v95, the header encodes update_time FIRST:
//
//	COutPacket(0x47); Encode4(get_update_time()); Encode2(nPOS); Encode4(nItemID)
//
// then switches on get_consume_cash_item_type. case 0x17 (AP reset) @line 1585
// encodes Encode4(to) then Encode4(from); case 0x18 (SP reset) @line 1766 encodes
// Encode4(to) then Encode4(from). The send tail (LABEL_528; decompiler line 1024)
// contains NO trailing Encode4(update_time). So the jms_v185 point-reset sub-body
// is exactly:
//
//	Encode4(to) + Encode4(from)                          [updateTime is in header]
//
// => updateTimeFirst == true for jms_v185. This CONTRADICTS the original task-126
// hypothesis (which assumed JMS was trailing because Region != "GMS"). The header
// gate is region-independent MajorVersion()>=87, so jms_v185 (185) reads
// update_time from the header.
//
// NOTE (no packet-audit:verify marker): same runtime-bool tooling gap as the v95
// and v87 fixtures — the analyzer grades the generated jms_v185 report FlatInvalid.
// Read order is IDA-verified.
func TestItemUsePointResetBytesJMS185(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := ItemUsePointReset{to: 2048, from: 64, updateTime: 0x12345678, updateTimeFirst: true}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("JMS", 185, 1))(nil))
	// 00080000 (to) | 40000000 (from) = 8 bytes; no trailing updateTime.
	if want := "00080000" + "40000000"; got != want {
		t.Errorf("jms_v185 bytes: got %s want %s", got, want)
	}
}
