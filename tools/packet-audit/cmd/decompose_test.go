package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// decomposeFakeMCP is a local MCPClient fake for the decompose command tests.
// decompile maps base address -> Hex-Rays text; decompErr injects per-address
// soft-fails. Mirrors validateFakeMCP, kept local so the tests stay independent.
type decomposeFakeMCP struct {
	decomp    map[string]string
	decompErr map[string]error
}

func (f *decomposeFakeMCP) GetFunctionByName(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}
func (f *decomposeFakeMCP) DecompileFunction(_ context.Context, a string) (string, error) {
	if err := f.decompErr[a]; err != nil {
		return "", err
	}
	return f.decomp[a], nil
}
func (f *decomposeFakeMCP) GetCallees(_ context.Context, _ string) ([]idasrc.Callee, error) {
	return nil, nil
}
func (f *decomposeFakeMCP) StructInfo(_ context.Context, _ string) (idasrc.StructLayout, error) {
	return idasrc.StructLayout{}, nil
}

// onADecomp: the FULL faithful read-order for Foo::OnA — Decode1, Decode4,
// Decode4, DecodeStr. The hand-authored baseline was TRUNCATED to the first two.
const onADecomp = "void __thiscall Foo::OnA(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"  CInPacket::DecodeStr(a2);\n" +
	"}\n"

// onBDecomp: matches the hand-authored Foo::OnB baseline exactly (Decode1, Decode4).
const onBDecomp = "void __thiscall Foo::OnB(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"}\n"

// onDDecomp: mid-stream DIVERGENCE — faithful order is Decode1, Decode4, DecodeStr.
// The hand-authored baseline has Decode2 at index 2; the faithful has DecodeStr.
// Decode2 and DecodeStr are NOT width-equivalent, so this is a genuine divergence.
const onDDecomp = "void __thiscall Foo::OnD(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"  CInPacket::DecodeStr(a2);\n" +
	"}\n"

func TestDecomposeRunUpgradesTruncated(t *testing.T) {
	fc := &decomposeFakeMCP{
		decomp: map[string]string{
			"0x100": onADecomp,
			"0x200": onBDecomp,
			"0x300": "void Foo::OnC() {}\n",
			"0x400": onDDecomp,
		},
		decompErr: map[string]error{},
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "extended.json")
	report := filepath.Join(dir, "report.md")

	code := decomposeRun(decomposeOpts{
		Baseline:     "testdata/decompose_baseline.json",
		AuditDir:     "testdata/decompose_audit",
		Out:          out,
		Report:       report,
		DescentDepth: 4,
	}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}

	// Foo::OnA was non-✅ and truncated (2 hand ops vs 4 faithful) → upgraded,
	// and the OUTPUT baseline's calls now carry the full 4-op faithful order.
	outFns := readCalls(t, out)
	if got := len(outFns["Foo::OnA"]); got != 4 {
		t.Errorf("Foo::OnA upgraded calls = %d ops, want 4", got)
	}
	wantOps := []string{"Decode1", "Decode4", "Decode4", "DecodeStr"}
	for i, op := range wantOps {
		if outFns["Foo::OnA"][i].Op != op {
			t.Errorf("Foo::OnA op[%d] = %q, want %q", i, outFns["Foo::OnA"][i].Op, op)
		}
	}

	// Foo::OnB was ✅ in the audit → NOT processed; unchanged in output (still 2).
	if got := len(outFns["Foo::OnB"]); got != 2 {
		t.Errorf("Foo::OnB calls = %d ops, want 2 (untouched)", got)
	}

	// Foo::OnC#X is a # entry → skipped (needs-dispatch); unchanged in output.
	if got := len(outFns["Foo::OnC#X"]); got != 1 {
		t.Errorf("Foo::OnC#X calls = %d ops, want 1 (untouched)", got)
	}

	// Foo::OnD was non-✅ with a MID-STREAM DIVERGENCE (hand Decode2 vs faithful
	// DecodeStr at index 2) → classified as divergence, output entry UNCHANGED (still
	// 3 hand ops: Decode1, Decode4, Decode2).
	if got := len(outFns["Foo::OnD"]); got != 3 {
		t.Errorf("Foo::OnD diverged calls = %d ops, want 3 (entry must be untouched)", got)
	}
	// The third op must remain Decode2 (the hand-authored value), NOT DecodeStr.
	if len(outFns["Foo::OnD"]) >= 3 {
		if got := outFns["Foo::OnD"][2].Op; got != "Decode2" {
			t.Errorf("Foo::OnD op[2] = %q, want %q (divergence must not overwrite)", got, "Decode2")
		}
	}

	// Report classifications present.
	rb, _ := os.ReadFile(report)
	rs := string(rb)
	for _, want := range []string{"upgraded", "divergence", "needs-dispatch", "Foo::OnA", "Foo::OnC#X", "Foo::OnD"} {
		if !strings.Contains(rs, want) {
			t.Errorf("report missing %q\n%s", want, rs)
		}
	}
	if sectionContains(rs, "upgraded", "Foo::OnA") == false {
		t.Errorf("Foo::OnA not in upgraded section\n%s", rs)
	}
	if sectionContains(rs, "needs-dispatch", "Foo::OnC#X") == false {
		t.Errorf("Foo::OnC#X not in needs-dispatch section\n%s", rs)
	}
	if sectionContains(rs, "divergence", "Foo::OnD") == false {
		t.Errorf("Foo::OnD not in divergence section\n%s", rs)
	}

	// INPUT baseline must be byte-unchanged.
	before, _ := os.ReadFile("testdata/decompose_baseline.json")
	_ = decomposeRun(decomposeOpts{
		Baseline:     "testdata/decompose_baseline.json",
		AuditDir:     "testdata/decompose_audit",
		Out:          out,
		Report:       report,
		DescentDepth: 4,
	}, fc, io.Discard)
	after, _ := os.ReadFile("testdata/decompose_baseline.json")
	if !bytes.Equal(before, after) {
		t.Error("decompose mutated the input baseline")
	}
}

// TestDecomposeRunDivergenceNotOverwritten specifically verifies that a mid-stream
// divergence entry (hand and faithful differ at a common prefix position) is:
//  1. Classified as "divergence" (NOT "upgraded").
//  2. The output entry is byte-identical to the input (NEVER overwritten with the
//     faithful order, because that would silently hide a genuine Atlas wire bug).
func TestDecomposeRunDivergenceNotOverwritten(t *testing.T) {
	fc := &decomposeFakeMCP{
		decomp: map[string]string{
			"0x100": onADecomp,
			"0x200": onBDecomp,
			"0x300": "void Foo::OnC() {}\n",
			"0x400": onDDecomp,
		},
		decompErr: map[string]error{},
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "extended.json")
	report := filepath.Join(dir, "report.md")

	code := decomposeRun(decomposeOpts{
		Baseline:     "testdata/decompose_baseline.json",
		AuditDir:     "testdata/decompose_audit",
		Out:          out,
		Report:       report,
		DescentDepth: 4,
	}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}

	outFns := readCalls(t, out)

	// Foo::OnD: hand was [Decode1, Decode4, Decode2], faithful is [Decode1, Decode4, DecodeStr].
	// The mismatch is at index 2 (Decode2 vs DecodeStr — not width-equivalent).
	// This is a divergence: the output entry MUST remain the hand-authored version.
	if got := len(outFns["Foo::OnD"]); got != 3 {
		t.Errorf("Foo::OnD: output has %d ops, want 3 (entry must not be overwritten)", got)
	}
	wantHandOps := []string{"Decode1", "Decode4", "Decode2"}
	for i, want := range wantHandOps {
		if i >= len(outFns["Foo::OnD"]) {
			t.Errorf("Foo::OnD: op[%d] missing in output", i)
			continue
		}
		if got := outFns["Foo::OnD"][i].Op; got != want {
			t.Errorf("Foo::OnD: op[%d] = %q, want %q (divergence must preserve hand-authored entry)", i, got, want)
		}
	}

	// The report must classify Foo::OnD as divergence, NOT as upgraded.
	rb, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	rs := string(rb)
	if sectionContains(rs, "upgraded", "Foo::OnD") {
		t.Errorf("Foo::OnD must NOT appear in upgraded section — it is a divergence\n%s", rs)
	}
	if !sectionContains(rs, "divergence", "Foo::OnD") {
		t.Errorf("Foo::OnD must appear in divergence section\n%s", rs)
	}
}

// readCalls parses an extended-baseline JSON and returns FName -> raw call list.
func readCalls(t *testing.T, path string) map[string][]struct{ Op string } {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc struct {
		Functions map[string]struct {
			Calls []struct {
				Op string `json:"op"`
			} `json:"calls"`
		} `json:"functions"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	out := map[string][]struct{ Op string }{}
	for fn, f := range doc.Functions {
		var calls []struct{ Op string }
		for _, c := range f.Calls {
			calls = append(calls, struct{ Op string }{Op: c.Op})
		}
		out[fn] = calls
	}
	return out
}

// sectionContains reports whether fname appears under the "## <section>" header.
func sectionContains(report, section, fname string) bool {
	cur := ""
	for _, line := range strings.Split(report, "\n") {
		tl := strings.TrimSpace(line)
		if strings.HasPrefix(tl, "## ") {
			cur = strings.TrimSpace(strings.TrimPrefix(tl, "##"))
			continue
		}
		if cur == section && strings.Contains(line, fname) {
			return true
		}
	}
	return false
}
