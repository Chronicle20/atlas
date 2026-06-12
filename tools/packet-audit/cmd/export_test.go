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
func (f *fakeMCP) DecompileFunction(_ context.Context, a string) (string, error) { return f.decomp[a], nil }
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
	opts := exportOpts{Version: "gms_v95", Output: out,
		PriorExport: "testdata/gms_v95_mini.json", GeneratedAt: "2026-01-01T00:00:00Z", DescentDepth: 4}
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
