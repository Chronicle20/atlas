package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// triageFakeMCP is a local MCPClient fake for the triage command tests.
// decomp maps base address -> Hex-Rays text; decompErr injects per-address
// soft-fails. Mirrors decomposeFakeMCP, kept local so the tests stay independent.
type triageFakeMCP struct {
	decomp    map[string]string
	decompErr map[string]error
}

func (f *triageFakeMCP) GetFunctionByName(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}
func (f *triageFakeMCP) DecompileFunction(_ context.Context, a string) (string, error) {
	if err := f.decompErr[a]; err != nil {
		return "", err
	}
	return f.decomp[a], nil
}
func (f *triageFakeMCP) GetCallees(_ context.Context, _ string) ([]idasrc.Callee, error) {
	return nil, nil
}
func (f *triageFakeMCP) StructInfo(_ context.Context, _ string) (idasrc.StructLayout, error) {
	return idasrc.StructLayout{}, nil
}

// flatMismatchDecomp: Foo::OnFlat reads a FLAT order Decode1, Decode4 — no
// branching. The audit Row 1 has AtlasOp Encode4 vs IDAOp Decode1 (a genuine,
// non-width-tolerable op mismatch on a flat handler) → candidate-real-divergence.
const flatMismatchDecomp = "void __thiscall Foo::OnFlat(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"}\n"

// branchDecomp: Foo::OnBranch is the DropDestroy shape — read a leave-type byte,
// then a GUARDED Decode4 inside `if ( v3 == 2 )`. The handler branches on an
// early read → per-mode-branch (the flat compare is invalid).
const branchDecomp = "void __thiscall Foo::OnBranch(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  v3 = CInPacket::Decode1(a2);\n" +
	"  if ( v3 == 2 )\n" +
	"  {\n" +
	"    CInPacket::Decode4(a2);\n" +
	"  }\n" +
	"}\n"

// reprDecomp: Foo::OnRepr reads a FLAT order Decode1, DecodeBuf — no branching.
// The non-OK Row pairs AtlasOp byte with IDAOp buffer (width-tolerable under
// FieldEquivalent) → representation.
const reprDecomp = "void __thiscall Foo::OnRepr(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  CInPacket::DecodeBuffer(a2);\n" +
	"}\n"

// hashModeDecomp: Foo::OnDispatch#HashMode — a flat handler (no branches), so
// ReadsAreConditional returns false. But the FName contains '#', meaning it's a
// per-dispatch slice of a switch handler → must be per-mode-branch (NOT candidate-real).
const hashModeDecomp = "void __thiscall Foo__OnDispatch_HashMode(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"}\n"

// repeatRunDecomp: Foo::OnRepeat — a flat handler whose decompile contains 9
// packet reads in a [Decode4,Decode2,Decode2]×3 pattern. No explicit for/if/switch
// construct survives decompilation, so ReadsAreConditional returns false. But the
// faithful read-order contains a repeating run → per-mode-branch (loop/array unrolled).
const repeatRunDecomp = "void __thiscall Foo__OnRepeat(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode4(a2);\n" +
	"  CInPacket::Decode2(a2);\n" +
	"  CInPacket::Decode2(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"  CInPacket::Decode2(a2);\n" +
	"  CInPacket::Decode2(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"  CInPacket::Decode2(a2);\n" +
	"  CInPacket::Decode2(a2);\n" +
	"}\n"

// branchDepthDecomp: Foo::OnBranchDepth looks flat to the decompiler (no visible
// branches), but its audit JSON carries BranchDepth=1 — the authoritative signal
// that the handler actually branches. The BranchDepth pre-check must fire BEFORE
// the ReadsAreConditional check and classify this as per-mode-branch, not candidate-real.
const branchDepthDecomp = "void __thiscall Foo__OnBranchDepth(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  CInPacket::Decode4(a2);\n" +
	"}\n"

// emptyReadDecomp: Foo::OnEmptyRead — the decompile contains no CInPacket::Decode
// calls, so ResolveLive returns an empty faithful read-order. Empty reads are an
// extraction failure, not a divergence → unverifiable.
const emptyReadDecomp = "void __thiscall Foo__OnEmptyRead(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  g_Logger.Log(\"empty\");\n" +
	"}\n"

// unresolvedReadDecomp: Foo::OnUnresolved — the decompile reads one byte, then
// calls an indirect dispatch that passes the packet pointer. The indirect call
// cannot be resolved → the faithful read-order contains an Unresolved entry →
// unverifiable (faithful read contains Unresolved span).
const unresolvedReadDecomp = "void __thiscall Foo__OnUnresolved(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  CInPacket::Decode1(a2);\n" +
	"  (*v2)(this, a2);\n" +
	"}\n"

func TestTriageRunClassifies(t *testing.T) {
	fc := &triageFakeMCP{
		decomp: map[string]string{
			"0x100": flatMismatchDecomp,
			"0x200": branchDecomp,
			"0x300": reprDecomp,
			// 0x400 (Foo::OnFail) injected as a soft-fail below.
			"0x500": hashModeDecomp,
			"0x600": repeatRunDecomp,
			"0x700": branchDepthDecomp,
			"0x800": emptyReadDecomp,
			"0x900": unresolvedReadDecomp,
		},
		decompErr: map[string]error{
			"0x400": idasrc.NewRPCDecompileError("0x400"),
		},
	}

	dir := t.TempDir()
	report := filepath.Join(dir, "report.md")

	var stdout bytes.Buffer
	code := triageRun(triageOpts{
		Baseline:     "testdata/triage_baseline.json",
		AuditDir:     "testdata/triage_audit",
		Report:       report,
		DescentDepth: 4,
	}, fc, &stdout)
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s", code, stdout.String())
	}

	rb, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	rs := string(rb)

	// Foo::OnFlat: flat handler, genuine op mismatch → candidate-real-divergence.
	if !sectionContains(rs, "candidate-real-divergence", "Foo::OnFlat") {
		t.Errorf("Foo::OnFlat must be candidate-real-divergence\n%s", rs)
	}
	// Foo::OnBranch: branches on an early read → per-mode-branch.
	if !sectionContains(rs, "per-mode-branch", "Foo::OnBranch") {
		t.Errorf("Foo::OnBranch must be per-mode-branch\n%s", rs)
	}
	// Foo::OnRepr: flat, width-tolerable rows → representation.
	if !sectionContains(rs, "representation", "Foo::OnRepr") {
		t.Errorf("Foo::OnRepr must be representation\n%s", rs)
	}
	// Foo::OnFail: decompile soft-fail → unverifiable.
	if !sectionContains(rs, "unverifiable", "Foo::OnFail") {
		t.Errorf("Foo::OnFail must be unverifiable\n%s", rs)
	}

	// A branching handler must NOT be cried as a real divergence.
	if sectionContains(rs, "candidate-real-divergence", "Foo::OnBranch") {
		t.Errorf("Foo::OnBranch must NOT be candidate-real-divergence (it branches)\n%s", rs)
	}

	// Foo::OnDispatch#HashMode: '#' FName with flat decompile and genuine mismatch
	// → must be per-mode-branch (the '#' pre-check fires BEFORE candidate-real logic).
	if !sectionContains(rs, "per-mode-branch", "Foo::OnDispatch#HashMode") {
		t.Errorf("Foo::OnDispatch#HashMode must be per-mode-branch (# entry)\n%s", rs)
	}
	if sectionContains(rs, "candidate-real-divergence", "Foo::OnDispatch#HashMode") {
		t.Errorf("Foo::OnDispatch#HashMode must NOT be candidate-real-divergence (# entry)\n%s", rs)
	}

	// Foo::OnRepeat: flat decompile (no branches) but faithful read-order is a
	// [D4,D2,D2]×3 repeating run → must be per-mode-branch (loop/array unrolled).
	if !sectionContains(rs, "per-mode-branch", "Foo::OnRepeat") {
		t.Errorf("Foo::OnRepeat must be per-mode-branch (repeating run)\n%s", rs)
	}
	if sectionContains(rs, "candidate-real-divergence", "Foo::OnRepeat") {
		t.Errorf("Foo::OnRepeat must NOT be candidate-real-divergence (repeating run)\n%s", rs)
	}

	// Foo::OnBranchDepth: audit JSON has BranchDepth=1 with an otherwise-flat
	// genuine mismatch. The BranchDepth pre-check is authoritative → per-mode-branch,
	// NOT candidate-real.
	if !sectionContains(rs, "per-mode-branch", "Foo::OnBranchDepth") {
		t.Errorf("Foo::OnBranchDepth must be per-mode-branch (audit BranchDepth=1)\n%s", rs)
	}
	if sectionContains(rs, "candidate-real-divergence", "Foo::OnBranchDepth") {
		t.Errorf("Foo::OnBranchDepth must NOT be candidate-real-divergence (BranchDepth=1)\n%s", rs)
	}

	// Foo::OnEmptyRead: the faithful read-order is empty (no Decode calls in
	// decompile) → extraction failure → unverifiable.
	if !sectionContains(rs, "unverifiable", "Foo::OnEmptyRead") {
		t.Errorf("Foo::OnEmptyRead must be unverifiable (empty read-order)\n%s", rs)
	}
	if sectionContains(rs, "candidate-real-divergence", "Foo::OnEmptyRead") {
		t.Errorf("Foo::OnEmptyRead must NOT be candidate-real-divergence (empty reads)\n%s", rs)
	}

	// Foo::OnUnresolved: the faithful read-order contains an Unresolved entry
	// (indirect dispatch passes packet alias) → cannot fully compare → unverifiable.
	if !sectionContains(rs, "unverifiable", "Foo::OnUnresolved") {
		t.Errorf("Foo::OnUnresolved must be unverifiable (Unresolved span)\n%s", rs)
	}
	if sectionContains(rs, "candidate-real-divergence", "Foo::OnUnresolved") {
		t.Errorf("Foo::OnUnresolved must NOT be candidate-real-divergence (Unresolved span)\n%s", rs)
	}

	// stdout roll-up: 1 candidate-real (Foo::OnFlat), 4 per-mode-branch
	// (Foo::OnBranch + Foo::OnDispatch#HashMode + Foo::OnRepeat + Foo::OnBranchDepth),
	// 1 representation, 3 unverifiable (Foo::OnFail + Foo::OnEmptyRead + Foo::OnUnresolved).
	if !strings.Contains(stdout.String(), "triage: candidate-real-divergence 1") {
		t.Errorf("stdout roll-up missing candidate count: %q", stdout.String())
	}
	for _, want := range []string{"per-mode-branch 4", "representation 1", "unverifiable 3"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout roll-up missing %q: %q", want, stdout.String())
		}
	}

	// Each worklist entry carries the FName, address, and client-read ops.
	if !strings.Contains(rs, "Foo::OnFlat") || !strings.Contains(rs, "@0x100") {
		t.Errorf("worklist entry must carry FName + @addr\n%s", rs)
	}

	// Read-only: baseline + every audit file byte-unchanged across a re-run.
	baselineBefore, _ := os.ReadFile("testdata/triage_baseline.json")
	auditBefore := snapshotDir(t, "testdata/triage_audit")
	_ = triageRun(triageOpts{
		Baseline:     "testdata/triage_baseline.json",
		AuditDir:     "testdata/triage_audit",
		Report:       report,
		DescentDepth: 4,
	}, fc, &bytes.Buffer{})
	baselineAfter, _ := os.ReadFile("testdata/triage_baseline.json")
	if !bytes.Equal(baselineBefore, baselineAfter) {
		t.Error("triage mutated the input baseline")
	}
	auditAfter := snapshotDir(t, "testdata/triage_audit")
	for name, before := range auditBefore {
		if !bytes.Equal(before, auditAfter[name]) {
			t.Errorf("triage mutated audit file %s", name)
		}
	}

	// Report is deterministic across runs.
	rb2, _ := os.ReadFile(report)
	if !bytes.Equal(rb, rb2) {
		t.Error("triage report is not deterministic")
	}
}

// snapshotDir reads every file in dir into a name->bytes map for read-only checks.
func snapshotDir(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		out[e.Name()] = b
	}
	return out
}
