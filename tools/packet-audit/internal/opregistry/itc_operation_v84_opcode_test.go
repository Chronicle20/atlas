package opregistry

import (
	"path/filepath"
	"testing"
)

// TestItcOperationV84Opcode is the wire-divergence proof for task-102.
//
// The CSV/registry seeded gms_v84 ITC_OPERATION (serverbound) at opcode 253
// (0xFD) — the v83 opcode carried over unshifted (the CSVs have no v84
// column). IDA disproves that value: in GMS_v84.1_U_DEVM.exe (IDA port 13337)
// the ITC_OPERATION dispatcher sender CITC::OnRegisterSaleEntry @0x5aefd2
// builds COutPacket with `push 104h` @0x5aefff, i.e. opcode 0x104 (260). All
// 16 sibling CITC senders (OnSaleCurrentItem @0x5af1db, OnBuy @0x5af9fa, ...
// OnMoveITCPurchaseItemLtoS @0x5aff86) and the two inlined dialog handlers
// (CITCWnd_Tab::OnButtonClicked @0x5c77ca, CITCBidAuctionDlg::OnButtonClicked
// @0x5d3ec7) build COutPacket(260) as well — proving 0x104 is the v84 wire
// opcode.
//
// This is the same +7 region shift task-102 already corrected for the two
// sibling MTS standalone ops ITC_STATUS_CHARGE (251 -> 258) and
// ITC_QUERY_CASH_REQUEST (252 -> 259), and the task-092 csv-carryover class
// CATCH_MONSTER_WITH_ITEM (251 -> 257). The COutPacket opcode is the wire
// ground truth (VERIFYING_A_PACKET.md §10: "distrust IDB names ... the
// csv-seeded registry opcode can also be off-by-one").
func TestItcOperationV84Opcode(t *testing.T) {
	// Load the REAL committed v84 registry (not testdata) so this test guards
	// the live wire value, not a fixture.
	path := filepath.Join("..", "..", "..", "..", "docs", "packets", "registry", "gms_v84.yaml")
	v, err := LoadVersion(path)
	if err != nil {
		t.Fatalf("LoadVersion(%s): %v", path, err)
	}
	e, ok := v.Lookup("ITC_OPERATION", DirServerbound)
	if !ok {
		t.Fatalf("ITC_OPERATION serverbound not found in %s", path)
	}
	// 0x104 = 260, the opcode pushed at 0x5aefff in CITC::OnRegisterSaleEntry.
	if e.Opcode != 0x104 {
		t.Fatalf("ITC_OPERATION v84 opcode = 0x%X (%d), want 0x104 (260) per COutPacket @0x5aefff", e.Opcode, e.Opcode)
	}
	if e.FName != "CITC::OnRegisterSaleEntry" {
		t.Errorf("ITC_OPERATION v84 fname = %q, want CITC::OnRegisterSaleEntry", e.FName)
	}
}
