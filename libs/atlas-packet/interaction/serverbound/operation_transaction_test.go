package serverbound

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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

// TestOperationTransactionByteOutput pins the TRANSACTION body against the ONE
// function that actually sends it on each version: CTradingRoomDlg::OnTrade,
// the mode-17 clientbound receive handler, which replies automatically with the
// client's own {itemId, itemCRC} attestation list. It is NOT a user action and
// it is NOT CCashTradingRoomDlg::Trade — that function is the cash room's Trade
// BUTTON handler and encodes TRADE_CONFIRM (mode 0x11), verified on gms_v83
// @0x485dcd and gms_v95 @0x49e180 (task-205 design.md 1.5, 11.1;
// docs/tasks/task-205-player-trade/version-matrix.md).
//
// Derived read/write order, identical on every version below (IDA-verified per
// version, addresses in the markers):
//
//	COutPacket(<opcode>)        opcode is the PLAYER_INTERACTION serverbound op
//	Encode1(<mode>)             0x14 on GMS v83+, 0x12 on jms_v185 — the
//	                            dispatcher mode byte, not part of this body
//	Encode1(count)              number of {itemId, crc} entries
//	repeat count times:
//	  Encode4(itemId)           TSecType<long>::GetData of the staged slot
//	  Encode4(itemCRC)          CItemInfo::GetItemCRC(itemId)
//
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v83 ida=0x7c20bc
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v84 ida=0x7e8202
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v87 ida=0x815773
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v92 ida=0x744440
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v95 ida=0x763f20
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=jms_v185 ida=0x845ed5
func TestOperationTransactionByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}, {data: 300, crc: 400}}}

	// count(02) | itemId(64000000) crc(c8000000) | itemId(2c010000) crc(90010000)
	const want = "0264000000c80000002c01000090010000"

	for _, c := range []struct {
		name   string
		region string
		major  uint16
	}{
		{"gms_v83", "GMS", 83},
		{"gms_v84", "GMS", 84},
		{"gms_v87", "GMS", 87},
		{"gms_v92", "GMS", 92},
		{"gms_v95", "GMS", 95},
		{"jms_v185", "JMS", 185},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := hex.EncodeToString(input.Encode(l, pt.CreateContext(c.region, c.major, 1))(nil))
			if got != want {
				t.Errorf("%s bytes: got %s, want %s", c.name, got, want)
			}
		})
	}
}

// TestOperationTransactionAbsentOnLegacy pins the version boundary from the
// other side. On gms_v48/v61/v72/v79 the mode-17 (mode-15 on v48, mode-16 on
// v72) confirm receive handler is BODYLESS AND SILENT — it flips a local
// confirm flag and repaints, and constructs no COutPacket at all, so no
// TRANSACTION packet exists on those clients. Full-switch enumeration and
// decompiles: v48 CTradingRoomDlg::OnTrade @0x5e6bd3, v72 @0x6fddec,
// v79 @0x7358c4 (see version-matrix.md). Those matrix cells are therefore n-a,
// recorded in each version's docs/packets/audits/<v>/_unimplemented.json — this
// test only guards the codec against silently growing a legacy body.
func TestOperationTransactionAbsentOnLegacy(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}}}
	for _, c := range []struct {
		name  string
		major uint16
	}{
		{"gms_v48", 48},
		{"gms_v61", 61},
		{"gms_v72", 72},
		{"gms_v79", 79},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := hex.EncodeToString(input.Encode(l, pt.CreateContext("GMS", c.major, 1))(nil))
			if got != "" {
				t.Errorf("%s bytes: got %s, want (empty)", c.name, got)
			}
		})
	}
}
