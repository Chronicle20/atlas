package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realDocFreshnessConfig points the freshness check at the committed repo docs,
// resolved up from the cmd test cwd (tools/packet-audit/cmd).
func realDocFreshnessConfig() docFreshnessConfig {
	up := func(parts ...string) string {
		return filepath.Join(append([]string{"..", "..", ".."}, parts...)...)
	}
	return docFreshnessConfig{
		ProcessMD:    up("docs", "packets", "PROCESS.md"),
		BaselineYAML: up("docs", "packets", "dispatcher-lint-baseline.yaml"),
		FamiliesYAML: up("docs", "packets", "evidence", "families.yaml"),
		WorkflowYML:  up(".github", "workflows", "packet-matrix.yml"),
	}
}

// The committed tree's PROCESS.md facts block must agree with the tool's ground
// truth. (task-169 T4.5 / FR-2.3 — the passing direction.)
func TestDocFreshnessRealTreePasses(t *testing.T) {
	cfg := realDocFreshnessConfig()
	if _, err := os.Stat(cfg.ProcessMD); err != nil {
		t.Skipf("PROCESS.md not reachable from test cwd: %v", err)
	}
	var stderr bytes.Buffer
	if code := docFreshnessRun(cfg, &bytes.Buffer{}, &stderr); code != 0 {
		t.Fatalf("real tree must pass doc-freshness; exit %d:\n%s", code, stderr.String())
	}
}

// Mutating the documented version_count away from matrix.VersionKeys must fail.
// (task-169 T4.5 / FR-2.3 — the failing direction.)
func TestDocFreshnessDetectsVersionCountDrift(t *testing.T) {
	cfg := realDocFreshnessConfig()
	if _, err := os.Stat(cfg.ProcessMD); err != nil {
		t.Skipf("PROCESS.md not reachable from test cwd: %v", err)
	}
	orig, err := os.ReadFile(cfg.ProcessMD)
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(orig, []byte("version_count: 9"), []byte("version_count: 5"), 1)
	if bytes.Equal(mutated, orig) {
		t.Fatal("fixture setup: 'version_count: 9' not found in PROCESS.md")
	}
	dir := t.TempDir()
	procCopy := filepath.Join(dir, "PROCESS.md")
	if err := os.WriteFile(procCopy, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.ProcessMD = procCopy

	var stderr bytes.Buffer
	if code := docFreshnessRun(cfg, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatal("mutated version_count must fail doc-freshness; got exit 0")
	}
	if !strings.Contains(stderr.String(), "version_count") {
		t.Errorf("expected a version_count divergence on stderr, got:\n%s", stderr.String())
	}
}

// Mutating a documented version_key value must fail too.
func TestDocFreshnessDetectsVersionKeyDrift(t *testing.T) {
	cfg := realDocFreshnessConfig()
	if _, err := os.Stat(cfg.ProcessMD); err != nil {
		t.Skipf("PROCESS.md not reachable from test cwd: %v", err)
	}
	orig, err := os.ReadFile(cfg.ProcessMD)
	if err != nil {
		t.Fatal(err)
	}
	mutated := bytes.Replace(orig, []byte("- gms_v84"), []byte("- gms_v85"), 1)
	if bytes.Equal(mutated, orig) {
		t.Fatal("fixture setup: '- gms_v84' not found in PROCESS.md")
	}
	dir := t.TempDir()
	procCopy := filepath.Join(dir, "PROCESS.md")
	if err := os.WriteFile(procCopy, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.ProcessMD = procCopy
	var stderr bytes.Buffer
	if code := docFreshnessRun(cfg, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatal("mutated version_keys must fail doc-freshness; got exit 0")
	}
	if !strings.Contains(stderr.String(), "version_keys") {
		t.Errorf("expected a version_keys divergence on stderr, got:\n%s", stderr.String())
	}
}

// A documented CI gate that isn't present in the workflow must fail (doc claims
// a gate CI does not run).
func TestDocFreshnessDetectsMissingCIGate(t *testing.T) {
	cfg := realDocFreshnessConfig()
	if _, err := os.Stat(cfg.WorkflowYML); err != nil {
		t.Skipf("workflow not reachable from test cwd: %v", err)
	}
	wf, err := os.ReadFile(cfg.WorkflowYML)
	if err != nil {
		t.Fatal(err)
	}
	// Drop the operations --check step from a workflow copy while PROCESS.md
	// still documents operations-check.
	mutated := bytes.Replace(wf, []byte("operations --check"), []byte("operations SKIPPED"), 1)
	if bytes.Equal(mutated, wf) {
		t.Fatal("fixture setup: 'operations --check' not found in workflow")
	}
	dir := t.TempDir()
	wfCopy := filepath.Join(dir, "packet-matrix.yml")
	if err := os.WriteFile(wfCopy, mutated, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.WorkflowYML = wfCopy
	var stderr bytes.Buffer
	if code := docFreshnessRun(cfg, &bytes.Buffer{}, &stderr); code == 0 {
		t.Fatal("a documented CI gate absent from the workflow must fail; got exit 0")
	}
	if !strings.Contains(stderr.String(), "ci_gates") {
		t.Errorf("expected a ci_gates divergence on stderr, got:\n%s", stderr.String())
	}
}
