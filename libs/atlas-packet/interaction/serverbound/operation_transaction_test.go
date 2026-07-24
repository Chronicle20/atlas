package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v95 ida=0x49e180
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v87 ida=0x494dcb
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v83 ida=0x485dcd
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=jms_v185 ida=0x499b67
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v84 ida=0x489210
func TestOperationTransactionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}, {data: 300, crc: 400}}}
			output := OperationTransaction{}
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

// TestOperationTransactionBytes pins the version gate. GMS v79
// CCashTradingRoomDlg::Trade@0x47e5f5 emits ONLY the miniroom mode byte (0x11)
// with no entry list — the cash-trade transaction body is bodyless. From GMS
// v83 onward (@0x485dcd, fixture-verified) the client appends Encode1(count) +
// per-entry Encode4(data)+Encode4(crc). Gate: tradeCrcPresent.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v79 ida=0x47e5f5
func TestOperationTransactionBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}, {data: 300, crc: 400}}}

	// v79: bodyless
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

// TestOperationTransactionV72Bytes pins the GMS v72 legacy body: bodyless.
// IDA v72 CCashTradingRoomDlg::Trade: the cash trade-room confirm inherits the base CTradingRoomDlg::Trade path (sub_6FF5BF) in v72 — Encode1(mode) only, no entry list (tradeCrcPresent gate false). Bodyless, == v79.
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v72 ida=0x6ff5bf
func TestOperationTransactionV72Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}}}
	got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", 72, 1))(nil))
	if got != "" {
		t.Errorf("v72 bytes: got %s, want (empty)", got)
	}
}
