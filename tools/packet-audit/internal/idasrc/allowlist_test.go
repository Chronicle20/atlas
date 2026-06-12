package idasrc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllowlist_Suppress(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "_unimplemented.json")
	const j = `{"entries":[{"fname":"CWvsContext::OnPartyResult","case":12,"reason":"PQ feature not built"}]}`
	if err := os.WriteFile(p, []byte(j), 0o644); err != nil {
		t.Fatal(err)
	}

	al, err := LoadAllowlist(p)
	if err != nil {
		t.Fatal(err)
	}
	if !al.Suppressed("CWvsContext::OnPartyResult", 12) {
		t.Fatal("case 12 should be suppressed")
	}
	if al.Suppressed("CWvsContext::OnPartyResult", 3) {
		t.Fatal("case 3 should NOT be suppressed")
	}
}

func TestAllowlist_MissingFileIsEmpty(t *testing.T) {
	al, err := LoadAllowlist(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if al.Suppressed("X", 1) {
		t.Fatal("empty allowlist suppressed something unexpected")
	}
}
