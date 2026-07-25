package opregistry

import (
	"path/filepath"
	"testing"
)

// TestItcQueryCashRequestV84Opcode is the wire-divergence proof for task-102.
//
// The CSV/registry seeded gms_v84 ITC_QUERY_CASH_REQUEST (serverbound) at opcode
// 252 (0xFC) — the v83 opcode carried over unshifted (the CSVs have no v84
// column). IDA disproves that value: in GMS_v84.1_U_DEVM.exe (IDA port 13337)
// CITC::TrySendQueryCashRequest @0x5af26a builds COutPacket with `push 103h`
// @0x5af284, i.e. opcode 0x103 (259). The function is structurally identical to
// the v83 twin CITC::TrySendQueryCashRequest @0x59eece (which pushes 0FCh) and
// sits at the exact same intra-class offset from CITC::OnStatusCharge
// (0x5af26a-0x5aef76 == 0x59eece-0x59ebda == 0x2F4), proving it is the v84
// TrySendQueryCashRequest — the only structural difference is the opcode immediate.
//
// This is the same class of csv-carryover drift task-092 already corrected for
// v84 CATCH_MONSTER_WITH_ITEM (251 -> 257) and task-102 corrected for the
// sibling ITC_STATUS_CHARGE (251 -> 258). The COutPacket opcode is the wire
// ground truth (VERIFYING_A_PACKET.md §10: "distrust IDB names ... the
// csv-seeded registry opcode can also be off-by-one").
func TestItcQueryCashRequestV84Opcode(t *testing.T) {
	// Load the REAL committed v84 registry (not testdata) so this test guards
	// the live wire value, not a fixture.
	path := filepath.Join("..", "..", "..", "..", "docs", "packets", "registry", "gms_v84.yaml")
	v, err := LoadVersion(path)
	if err != nil {
		t.Fatalf("LoadVersion(%s): %v", path, err)
	}
	e, ok := v.Lookup("ITC_QUERY_CASH_REQUEST", DirServerbound)
	if !ok {
		t.Fatalf("ITC_QUERY_CASH_REQUEST serverbound not found in %s", path)
	}
	// 0x103 = 259, the opcode pushed at 0x5af284 in CITC::TrySendQueryCashRequest.
	if e.Opcode != 0x103 {
		t.Fatalf("ITC_QUERY_CASH_REQUEST v84 opcode = 0x%X (%d), want 0x103 (259) per COutPacket @0x5af284", e.Opcode, e.Opcode)
	}
	if e.FName != "CITC::TrySendQueryCashRequest" {
		t.Errorf("ITC_QUERY_CASH_REQUEST v84 fname = %q, want CITC::TrySendQueryCashRequest", e.FName)
	}
}
