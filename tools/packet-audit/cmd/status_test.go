package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A minimal status.json fixture with one verified op, one open gap, one stale
// incomplete, and one n-a sub-struct for gms_v83.
const statusFixture = `{
  "toolSha": "test",
  "exportHashes": {},
  "rows": [
    {"kind":"op","op":"VERIFIED_OP","direction":"clientbound","tier1":false,
     "cells":{"gms_v83":{"state":"verified","opcode":1}}},
    {"kind":"op","op":"OPEN_OP","direction":"clientbound","tier1":false,
     "cells":{"gms_v83":{"state":"incomplete","note":"no audit report","opcode":2}}},
    {"kind":"op","op":"STALE_OP","direction":"clientbound","tier1":false,
     "cells":{"gms_v83":{"state":"incomplete","note":"evidence stale (decompile hash drift)","opcode":3}}},
    {"kind":"sub-struct","packet":"npc/clientbound/Detail","tier1":false,
     "cells":{"gms_v83":{"state":"n-a","note":"disposition","opcode":-1}}}
  ]
}`

func writeStatusFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "status.json")
	if err := os.WriteFile(p, []byte(statusFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStatusRunPrintsSummary(t *testing.T) {
	p := writeStatusFixture(t)
	var out, errBuf bytes.Buffer
	code := statusRun(p, "gms_v83", &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errBuf.String())
	}
	got := out.String()
	for _, want := range []string{
		"packet coverage — gms_v83",
		"verified   1",
		"n-a        1",
		"OPEN_OP",  // open gap listed
		"STALE_OP", // appears in both open gaps and stale
		"stale evidence (1):",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	// status writes nothing to disk beyond the fixture we created.
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Errorf("status must not write files; dir now has %d entries", len(entries))
	}
}

func TestStatusRunUnknownVersion(t *testing.T) {
	p := writeStatusFixture(t)
	var out, errBuf bytes.Buffer
	if code := statusRun(p, "gms_v999", &out, &errBuf); code != 3 {
		t.Fatalf("unknown version should exit 3, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown version") {
		t.Errorf("expected unknown-version error, got %q", errBuf.String())
	}
}
