package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- discovery / parsing --------------------------------------------------

func TestParseDispatcherArms(t *testing.T) {
	dir := t.TempDir()
	runGo := filepath.Join(dir, "run.go")
	src := `package cmd
func candidatesFromFName(fname string) []candidate {
	switch fname {
	case "CFoo::OnPacket#Add":
		return []candidate{{name: "Add", pkg: "foo", dir: csvpkg.DirClientbound}}
	case "CFoo::OnPacket#Remove":
		return []candidate{{name: "Remove", pkg: "foo", dir: csvpkg.DirClientbound}}
	// serverbound arm: not a clientbound dispatcher arm, must be ignored
	case "CFoo::Send#Action":
		return []candidate{{name: "Action", pkg: "foo", dir: csvpkg.DirServerbound}}
	// login-style #-entry with NO pkg: must be ignored
	case "CLogin::X#Y":
		return []candidate{{name: "Y", dir: csvpkg.DirClientbound}}
	}
	return nil
}
`
	if err := os.WriteFile(runGo, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	arms, err := parseDispatcherArms(runGo)
	if err != nil {
		t.Fatal(err)
	}
	if len(arms) != 2 {
		t.Fatalf("expected 2 clientbound-with-pkg arms, got %d: %+v", len(arms), arms)
	}
	for _, a := range arms {
		if a.family != "CFoo::OnPacket" || a.pkg != "foo" {
			t.Errorf("unexpected arm: %+v", a)
		}
	}
}

func TestModePrefixDispatcherArmsRequiresMultipleArms(t *testing.T) {
	arms := []dispatcherArm{
		{family: "CMulti::On", mode: "A", name: "A", pkg: "p"},
		{family: "CMulti::On", mode: "B", name: "B", pkg: "p"},
		{family: "CSingle::On", mode: "X", name: "X", pkg: "p"}, // single arm -> dropped
	}
	got := modePrefixDispatcherArms(arms)
	if len(got) != 2 {
		t.Fatalf("expected 2 (only the multi-arm family), got %d: %+v", len(got), got)
	}
	for _, a := range got {
		if a.family != "CMulti::On" {
			t.Errorf("single-arm family leaked through: %+v", a)
		}
	}
}

// --- INV-1: shared-by-shape -----------------------------------------------

func TestCheckINV1SharedStruct(t *testing.T) {
	arms := []dispatcherArm{
		{family: "CFoo::On", mode: "First", name: "Shared", pkg: "foo"},
		{family: "CFoo::On", mode: "Second", name: "Shared", pkg: "foo"},
		{family: "CFoo::On", mode: "Distinct", name: "Distinct", pkg: "foo"},
	}
	vs, err := checkINV1(arms)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected 1 INV-1 violation, got %d: %+v", len(vs), vs)
	}
	if vs[0].inv != "INV-1" || !strings.Contains(vs[0].msg, "foo/Shared") {
		t.Errorf("unexpected violation: %+v", vs[0])
	}
}

func TestCheckINV1Clean(t *testing.T) {
	arms := []dispatcherArm{
		{family: "CFoo::On", mode: "A", name: "A", pkg: "foo"},
		{family: "CFoo::On", mode: "B", name: "B", pkg: "foo"},
	}
	vs, err := checkINV1(arms)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Fatalf("expected clean, got %+v", vs)
	}
}

// --- INV-2(a): mode literal in a constructor ------------------------------

func TestCheckINV2ModeLiteral(t *testing.T) {
	dir := t.TempDir()
	clean := filepath.Join(dir, "clean.go")
	dirty := filepath.Join(dir, "dirty.go")
	if err := os.WriteFile(clean, []byte("package clientbound\nfunc NewA(mode byte) A { return A{mode: mode} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dirty, []byte("package clientbound\nfunc NewB() B { return B{mode: 0x1E} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	structs := []resolvedStruct{
		{family: "CClean::On", pkg: "p", name: "A", file: clean},
		{family: "CDirty::On", pkg: "p", name: "B", file: dirty},
	}
	vs, err := checkINV2ModeLiteral(structs)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected 1 INV-2 violation, got %d: %+v", len(vs), vs)
	}
	if vs[0].inv != "INV-2" || !strings.Contains(vs[0].msg, "[family=CDirty::On]") {
		t.Errorf("unexpected violation: %+v", vs[0])
	}
}

// --- INV-2(b) + INV-3: body-function file checks --------------------------

func TestCheckBodyFunctionFiles(t *testing.T) {
	root := t.TempDir()
	packetLib := filepath.Join(root, "libs", "atlas-packet")

	// good family: func(mode byte) passthrough, no selector param.
	writeFile(t, filepath.Join(packetLib, "good", "operation_body.go"), `package good
func GoodAddBody(x byte) E {
	return WithResolvedCode("operations", KEY, func(mode byte) packet.Encoder {
		return NewGoodAdd(mode, x)
	})
}
`)
	// bad family: AP-3 func(_ byte) + AP-4 caller-specified code string.
	writeFile(t, filepath.Join(packetLib, "bad", "operation_body.go"), `package bad
func BadErrorBody(code string) E {
	return WithResolvedCode("operations", code, func(_ byte) packet.Encoder {
		return NewBadError()
	})
}
func BadError2Body(code string, target string) E { return nil }
`)

	arms := []dispatcherArm{
		{family: "CGood::On", mode: "Add", name: "Add", pkg: "good"},
		{family: "CGood::On", mode: "Two", name: "Two", pkg: "good"},
		{family: "CBad::On", mode: "Error", name: "Error", pkg: "bad"},
		{family: "CBad::On", mode: "Other", name: "Other", pkg: "bad"},
	}
	cfg := dispatcherLintConfig{PacketLib: packetLib}
	vs, err := checkBodyFunctionFiles(cfg, arms)
	if err != nil {
		t.Fatal(err)
	}
	var inv2, inv3 int
	for _, v := range vs {
		switch v.inv {
		case "INV-2":
			inv2++
			if !strings.Contains(v.msg, "[family=CBad::On]") {
				t.Errorf("INV-2 missing family tag: %+v", v)
			}
		case "INV-3":
			inv3++
		default:
			t.Errorf("unexpected violation: %+v", v)
		}
	}
	if inv2 != 1 {
		t.Errorf("expected 1 INV-2 (func(_ byte)), got %d", inv2)
	}
	// BadErrorBody(code string) and BadError2Body(code string, target string).
	if inv3 != 2 {
		t.Errorf("expected 2 INV-3 (caller-specified selector), got %d: %+v", inv3, vs)
	}
}

func TestSelectorParamRe(t *testing.T) {
	cases := []struct {
		params string
		match  bool
	}{
		{"code string", true},
		{"code string, name string", true},
		{"op string", true},
		{"mode *Mode", true},
		{"key string", true},
		{"position byte", false},
		{"name string", false},    // name is not a selector keyword
		{"message string", false}, // message is not a selector keyword
		{"partyId uint32", false}, // not a selector
		{"x byte, code string", true},
	}
	for _, c := range cases {
		if got := selectorParamRe.MatchString(c.params); got != c.match {
			t.Errorf("selectorParamRe(%q) = %v, want %v", c.params, got, c.match)
		}
	}
}

// TestCheckBodyFunctionFilesSemanticSelector verifies INV-3 catches a
// caller-specified operations key whose param name is NOT one of the
// op/code/mode/key keywords (the buddy `errorCode` escapee), while leaving a
// fixed-const key (and a `string(Const)` cast) clean.
func TestCheckBodyFunctionFilesSemanticSelector(t *testing.T) {
	root := t.TempDir()
	packetLib := filepath.Join(root, "libs", "atlas-packet")

	// buddy-style: selector named `errorCode` (escapes by-name matching) flows
	// into the operations key -> must be caught semantically.
	writeFile(t, filepath.Join(packetLib, "bud", "operation_body.go"), `package bud
func BudErrorBody(errorCode string) E {
	return WithResolvedCode("operations", errorCode, func(mode byte) packet.Encoder {
		return NewBudError(mode)
	})
}
func BudOkBody(slot byte) E {
	return WithResolvedCode("operations", BudOpFixed, func(mode byte) packet.Encoder {
		return NewBudOk(mode, slot)
	})
}
func BudCastBody(name string) E {
	return WithResolvedCode("operations", string(BudOpCast), func(mode byte) packet.Encoder {
		return NewBudCast(mode, name)
	})
}
`)

	arms := []dispatcherArm{
		{family: "CBud::On", mode: "Error", name: "Error", pkg: "bud"},
		{family: "CBud::On", mode: "Ok", name: "Ok", pkg: "bud"},
	}
	cfg := dispatcherLintConfig{PacketLib: packetLib}
	vs, err := checkBodyFunctionFiles(cfg, arms)
	if err != nil {
		t.Fatal(err)
	}
	var inv3 []violation
	for _, v := range vs {
		if v.inv == "INV-3" {
			inv3 = append(inv3, v)
		}
	}
	if len(inv3) != 1 {
		t.Fatalf("expected exactly 1 INV-3 (the errorCode selector), got %d: %+v", len(inv3), inv3)
	}
	if !strings.Contains(inv3[0].msg, "BudErrorBody") || !strings.Contains(inv3[0].msg, "errorCode") {
		t.Errorf("INV-3 should name BudErrorBody/errorCode; got: %s", inv3[0].msg)
	}
}

func TestParamNameSet(t *testing.T) {
	cases := []struct {
		params string
		want   []string
	}{
		{"errorCode string", []string{"errorCode"}},
		{"characterId uint32, slot int8, reason string", []string{"characterId", "slot", "reason"}},
		{"", nil},
		{"x byte", []string{"x"}},
	}
	for _, c := range cases {
		got := paramNameSet(c.params)
		for _, w := range c.want {
			if !got[w] {
				t.Errorf("paramNameSet(%q) missing %q; got %v", c.params, w, got)
			}
		}
		if len(c.want) == 0 && len(got) != 0 {
			t.Errorf("paramNameSet(%q) = %v, want empty", c.params, got)
		}
	}
}

// --- INV-4(a): dangling candidate -----------------------------------------

func TestCheckINV4Candidates(t *testing.T) {
	root := t.TempDir()
	packetLib := filepath.Join(root, "libs", "atlas-packet")
	writeFile(t, filepath.Join(packetLib, "foo", "clientbound", "add.go"), "package clientbound\ntype Add struct{}\n")

	arms := []dispatcherArm{
		{family: "CFoo::On", mode: "Add", name: "Add", pkg: "foo"},      // exists
		{family: "CFoo::On", mode: "Gone", name: "Missing", pkg: "foo"}, // dangling
	}
	cfg := dispatcherLintConfig{PacketLib: packetLib}
	vs, err := checkINV4Candidates(cfg, arms)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected 1 INV-4 dangling, got %d: %+v", len(vs), vs)
	}
	if vs[0].inv != "INV-4" || !strings.Contains(vs[0].msg, "Missing") {
		t.Errorf("unexpected violation: %+v", vs[0])
	}
}

// --- INV-4(b): phantom report ---------------------------------------------

func TestCheckINV4Reports(t *testing.T) {
	root := t.TempDir()
	audits := filepath.Join(root, "audits")

	// existing target file
	existing := filepath.Join(root, "libs", "atlas-packet", "foo", "clientbound", "add.go")
	writeFile(t, existing, "package clientbound\ntype Add struct{}\n")

	// JSON report citing an existing file: clean.
	writeFile(t, filepath.Join(audits, "ok.json"), `{"AtlasFile": "`+filepath.ToSlash(existing)+`"}`)
	// JSON report citing a deleted file: phantom.
	writeFile(t, filepath.Join(audits, "stale.json"), `{"AtlasFile": "libs/atlas-packet/foo/clientbound/gone.go"}`)
	// MD report citing a deleted file: phantom.
	writeFile(t, filepath.Join(audits, "stale.md"), "- **Atlas file:** `libs/atlas-packet/foo/clientbound/also_gone.go`\n")

	cfg := dispatcherLintConfig{AuditsDir: audits}
	vs, err := checkINV4Reports(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 {
		t.Fatalf("expected 2 phantom-report violations, got %d: %+v", len(vs), vs)
	}
	for _, v := range vs {
		if v.inv != "INV-4" {
			t.Errorf("unexpected inv: %+v", v)
		}
	}
}

// --- INV-5: orphaned codec ------------------------------------------------

func TestCheckINV5Orphans(t *testing.T) {
	root := t.TempDir()
	packetLib := filepath.Join(root, "libs", "atlas-packet")
	services := filepath.Join(root, "services")

	defFile := filepath.Join(packetLib, "foo", "clientbound", "ops.go")
	writeFile(t, defFile, `package clientbound
type Wrapped struct{ mode byte }
func NewWrapped(mode byte) Wrapped { return Wrapped{mode: mode} }
type Orphan struct{ mode byte }
func NewOrphan(mode byte) Orphan { return Orphan{mode: mode} }
type LiteralUsed struct{ Message string }
`)
	// body function wraps NewWrapped; a service constructs LiteralUsed via a
	// composite literal; Orphan is constructed nowhere.
	writeFile(t, filepath.Join(packetLib, "foo", "operation_body.go"), `package foo
func WrappedBody(x byte) E {
	return WithResolvedCode("operations", KEY, func(mode byte) packet.Encoder {
		return clientbound.NewWrapped(mode)
	})
}
`)
	writeFile(t, filepath.Join(services, "atlas-channel", "consumer.go"), `package channel
func use() { _ = &foo.LiteralUsed{Message: "hi"} }
`)

	structs := []resolvedStruct{
		{family: "CFoo::On", pkg: "foo", name: "Wrapped", file: defFile},
		{family: "CFoo::On", pkg: "foo", name: "Orphan", file: defFile},
		{family: "CFoo::On", pkg: "foo", name: "LiteralUsed", file: defFile},
	}
	cfg := dispatcherLintConfig{PacketLib: packetLib, UsageRoots: []string{packetLib, services}}
	vs, err := checkINV5Orphans(cfg, structs)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected exactly 1 orphan (Orphan), got %d: %+v", len(vs), vs)
	}
	if !strings.Contains(vs[0].msg, "Orphan") || vs[0].inv != "INV-5" {
		t.Errorf("unexpected violation: %+v", vs[0])
	}
}

// --- baseline filtering + end-to-end --------------------------------------

func TestLoadDispatcherBaseline(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "baseline.yaml")
	writeFile(t, p, "exempt_families:\n  - A::B\n  - C::D\n")
	set, err := loadDispatcherBaseline(p)
	if err != nil {
		t.Fatal(err)
	}
	if !set["A::B"] || !set["C::D"] || len(set) != 2 {
		t.Errorf("unexpected baseline set: %+v", set)
	}
}

func TestDispatcherLintRunEndToEnd(t *testing.T) {
	root := t.TempDir()
	packetLib := filepath.Join(root, "libs", "atlas-packet")
	audits := filepath.Join(root, "audits")
	runGo := filepath.Join(root, "run.go")
	baseline := filepath.Join(root, "baseline.yaml")

	// A footgun family "CBad::On" (multi-arm, body file with code-string
	// selector) and a clean family "CGood::On".
	writeFile(t, filepath.Join(packetLib, "good", "clientbound", "add.go"), "package clientbound\ntype GAdd struct{ mode byte }\nfunc NewGAdd(mode byte) GAdd { return GAdd{mode: mode} }\n")
	writeFile(t, filepath.Join(packetLib, "good", "clientbound", "rem.go"), "package clientbound\ntype GRem struct{ mode byte }\nfunc NewGRem(mode byte) GRem { return GRem{mode: mode} }\n")
	writeFile(t, filepath.Join(packetLib, "good", "operation_body.go"), `package good
func GoodAddBody() E { return WithResolvedCode("operations", K, func(mode byte) packet.Encoder { return clientbound.NewGAdd(mode) }) }
func GoodRemBody() E { return WithResolvedCode("operations", K, func(mode byte) packet.Encoder { return clientbound.NewGRem(mode) }) }
`)
	writeFile(t, filepath.Join(packetLib, "bad", "clientbound", "err.go"), "package clientbound\ntype BErr struct{ mode byte }\nfunc NewBErr(mode byte) BErr { return BErr{mode: mode} }\n")
	writeFile(t, filepath.Join(packetLib, "bad", "clientbound", "ok.go"), "package clientbound\ntype BOk struct{ mode byte }\nfunc NewBOk(mode byte) BOk { return BOk{mode: mode} }\n")
	writeFile(t, filepath.Join(packetLib, "bad", "operation_body.go"), `package bad
func BadErrorBody(code string) E { return WithResolvedCode("operations", code, func(mode byte) packet.Encoder { return clientbound.NewBErr(mode) }) }
func BadOkBody() E { return WithResolvedCode("operations", K, func(mode byte) packet.Encoder { return clientbound.NewBOk(mode) }) }
`)

	writeFile(t, runGo, `package cmd
func candidatesFromFName(fname string) []candidate {
	switch fname {
	case "CGood::On#Add":
		return []candidate{{name: "GAdd", pkg: "good", dir: csvpkg.DirClientbound}}
	case "CGood::On#Rem":
		return []candidate{{name: "GRem", pkg: "good", dir: csvpkg.DirClientbound}}
	case "CBad::On#Err":
		return []candidate{{name: "BErr", pkg: "bad", dir: csvpkg.DirClientbound}}
	case "CBad::On#Ok":
		return []candidate{{name: "BOk", pkg: "bad", dir: csvpkg.DirClientbound}}
	}
	return nil
}
`)

	cfg := dispatcherLintConfig{
		RunGo:        runGo,
		PacketLib:    packetLib,
		AuditsDir:    audits,
		BaselinePath: baseline,
		UsageRoots:   []string{packetLib},
	}
	_ = os.MkdirAll(audits, 0o755)

	// 1) No baseline -> CBad::On fires (INV-3).
	writeFile(t, baseline, "exempt_families: []\n")
	var out, errb bytes.Buffer
	code := dispatcherLintRun(cfg, &out, &errb)
	if code == 0 {
		t.Fatalf("expected non-zero exit with the footgun family un-baselined; out=%s", out.String())
	}
	if !strings.Contains(out.String(), "INV-3") || !strings.Contains(out.String(), "BadErrorBody") {
		t.Errorf("expected the BadErrorBody INV-3 violation; got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "CGood::On") {
		t.Errorf("clean family should not appear in violations; got:\n%s", out.String())
	}

	// 2) Baseline CBad::On -> clean, with a note line.
	writeFile(t, baseline, "exempt_families:\n  - CBad::On\n")
	out.Reset()
	errb.Reset()
	code = dispatcherLintRun(cfg, &out, &errb)
	if code != 0 {
		t.Fatalf("expected exit 0 with footgun family baselined; out=%s err=%s", out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "note\tCBad::On\tbaselined") {
		t.Errorf("expected a note line for the baselined family; got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "dispatcher-lint: clean") {
		t.Errorf("expected clean message; got:\n%s", out.String())
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
