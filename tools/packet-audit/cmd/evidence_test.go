package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/evidence"
)

func TestEvidencePinScaffoldsRecord(t *testing.T) {
	out := t.TempDir()
	code := runEvidence([]string{
		"pin",
		"--packet", "login/clientbound/Foo",
		"--version", "gms_v83",
		"--ida", "CLogin::OnFoo",
		"--category", "TIER1-FIXTURE",
		"--export", filepath.Join("testdata", "gms_v95_mini.json"),
		"--evidence-dir", out,
	}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("pin exit = %d", code)
	}
	r, err := evidence.LoadRecord(evidence.RecordPath(out, "gms_v83", "login/clientbound/Foo"))
	if err != nil {
		t.Fatalf("record not written/loadable: %v", err)
	}
	if r.IDA.Function != "CLogin::OnFoo" || r.IDA.DecompileSHA256 == "" || r.IDA.Address == "" {
		t.Errorf("record = %+v", r)
	}
	// Pin must use the same hash code path as --check.
	want, _ := evidence.FunctionHash(filepath.Join("testdata", "gms_v95_mini.json"), "CLogin::OnFoo")
	if r.IDA.DecompileSHA256 != want {
		t.Errorf("hash mismatch pin=%s want=%s", r.IDA.DecompileSHA256, want)
	}
}

func TestEvidencePinUnresolvableCitationFails(t *testing.T) {
	code := runEvidence([]string{
		"pin", "--packet", "x/clientbound/Y", "--version", "gms_v83",
		"--ida", "CLogin::Missing", "--category", "OPAQUE",
		"--export", filepath.Join("testdata", "gms_v95_mini.json"),
		"--evidence-dir", t.TempDir(),
	}, &bytes.Buffer{})
	if code == 0 {
		t.Fatal("pin must fail when the export lacks the cited function")
	}
}

// TestEvidencePinBogusCategory: --category BOGUS must exit 3 and write nothing.
func TestEvidencePinBogusCategory(t *testing.T) {
	out := t.TempDir()
	var stderrBuf bytes.Buffer
	code := runEvidence([]string{
		"pin",
		"--packet", "login/clientbound/Foo",
		"--version", "gms_v83",
		"--ida", "CLogin::OnFoo",
		"--category", "BOGUS",
		"--export", filepath.Join("testdata", "gms_v95_mini.json"),
		"--evidence-dir", out,
	}, &stderrBuf)
	if code != 3 {
		t.Fatalf("pin with bogus category: exit = %d (want 3); stderr: %s", code, stderrBuf.String())
	}
	// Must write nothing.
	p := evidence.RecordPath(out, "gms_v83", "login/clientbound/Foo")
	if _, err := evidence.LoadRecord(p); err == nil {
		t.Error("pin with bogus category must not write any file")
	}
}
