package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// e2eInputs: 2 versions, 3 ops (one verified-less partial, one conflict with a
// real template-wiring gap, one sub-struct report), plus one sub-struct report
// that joins no registry op.
//
// ACCOUNT_INFO is present in both versions (0x002 in v83, 0x003 in v87).
// v83 routes ACCOUNT_INFO (0x002); v87 does NOT route its 0x003.
// v87 has an audit report for ACCOUNT_INFO (Atlas implements it) → real
// template-wiring-gap conflict (routedElsewhere=true, hasReport=true).
func e2eInputs(t *testing.T) Inputs {
	t.Helper()
	in := Inputs{
		Registry: opregistry.Registry{Versions: map[string]*opregistry.VersionFile{
			"gms_v83": opregistry.NewVersionFile([]opregistry.Entry{
				{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0x000, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
				{Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x002, FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"},
			}),
			"gms_v87": opregistry.NewVersionFile([]opregistry.Entry{
				{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0x000, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
				// ACCOUNT_INFO is present in v87 at a different opcode (0x003).
				// Its opcode is NOT routed in v87's template but IS routed in v83's,
				// so Build will set routedElsewhere=true and grade a coverage-gap conflict.
				{Op: "ACCOUNT_INFO", Direction: opregistry.DirClientbound, Opcode: 0x003, FName: "CLogin::OnAccountInfoResult", Provenance: "csv-import"},
			}),
		}},
		Reports: map[string]map[string]LoadedReport{
			"gms_v83": {
				"AuthResult": {WriterName: "AuthResult", IDAName: "CLogin::OnCheckPasswordResult", Address: "0x5e1230",
					AtlasFile: "libs/atlas-packet/login/clientbound/auth_result.go", Verdict: diff.VerdictMatch},
				"StatRegistry": {WriterName: "StatRegistry", IDAName: "GW_CharacterStat::Decode", Address: "0x123456",
					AtlasFile: "libs/atlas-packet/model/stat_registry.go", Verdict: diff.VerdictMatch},
			},
			"gms_v87": {
				"AuthResult": {WriterName: "AuthResult", IDAName: "CLogin::OnCheckPasswordResult", Address: "0x6f1230",
					AtlasFile: "libs/atlas-packet/login/clientbound/auth_result.go", Verdict: diff.VerdictDeferred},
				// Report present for v87 ACCOUNT_INFO — Atlas implements it — so the
				// coverage-gap conflict fires (not mere absence).
				"AccountInfo": {WriterName: "AccountInfo", IDAName: "CLogin::OnAccountInfoResult", Address: "0x7a4500",
					AtlasFile: "libs/atlas-packet/login/clientbound/account_info.go", Verdict: diff.VerdictMatch},
			},
		},
		Routed: map[string]map[RouteKey]bool{
			// v83 routes LOGIN_STATUS (0x000) and ACCOUNT_INFO (0x002).
			"gms_v83": {{0x000, opregistry.DirClientbound}: true, {0x002, opregistry.DirClientbound}: true},
			// v87 routes LOGIN_STATUS (0x000) only; ACCOUNT_INFO (0x003) is NOT wired.
			"gms_v87": {{0x000, opregistry.DirClientbound}: true},
		},
		Evidence: map[EvKey]EvidenceStatus{},
		Tier1:    map[string]bool{},
		Markers:  map[EvKey]MarkerStatus{},
	}
	return in
}

func TestBuildAndRenderGolden(t *testing.T) {
	m := Build(e2eInputs(t), []string{"gms_v83", "gms_v87"})
	m.ToolSHA = "testsha"
	m.ExportHashes = map[string]string{"gms_v83": "aaa", "gms_v87": "bbb"}

	got := RenderMarkdown(m, []string{"gms_v83", "gms_v87"})
	golden := filepath.Join("testdata", "golden_STATUS.md")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v (run once with UPDATE_GOLDEN=1)", err)
	}
	if got != string(want) {
		t.Errorf("STATUS.md drifted from golden:\n%s", got)
	}

	// Spot-check semantics independent of the golden bytes:
	if !strings.Contains(got, "## Conflicts") {
		t.Error("missing conflicts section")
	}
	// ACCOUNT_INFO is present in v87 (opcode 0x003) but not wired in v87's template,
	// while v83 routes it — a real template-wiring-gap conflict.
	if !strings.Contains(got, "ACCOUNT_INFO") {
		t.Error("conflict row missing")
	}
	// Sub-struct section exists with StatRegistry.
	if !strings.Contains(got, "## Sub-structs") || !strings.Contains(got, "StatRegistry") {
		t.Error("sub-struct section missing")
	}
	// LOGIN_STATUS must show 0x000 in the v83 opcode column.
	loginFound := false
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "| LOGIN_STATUS ") {
			loginFound = true
			if !strings.Contains(line, "0x000") {
				t.Errorf("LOGIN_STATUS row missing 0x000 opcode: %q", line)
			}
		}
	}
	if !loginFound {
		t.Error("LOGIN_STATUS row not found in markdown output")
	}
	// No wall-clock date anywhere (determinism; context.md D2).
	if strings.Contains(got, "20") && strings.Contains(got, "T") && strings.Contains(got, "Z") {
		// crude guard: full ISO timestamps must not appear
		for _, line := range strings.Split(got, "\n") {
			if strings.Contains(line, "Z") && strings.Contains(line, ":") && strings.Contains(line, "T") {
				t.Errorf("timestamp-looking line breaks determinism: %q", line)
			}
		}
	}
}

func TestRenderDeterminism(t *testing.T) {
	m := Build(e2eInputs(t), []string{"gms_v83", "gms_v87"})
	a := RenderMarkdown(m, []string{"gms_v83", "gms_v87"})
	b := RenderMarkdown(Build(e2eInputs(t), []string{"gms_v83", "gms_v87"}), []string{"gms_v83", "gms_v87"})
	if a != b {
		t.Fatal("two consecutive builds differ")
	}
}
