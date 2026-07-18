package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

type fakeMCP struct {
	addrs  map[string]string
	decomp map[string]string
}

func (f *fakeMCP) GetFunctionByName(_ context.Context, n string) (string, bool, error) {
	a, ok := f.addrs[n]
	return a, ok, nil
}

func (f *fakeMCP) DecompileFunction(_ context.Context, a string) (string, error) {
	return f.decomp[a], nil
}
func (f *fakeMCP) GetCallees(_ context.Context, a string) ([]idasrc.Callee, error) { return nil, nil }
func (f *fakeMCP) StructInfo(_ context.Context, n string) (idasrc.StructLayout, error) {
	return idasrc.StructLayout{}, nil
}

func TestExportRunDeterministic(t *testing.T) {
	fc := &fakeMCP{
		addrs: map[string]string{
			"CLogin::OnFoo": "0x1",
			"CLogin::OnBar": "0x2",
		},
		decomp: map[string]string{
			"0x1": "void CLogin::OnFoo(CLogin *this, CInPacket *a2)\n{\n  CInPacket::Decode4(a2);\n}\n",
			"0x2": "void CLogin::OnBar(CLogin *this, CInPacket *a2)\n{\n  CInPacket::Decode1(a2);\n}\n",
		},
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "gms_v95.json")
	opts := exportOpts{
		Version: "gms_v95", Output: out,
		PriorExport: "testdata/gms_v95_mini.json", GeneratedAt: "2026-01-01T00:00:00Z", DescentDepth: 4,
	}
	if code := exportRun(opts, fc, io.Discard, io.Discard); code != 0 {
		t.Fatalf("exportRun exit = %d", code)
	}
	a, _ := os.ReadFile(out)
	if code := exportRun(opts, fc, io.Discard, io.Discard); code != 0 {
		t.Fatalf("exportRun re-run exit = %d", code)
	}
	b, _ := os.ReadFile(out)
	if !bytes.Equal(a, b) {
		t.Error("export not deterministic across runs")
	}
	var ef struct {
		Functions map[string]json.RawMessage `json:"functions"`
	}
	if err := json.Unmarshal(a, &ef); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(ef.Functions) == 0 {
		t.Fatal("no functions in export")
	}
}

// twoFnMCP harvests exactly CLogin::OnFoo + CLogin::OnBar. The Foo body is
// parameterised so a caller can force the harvest to DIFFER from a previously
// written export.
func twoFnMCP(fooBody string) *fakeMCP {
	return &fakeMCP{
		addrs: map[string]string{"CLogin::OnFoo": "0x1", "CLogin::OnBar": "0x2"},
		decomp: map[string]string{
			"0x1": "void CLogin::OnFoo(CLogin *this, CInPacket *a2)\n{\n  " + fooBody + "\n}\n",
			"0x2": "void CLogin::OnBar(CLogin *this, CInPacket *a2)\n{\n  CInPacket::Decode1(a2);\n}\n",
		},
	}
}

func baseExportOpts(out string) exportOpts {
	return exportOpts{
		Version: "gms_v95", Output: out,
		PriorExport: "testdata/gms_v95_mini.json", GeneratedAt: "2026-01-01T00:00:00Z", DescentDepth: 4,
	}
}

// TestExportRefusesDifferingOverwrite proves the non-destructive default
// (FR-3.2): a second export whose harvest DIFFERS from the committed file
// refuses, leaves the file byte-unchanged, writes <output>.new, and exits
// non-zero.
func TestExportRefusesDifferingOverwrite(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "gms_v95.json")

	// First export establishes the committed file.
	if code := exportRun(baseExportOpts(out), twoFnMCP("CInPacket::Decode4(a2);"), io.Discard, io.Discard); code != 0 {
		t.Fatalf("first export exit = %d", code)
	}
	orig, _ := os.ReadFile(out)

	// A differing harvest (Decode2 instead of Decode4) without --force.
	var stderr bytes.Buffer
	code := exportRun(baseExportOpts(out), twoFnMCP("CInPacket::Decode2(a2);"), io.Discard, &stderr)
	if code == 0 {
		t.Fatalf("differing export without --force should refuse (non-zero); got exit 0")
	}
	after, _ := os.ReadFile(out)
	if !bytes.Equal(orig, after) {
		t.Error("committed export was overwritten despite refusal")
	}
	if _, err := os.Stat(out + ".new"); err != nil {
		t.Errorf("expected <output>.new to be written: %v", err)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("changed")) {
		t.Errorf("expected change summary on stderr, got: %s", stderr.String())
	}
}

// TestExportForceOverwrites proves --force restores today's overwrite.
func TestExportForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "gms_v95.json")
	if code := exportRun(baseExportOpts(out), twoFnMCP("CInPacket::Decode4(a2);"), io.Discard, io.Discard); code != 0 {
		t.Fatalf("first export exit = %d", code)
	}
	orig, _ := os.ReadFile(out)

	fo := baseExportOpts(out)
	fo.Force = true
	if code := exportRun(fo, twoFnMCP("CInPacket::Decode2(a2);"), io.Discard, io.Discard); code != 0 {
		t.Fatalf("--force export exit = %d", code)
	}
	after, _ := os.ReadFile(out)
	if bytes.Equal(orig, after) {
		t.Error("--force did not overwrite the differing export")
	}
	if _, err := os.Stat(out + ".new"); err == nil {
		t.Error("--force must not leave an <output>.new sidecar")
	}
}

// TestExportIdenticalIsIdempotent proves re-running with an identical harvest
// succeeds (no refusal, no .new sidecar) even though the file exists.
func TestExportIdenticalIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "gms_v95.json")
	if code := exportRun(baseExportOpts(out), twoFnMCP("CInPacket::Decode4(a2);"), io.Discard, io.Discard); code != 0 {
		t.Fatalf("first export exit = %d", code)
	}
	if code := exportRun(baseExportOpts(out), twoFnMCP("CInPacket::Decode4(a2);"), io.Discard, io.Discard); code != 0 {
		t.Fatalf("identical re-export should succeed; got exit %d", code)
	}
	if _, err := os.Stat(out + ".new"); err == nil {
		t.Error("identical re-export must not leave an <output>.new sidecar")
	}
}

// TestExportSpliceMergesOneEntry proves --splice merges exactly one function
// entry (updating it) and leaves every other entry byte-unchanged.
func TestExportSpliceMergesOneEntry(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "gms_v95.json")
	if code := exportRun(baseExportOpts(out), twoFnMCP("CInPacket::Decode4(a2);"), io.Discard, io.Discard); code != 0 {
		t.Fatalf("first export exit = %d", code)
	}
	orig, _ := os.ReadFile(out)
	var origDoc struct {
		Functions map[string]json.RawMessage `json:"functions"`
	}
	_ = json.Unmarshal(orig, &origDoc)

	so := baseExportOpts(out)
	so.Splice = "CLogin::OnFoo"
	// Fresh harvest: OnFoo changes to Decode2, OnBar unchanged.
	if code := exportRun(so, twoFnMCP("CInPacket::Decode2(a2);"), io.Discard, io.Discard); code != 0 {
		t.Fatalf("--splice export exit = %d", code)
	}
	merged, _ := os.ReadFile(out)
	var mergedDoc struct {
		Functions map[string]json.RawMessage `json:"functions"`
	}
	if err := json.Unmarshal(merged, &mergedDoc); err != nil {
		t.Fatalf("unmarshal merged: %v", err)
	}
	// The spliced entry changed.
	if bytes.Equal(origDoc.Functions["CLogin::OnFoo"], mergedDoc.Functions["CLogin::OnFoo"]) {
		t.Error("--splice did not update the target entry CLogin::OnFoo")
	}
	// The untouched entry is byte-identical.
	if !bytes.Equal(origDoc.Functions["CLogin::OnBar"], mergedDoc.Functions["CLogin::OnBar"]) {
		t.Error("--splice altered the untouched entry CLogin::OnBar")
	}
	if _, err := os.Stat(out + ".new"); err == nil {
		t.Error("--splice must not leave an <output>.new sidecar")
	}
}
