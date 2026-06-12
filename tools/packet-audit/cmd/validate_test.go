package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// validateFakeMCP is a local MCPClient fake for the validate command tests
// (mirrors export_test.go's fakeMCP, but adds per-address decompile errors so a
// base decompilation soft-fail can be exercised). It lives here, not in
// export_test.go, so the two tests stay independent.
type validateFakeMCP struct {
	decomp    map[string]string
	decompErr map[string]error
}

func (f *validateFakeMCP) GetFunctionByName(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}
func (f *validateFakeMCP) DecompileFunction(_ context.Context, a string) (string, error) {
	if err := f.decompErr[a]; err != nil {
		return "", err
	}
	return f.decomp[a], nil
}
func (f *validateFakeMCP) GetCallees(_ context.Context, _ string) ([]idasrc.Callee, error) {
	return nil, nil
}
func (f *validateFakeMCP) StructInfo(_ context.Context, _ string) (idasrc.StructLayout, error) {
	return idasrc.StructLayout{}, nil
}

// base decompile for 0x100: switch on Decode1; case 1 reads Decode4 (matches #A -> verified);
// case 2 reads Decode2 (hand #B says Decode4 -> divergent at [1]).
const fooDecomp = "void __thiscall Foo::OnBar(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  switch ( CInPacket::Decode1(a2) )\n" +
	"  {\n" +
	"    case 1:\n      CInPacket::Decode4(a2);\n      break;\n" +
	"    case 2:\n      CInPacket::Decode2(a2);\n      break;\n" +
	"  }\n}\n"

func TestValidateRunReport(t *testing.T) {
	fc := &validateFakeMCP{
		decomp:    map[string]string{"0x100": fooDecomp},
		decompErr: map[string]error{"0xDEAD": idasrc.NewRPCDecompileError("0xDEAD")},
	}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/validate_mini.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	b, _ := os.ReadFile(report)
	s := string(b)
	// #A verified, #B divergent (Decode4 vs Decode2), Baz unverifiable (decompile failed).
	// sectionOf ties each FName to its verdict section so a mislabel is caught.
	if got := sectionOf(s, "Foo::OnBar#A"); got != "verified" {
		t.Errorf("Foo::OnBar#A in section %q, want verified", got)
	}
	if got := sectionOf(s, "Foo::OnBar#B"); got != "divergent" {
		t.Errorf("Foo::OnBar#B in section %q, want divergent", got)
	}
	if got := sectionOf(s, "Baz::OnQux"); got != "unverifiable" {
		t.Errorf("Baz::OnQux in section %q, want unverifiable", got)
	}
	// baseline file must be UNCHANGED (read-only):
	before, _ := os.ReadFile("testdata/validate_mini.json")
	_ = validateRun(validateOpts{Baseline: "testdata/validate_mini.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	after, _ := os.ReadFile("testdata/validate_mini.json")
	if !bytes.Equal(before, after) {
		t.Error("validate mutated the baseline")
	}
}

// authDecomp is the base decompile for 0x200 (Auth::OnCheckPasswordResult):
// switch on Decode1; case 0 reads Decode4 (matches #AuthSuccess -> verified);
// case 1 reads Decode2+Decode4. #AuthLoginFailed has NO dispatch selector, so
// ExtractShape returns the whole function (4 reads). Without the fix it is a
// false divergent (hand 2 vs live 4); with the fix it is unverifiable.
const authDecomp = "void __thiscall Auth::OnCheckPasswordResult(Auth *this, CInPacket *a2)\n" +
	"{\n" +
	"  switch ( CInPacket::Decode1(a2) )\n" +
	"  {\n" +
	"    case 0:\n      CInPacket::Decode4(a2);\n      break;\n" +
	"    case 1:\n      CInPacket::Decode2(a2);\n      CInPacket::Decode4(a2);\n      break;\n" +
	"  }\n}\n"

// TestValidateRunUndispatchable verifies that a per-mode entry (#) with NO
// dispatch selector is classified as unverifiable (not a false divergent), while
// a correctly-dispatched # entry still gets its real verdict (verified).
func TestValidateRunUndispatchable(t *testing.T) {
	fc := &validateFakeMCP{
		decomp:    map[string]string{"0x200": authDecomp},
		decompErr: map[string]error{},
	}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{
		Baseline:     "testdata/validate_undispatchable.json",
		Report:       report,
		DescentDepth: 4,
	}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	b, _ := os.ReadFile(report)
	s := string(b)
	// #AuthLoginFailed has no dispatch — must be unverifiable (not a false divergent).
	if got := sectionOf(s, "Auth::OnCheckPasswordResult#AuthLoginFailed"); got != "unverifiable" {
		t.Errorf("Auth::OnCheckPasswordResult#AuthLoginFailed in section %q, want unverifiable", got)
	}
	// #AuthSuccess has dispatch case 0 that extracts cleanly — must stay verified.
	if got := sectionOf(s, "Auth::OnCheckPasswordResult#AuthSuccess"); got != "verified" {
		t.Errorf("Auth::OnCheckPasswordResult#AuthSuccess in section %q, want verified", got)
	}
}

// sectionOf returns the verdict section header under which fname appears in the
// report, or "" if absent. It scans lines sequentially, tracking the current
// "## <verdict>" section and returning the section name on the first line that
// contains fname.
func sectionOf(report, fname string) string {
	section := ""
	for _, line := range strings.Split(report, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") {
			section = strings.TrimSpace(strings.TrimPrefix(t, "##"))
			continue
		}
		if strings.Contains(line, fname) {
			return section
		}
	}
	return ""
}

// bijDecomp: NAMED discriminator `mode` (so CaseLabels is keyed "mode", matching
// the selectors). Client switch has cases 1 and 2. Baseline binds #One->case1 and
// #Ghost->case7. So case 2 is missing-mode and #Ghost(case7) is extra-mode.
const bijDecomp = "void __thiscall Foo::OnBar(Foo *this, CInPacket *a2)\n" +
	"{\n" +
	"  unsigned __int8 mode = CInPacket::Decode1(a2);\n" +
	"  switch ( mode )\n" +
	"  {\n" +
	"    case 1:\n      CInPacket::Decode4(a2);\n      break;\n" +
	"    case 2:\n      CInPacket::Decode2(a2);\n      break;\n" +
	"  }\n}\n"

func TestValidate_BijectionMissingExtra(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x200": bijDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/bijection_mini.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	b, _ := os.ReadFile(report)
	s := string(b)
	if got := sectionOf(s, "Foo::OnBar#case<2>"); got != "missing-mode" {
		t.Errorf("client case 2 in section %q, want missing-mode\n%s", got, s)
	}
	if got := sectionOf(s, "Foo::OnBar#Ghost"); got != "extra-mode" {
		t.Errorf("Foo::OnBar#Ghost in section %q, want extra-mode\n%s", got, s)
	}
}

func TestValidate_AllowlistSuppressesMissing(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x200": bijDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	allow := filepath.Join(dir, "_unimplemented.json")
	// Suppress the otherwise-missing client case 2.
	if err := os.WriteFile(allow, []byte(`{"entries":[{"fname":"Foo::OnBar","case":2,"reason":"not built"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code := validateRun(validateOpts{Baseline: "testdata/bijection_mini.json", Report: report, Allowlist: allow, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	b, _ := os.ReadFile(report)
	s := string(b)
	if got := sectionOf(s, "Foo::OnBar#case<2>"); got != "allowlisted" {
		t.Errorf("case 2 in section %q, want allowlisted\n%s", got, s)
	}
}

// Two #Mode entries of the same base handler at DIFFERENT addresses; both
// decompiles expose the same switch (cases 1,2). Case 2 is bound at 0x201, so it
// must NOT be reported missing at 0x200 (per-handler grouping, no false-missing /
// no duplication).
func TestValidate_BijectionMultiAddressNoFalseMissing(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x200": bijDecomp, "0x201": bijDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	var dbg bytes.Buffer
	code := validateRun(validateOpts{Baseline: "testdata/bijection_multiaddr.json", Report: report, DescentDepth: 4}, fc, &dbg)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, dbg.String())
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	// Both cases bound (1@0x200, 2@0x201) -> NO missing-mode at all.
	if strings.Contains(s, "#case<") {
		t.Fatalf("expected no missing-mode (both cases bound across addresses):\n%s", s)
	}
}

// leafDecomp: a linear leaf handler (no dispatch) — Decode4 then Decode2.
const leafDecomp = "void __thiscall Foo::OnLeaf(Foo *this, CInPacket *a2)\n{\n" +
	"  CInPacket::Decode4(a2);\n  CInPacket::Decode2(a2);\n}\n"

func TestValidate_LeafModeFlatValidated(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x300": leafDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/leaf_mode.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	if got := sectionOf(s, "Foo::OnLeaf#Solo"); got != "verified" {
		t.Fatalf("Foo::OnLeaf#Solo in %q, want verified\n%s", got, s)
	}
}

func TestValidate_MultiwayModeStaysUnverifiable(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x200": authDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/multiway_nosel.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	if got := sectionOf(s, "Auth::OnCheckPasswordResult#NoSel"); got != "unverifiable" {
		t.Fatalf("multiway no-selector entry in %q, want unverifiable\n%s", got, s)
	}
}

// A leaf #Mode entry whose decompile yields ZERO reads must be unverifiable
// (extraction failed) — not a false hand-N-vs-live-0 divergence.
func TestValidate_LeafEmptyLiveIsUnverifiable(t *testing.T) {
	emptyDecomp := "void __thiscall Foo::OnLeaf(Foo *this, CInPacket *a2)\n{\n  return;\n}\n"
	fc := &validateFakeMCP{decomp: map[string]string{"0x300": emptyDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/leaf_mode.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	if got := sectionOf(s, "Foo::OnLeaf#Solo"); got != "unverifiable" {
		t.Fatalf("empty-live leaf in %q, want unverifiable\n%s", got, s)
	}
}

// A verbatim {Guard} selector must NOT be a bijection binding (Case==0 would
// otherwise show as a false extra-mode "case 0 absent from client").
func TestValidate_VerbatimSelectorNotBijectionBinding(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x200": bijDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "r.md")
	code := validateRun(validateOpts{Baseline: "testdata/verbatim_nobij.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	if strings.Contains(s, "extra-mode") && strings.Contains(s, "#case<0>") {
		t.Fatalf("verbatim selector produced a false case<0> extra-mode:\n%s", s)
	}
	if got := sectionOf(s, "Foo::OnBar#Verb"); got == "extra-mode" {
		t.Fatalf("verbatim #Verb wrongly flagged extra-mode\n%s", s)
	}
}
