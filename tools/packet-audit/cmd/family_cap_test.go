package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A phantom dispatchers/*.yaml family with no discrete implementation (no arms
// in run.go) and no families.yaml/baseline entry MUST fail the family-cap guard.
// A discrete-implemented family (arms present) passes. (FR-5.1 / task-169 T2.5)
func TestFamilyCapPhantomFailsDiscretePasses(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "covered.yaml"), "fname: CReal::OnResult\n")
	writeYAML(t, filepath.Join(dir, "zzz_test_family.yaml"), "fname: CPhantom::OnResult\n")
	fp := filepath.Join(t.TempDir(), "families.yaml")
	writeYAML(t, fp, "dispatchers: []\n")
	// CReal has a discrete #-suffixed case arm in run.go; CPhantom does not.
	runGo := filepath.Join(t.TempDir(), "run.go")
	writeYAML(t, runGo, "\tcase \"CReal::OnResult#A\":\n\tcase \"CReal::OnResult#B\":\n")

	cfg := dispatcherLintConfig{DispatchersDir: dir, FamiliesPath: fp, RunGo: runGo}

	vs, err := checkFamilyCap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("want exactly 1 violation (the phantom), got %d: %+v", len(vs), vs)
	}
	if familyOfViolation(vs[0]) != "CPhantom::OnResult" {
		t.Fatalf("violation family = %q; want CPhantom::OnResult", familyOfViolation(vs[0]))
	}
	if vs[0].inv != "FAM-CAP" {
		t.Fatalf("inv = %q; want FAM-CAP", vs[0].inv)
	}
}

// A family with no arms but listed in families.yaml (graduated) passes.
func TestFamilyCapFamiliesListedPasses(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "grad.yaml"), "fname: CGrad::OnResult\n")
	fp := filepath.Join(t.TempDir(), "families.yaml")
	writeYAML(t, fp, "dispatchers:\n  - CGrad::OnResult\n")

	cfg := dispatcherLintConfig{DispatchersDir: dir, FamiliesPath: fp}
	vs, err := checkFamilyCap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Fatalf("families.yaml-listed family should pass, got %+v", vs)
	}
}

// A dispatchers/*.yaml with no `fname:` is a violation (can't confirm capping).
func TestFamilyCapMissingFnameFails(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "broken.yaml"), "writer: Foo\n")
	fp := filepath.Join(t.TempDir(), "families.yaml")
	writeYAML(t, fp, "dispatchers: []\n")
	cfg := dispatcherLintConfig{DispatchersDir: dir, FamiliesPath: fp}
	vs, err := checkFamilyCap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].inv != "FAM-CAP" {
		t.Fatalf("missing-fname yaml should produce one FAM-CAP violation, got %+v", vs)
	}
}

// The real repo tree passes the family-cap guard (every dispatchers/*.yaml is
// discrete-implemented). Run from repo root, so DispatchersDir/FamiliesPath
// resolve relative to it.
func TestFamilyCapRealTreeClean(t *testing.T) {
	// defaultDispatcherLintConfig uses repo-root-relative paths; from the cmd
	// test cwd they resolve up the tree.
	cfg := defaultDispatcherLintConfig()
	cfg.DispatchersDir = filepath.Join("..", "..", "..", "docs", "packets", "dispatchers")
	cfg.FamiliesPath = filepath.Join("..", "..", "..", "docs", "packets", "evidence", "families.yaml")
	cfg.RunGo = "run.go"
	if _, err := os.Stat(cfg.DispatchersDir); err != nil {
		t.Skipf("dispatchers dir not reachable from test cwd: %v", err)
	}
	vs, err := checkFamilyCap(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Fatalf("real tree must pass family-cap guard; got %+v", vs)
	}
}
