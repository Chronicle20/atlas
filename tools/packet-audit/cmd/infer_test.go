package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// inferDecomp is the live base decompile for 0x100: a clientbound switch whose
// case 1 reads Decode4 and case 2 reads Decode2, so #One (Decode4) -> case 1 and
// #Two (Decode2) -> case 2.
const inferDecomp = "void __thiscall Foo::OnBar(Foo *this, CInPacket *a2)\n{\n" +
	"  switch ( CInPacket::Decode1(a2) )\n  {\n" +
	"    case 1:\n      CInPacket::Decode4(a2);\n      break;\n" +
	"    case 2:\n      CInPacket::Decode2(a2);\n      break;\n  }\n}\n"

func TestInferRunProposes(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x100": inferDecomp}}
	dir := t.TempDir()
	out := filepath.Join(dir, "proposal.json")
	if code := inferRun(inferOpts{Baseline: "testdata/infer_mini.json", Out: out, MinConfidence: 0.6, DescentDepth: 4}, fc, io.Discard); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var p struct {
		Proposals map[string]struct {
			Dispatch   []idasrc.Selector `json:"dispatch"`
			Confidence float64           `json:"confidence"`
		} `json:"proposals"`
	}
	b, _ := os.ReadFile(out)
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(p.Proposals["Foo::OnBar#One"].Dispatch) != 1 || p.Proposals["Foo::OnBar#One"].Dispatch[0].Case != 1 {
		t.Errorf("#One proposal = %+v, want case 1", p.Proposals["Foo::OnBar#One"].Dispatch)
	}
	if len(p.Proposals["Foo::OnBar#Two"].Dispatch) != 1 || p.Proposals["Foo::OnBar#Two"].Dispatch[0].Case != 2 {
		t.Errorf("#Two proposal = %+v, want case 2", p.Proposals["Foo::OnBar#Two"].Dispatch)
	}

	// baseline unchanged:
	before, _ := os.ReadFile("testdata/infer_mini.json")
	_ = inferRun(inferOpts{Baseline: "testdata/infer_mini.json", Out: out, MinConfidence: 0.6, DescentDepth: 4}, fc, io.Discard)
	after, _ := os.ReadFile("testdata/infer_mini.json")
	if !bytes.Equal(before, after) {
		t.Error("infer mutated the baseline")
	}
}
