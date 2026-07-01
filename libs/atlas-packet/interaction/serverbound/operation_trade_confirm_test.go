package serverbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v95 ida=0x7646b0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v87 ida=0x8170d5
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v83 ida=0x7c39a0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=jms_v185 ida=0x84830a
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v84 ida=0x7e9ae6
func TestOperationTradeConfirmRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationTradeConfirm{entries: []TradeConfirmEntry{{data: 100, crc: 200}, {data: 300, crc: 400}}}
			output := OperationTradeConfirm{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			entriesPresent := (v.Region == "GMS" && v.MajorVersion >= 83) || v.Region != "GMS"
			if !entriesPresent {
				if len(output.Entries()) != 0 {
					t.Errorf("entries should be absent for %s, got %d", v.Name, len(output.Entries()))
				}
				return
			}
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries length: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i := range input.Entries() {
				if output.Entries()[i].Data() != input.Entries()[i].Data() {
					t.Errorf("entries[%d].data: got %v, want %v", i, output.Entries()[i].Data(), input.Entries()[i].Data())
				}
				if output.Entries()[i].Crc() != input.Entries()[i].Crc() {
					t.Errorf("entries[%d].crc: got %v, want %v", i, output.Entries()[i].Crc(), input.Entries()[i].Crc())
				}
			}
		})
	}
}

// TestOperationTradeConfirmBytes pins the version gate. GMS v79
// CTradingRoomDlg::Trade@0x73709a emits ONLY the miniroom mode byte (0x11) with
// no entry list — the trade-confirm body is bodyless. From GMS v83 onward
// (@0x7c39a0, fixture-verified) the client appends Encode1(count) + per-entry
// Encode4(data)+Encode4(crc). Gate: tradeCrcPresent.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v79 ida=0x73709a
func TestOperationTradeConfirmBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTradeConfirm{entries: []TradeConfirmEntry{{data: 100, crc: 200}, {data: 300, crc: 400}}}

	// v79: bodyless (mode byte is framed by the dispatcher, not this sub-struct)
	got79 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 79, 1))(nil))
	if got79 != "" {
		t.Errorf("v79 bytes: got %s, want (empty)", got79)
	}

	// v83: count(02) | data(LE) crc(LE) | data(LE) crc(LE)
	got83 := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil))
	want83 := "0264000000c80000002c01000090010000"
	if got83 != want83 {
		t.Errorf("v83 bytes: got %s, want %s", got83, want83)
	}
}

// TestOperationTradeConfirmV72Bytes pins the GMS v72 legacy body: bodyless.
// IDA v72 CTradingRoomDlg::Trade (sub_6FF5BF): Encode1(0x10)=mode @0x6ff687 only, no entry list (tradeCrcPresent gate false for v72). Bodyless, == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v72 ida=0x6ff5bf
func TestOperationTradeConfirmV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTradeConfirm{entries: []TradeConfirmEntry{{data: 100, crc: 200}}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "" {
		t.Errorf("v72 bytes: got %s, want (empty)", got)
	}
}
