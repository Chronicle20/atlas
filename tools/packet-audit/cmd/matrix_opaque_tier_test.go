package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
)

// TestMatrixOpaqueTierViaTypeForWriter verifies Commit-1 Task 3.2 end-to-end:
// a packet whose WriterName differs from its struct name (e.g. "MonsterStatSet"
// vs struct "StatSet") must receive Tier-1 when the tiers.yaml opaque_types list
// includes an opaque sub-type reachable from the struct, WITHOUT requiring a
// packet_prefixes entry for the packet's directory.
//
// Fixture layout (synthetic; mirrors monster/clientbound pattern):
//   - packet-lib/zone/clientbound/stat.go  — struct StatSet, Operation()→OpaqueWriter="OpaquePacket"
//     struct uses an opaque sub-type (OpaqueModel)
//   - audit report WriterName "OpaquePacket", AtlasFile zone/clientbound/stat.go
//   - tiers.yaml opaque_types: [OpaqueModel]   (no packet_prefixes entry)
//
// Expected result: the "zone/clientbound/OpaquePacket" row has Tier1=true.
func TestMatrixOpaqueTierViaTypeForWriter(t *testing.T) {
	root := t.TempDir()

	// Registry: one op ZONE_STAT at clientbound 0x010 so the matrix has a row.
	// FName must match the audit report's IDAName base so rowPacketAndTier finds it.
	regYAML := "- op: ZONE_STAT\n  direction: clientbound\n  opcode: 0x010\n  fname: \"CZone::OnStat\"\n  provenance: csv-import\n"
	regPath := filepath.Join(root, "registry", "gms_v83.yaml")
	mustMkdirAndWrite(t, regPath, regYAML)

	// Audit report: WriterName "OpaquePacket" in zone/clientbound/stat.go
	auditJSON := `{
  "WriterName": "OpaquePacket",
  "IDAName": "CZone::OnStat",
  "Address": "0x111111",
  "Variant": "GMS/v83",
  "BranchDepth": 0,
  "AtlasFile": "libs/atlas-packet/zone/clientbound/stat.go",
  "Rows": [],
  "Verdict": 0,
  "FlatInvalid": false
}`
	auditPath := filepath.Join(root, "audits", "gms_v83", "OpaquePacket.json")
	mustMkdirAndWrite(t, auditPath, auditJSON)

	// Template: routes opcode 0x010 clientbound → routedAnywhere=true.
	// writer name must match the registry op name.
	templateJSON := `{"region":"GMS","majorVersion":83,"minorVersion":1,"socket":{"handlers":[],"writers":[{"opCode":"0x010","writer":"ZoneStat"}]}}`
	mustMkdirAndWrite(t, filepath.Join(root, "templates", "template_gms_83_1.json"), templateJSON)

	// Export: empty/minimal so no export-hash warning fails tests.
	mustMkdirAndWrite(t, filepath.Join(root, "exports", "gms_v83.json"), `{}`)

	// Tiers: OpaqueModel in opaque_types but NO packet_prefixes entry for zone/.
	tiersYAML := "opaque_types:\n  - OpaqueModel\npacket_prefixes: []\npackets: []\n"
	mustMkdirAndWrite(t, filepath.Join(root, "evidence", "tiers.yaml"), tiersYAML)

	// Synthetic packet-lib: struct StatSet with Operation() → const OpaqueWriter = "OpaquePacket"
	// and Encode that calls m.model.Encode (recurse into OpaqueModel which has no Encode → opaque).
	// The Encode signature mirrors the real atlas-packet pattern so the analyzer processes it.
	packetLibSrc := `package clientbound

import "context"

// OpaqueModel has no Encode method — it will be flagged Opaque by Pass-3.
type OpaqueModel struct {
	rawData []byte
}

const OpaqueWriter = "OpaquePacket"

type StatSet struct {
	uniqueId uint32
	stat     OpaqueModel
}

func (m StatSet) Operation() string { return OpaqueWriter }

func (m StatSet) Encode(l interface{}, ctx context.Context) func(map[string]interface{}) []byte {
	return func(opts map[string]interface{}) []byte {
		m.stat.Encode(l, ctx)(opts)
		return nil
	}
}
`
	packetLibPath := filepath.Join(root, "packetlib", "zone", "clientbound", "stat.go")
	mustMkdirAndWrite(t, packetLibPath, packetLibSrc)

	args := []string{
		"--registry-dir", filepath.Join(root, "registry"),
		"--audits-dir", filepath.Join(root, "audits"),
		"--templates-dir", filepath.Join(root, "templates"),
		"--exports-dir", filepath.Join(root, "exports"),
		"--tiers", filepath.Join(root, "evidence", "tiers.yaml"),
		"--packet-lib", filepath.Join(root, "packetlib"),
		"--versions", "gms_v83",
		"--out-dir", filepath.Join(root, "audits"),
	}

	if code := runMatrix(args, os.Stderr); code != 0 {
		t.Fatalf("matrix exit = %d", code)
	}

	// Parse the output to check tier membership.
	raw, err := os.ReadFile(filepath.Join(root, "audits", "status.json"))
	if err != nil {
		t.Fatalf("status.json not written: %v", err)
	}
	var m matrix.Matrix
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal status.json: %v", err)
	}

	// The ZONE_STAT op row must have Tier1=true (packet "zone/clientbound/OpaquePacket"
	// reaches OpaqueModel via TypeForWriter→transitiveRecurseTypes→IsTier1).
	var found bool
	for _, r := range m.Rows {
		if r.Kind == matrix.RowOp && r.Op == "ZONE_STAT" {
			found = true
			if !r.Tier1 {
				t.Errorf("ZONE_STAT row: Tier1=false; want true (opaque consumer via TypeForWriter)")
			}
			break
		}
	}
	if !found {
		// Dump rows for diagnosis.
		var ops []string
		for _, r := range m.Rows {
			ops = append(ops, r.Op+"/"+r.Packet)
		}
		t.Errorf("ZONE_STAT row not found in matrix; rows: %v", strings.Join(ops, ", "))
	}
}

// mustMkdirAndWrite creates parent directories and writes content to dst.
func mustMkdirAndWrite(t *testing.T, dst, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
