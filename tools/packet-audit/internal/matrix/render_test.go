package matrix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/diff"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// e2eInputs: 2 versions, 3 ops (one verified-less partial, one n-a in v87,
// one conflict), plus one sub-struct report that joins no registry op.
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
			}),
		}},
		Reports: map[string]map[string]LoadedReport{
			"gms_v83": {"AuthResult": {WriterName: "AuthResult", IDAName: "CLogin::OnCheckPasswordResult", Address: "0x5e1230",
				AtlasFile: "libs/atlas-packet/login/clientbound/auth_result.go", Verdict: diff.VerdictMatch},
				"StatRegistry": {WriterName: "StatRegistry", IDAName: "GW_CharacterStat::Decode", Address: "0x123456",
					AtlasFile: "libs/atlas-packet/model/stat_registry.go", Verdict: diff.VerdictMatch}},
			"gms_v87": {"AuthResult": {WriterName: "AuthResult", IDAName: "CLogin::OnCheckPasswordResult", Address: "0x6f1230",
				AtlasFile: "libs/atlas-packet/login/clientbound/auth_result.go", Verdict: diff.VerdictDeferred}},
		},
		Routed: map[string]map[routeKey]bool{
			"gms_v83": {{0x000, opregistry.DirClientbound}: true},
			"gms_v87": {{0x000, opregistry.DirClientbound}: true, {0x002, opregistry.DirClientbound}: true},
		},
		RoutedAnywhere: map[routeKey]bool{
			{0x000, opregistry.DirClientbound}: true,
			{0x002, opregistry.DirClientbound}: true,
		},
		Evidence: map[evKey]EvidenceStatus{},
		Tier1:    map[string]bool{},
		Markers:  map[evKey]MarkerStatus{},
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
	// ACCOUNT_INFO is absent in v87 registry but routed by v87 template -> conflict.
	if !strings.Contains(got, "ACCOUNT_INFO") {
		t.Error("conflict row missing")
	}
	// Sub-struct section exists with StatRegistry.
	if !strings.Contains(got, "## Sub-structs") || !strings.Contains(got, "StatRegistry") {
		t.Error("sub-struct section missing")
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
