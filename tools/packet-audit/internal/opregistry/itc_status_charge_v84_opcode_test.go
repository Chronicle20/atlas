package opregistry

import (
	"path/filepath"
	"testing"
)

// TestItcStatusChargeV84Opcode is the wire-divergence proof for task-102.
//
// The CSV/registry seeded gms_v84 ITC_STATUS_CHARGE (serverbound) at opcode
// 251 (0xFB) — the v83 opcode carried over unshifted (the CSVs have no v84
// column). IDA disproves that value: in GMS_v84.1_U_DEVM.exe (IDA port 13337)
// CITC::OnStatusCharge @0x5aef76 builds COutPacket with `push 102h` @0x5aef90,
// i.e. opcode 0x102 (258). The function is byte-for-byte identical to the v83
// twin CITC::OnStatusCharge @0x59ebda (which pushes 0FBh), proving it is the
// v84 OnStatusCharge — the only structural difference is the opcode immediate.
//
// This is the same class of csv-carryover drift task-092 already corrected for
// v84 CATCH_MONSTER_WITH_ITEM (251 -> 257). The COutPacket opcode is the wire
// ground truth (VERIFYING_A_PACKET.md §10: "distrust IDB names ... the
// csv-seeded registry opcode can also be off-by-one").
func TestItcStatusChargeV84Opcode(t *testing.T) {
	// Load the REAL committed v84 registry (not testdata) so this test guards
	// the live wire value, not a fixture.
	path := filepath.Join("..", "..", "..", "..", "docs", "packets", "registry", "gms_v84.yaml")
	v, err := LoadVersion(path)
	if err != nil {
		t.Fatalf("LoadVersion(%s): %v", path, err)
	}
	e, ok := v.Lookup("ITC_STATUS_CHARGE", DirServerbound)
	if !ok {
		t.Fatalf("ITC_STATUS_CHARGE serverbound not found in %s", path)
	}
	// 0x102 = 258, the opcode pushed at 0x5aef90 in CITC::OnStatusCharge.
	if e.Opcode != 0x102 {
		t.Fatalf("ITC_STATUS_CHARGE v84 opcode = 0x%X (%d), want 0x102 (258) per COutPacket @0x5aef90", e.Opcode, e.Opcode)
	}
	if e.FName != "CITC::OnStatusCharge" {
		t.Errorf("ITC_STATUS_CHARGE v84 fname = %q, want CITC::OnStatusCharge", e.FName)
	}
}
