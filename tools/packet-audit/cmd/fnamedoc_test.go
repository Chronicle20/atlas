package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyCommentInsertsAboveStruct(t *testing.T) {
	src := "package x\n\ntype Foo struct {\n\ta byte\n}\n"
	out, state := applyComment(src, "Foo", "// packet-audit:fname C::OnFoo")
	if state != commentMissing {
		t.Fatalf("state = %v, want missing", state)
	}
	want := "package x\n\n// packet-audit:fname C::OnFoo\ntype Foo struct {\n\ta byte\n}\n"
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestApplyCommentDriftRewrites(t *testing.T) {
	src := "// packet-audit:fname C::Old\ntype Foo struct {\n}\n"
	out, state := applyComment(src, "Foo", "// packet-audit:fname C::New")
	if state != commentDrift {
		t.Fatalf("state = %v, want drift", state)
	}
	if out != "// packet-audit:fname C::New\ntype Foo struct {\n}\n" {
		t.Errorf("drift not rewritten: %q", out)
	}
}

func TestApplyCommentOKWhenCurrent(t *testing.T) {
	src := "// packet-audit:fname C::Foo\ntype Foo struct {\n}\n"
	_, state := applyComment(src, "Foo", "// packet-audit:fname C::Foo")
	if state != commentOK {
		t.Errorf("state = %v, want ok", state)
	}
}

func TestApplyCommentListBlockGetsSeparator(t *testing.T) {
	// A preceding doc block containing a list item requires a blank `//`
	// separator before the appended fname line to stay gofmt-clean.
	src := "// Foo does things:\n//   - one\n//   - two\ntype Foo struct {\n}\n"
	out, _ := applyComment(src, "Foo", "// packet-audit:fname C::Foo")
	want := "// Foo does things:\n//   - one\n//   - two\n//\n// packet-audit:fname C::Foo\ntype Foo struct {\n}\n"
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestApplyCommentPlainDocNoSeparator(t *testing.T) {
	src := "// Foo is a packet.\ntype Foo struct {\n}\n"
	out, _ := applyComment(src, "Foo", "// packet-audit:fname C::Foo")
	want := "// Foo is a packet.\n// packet-audit:fname C::Foo\ntype Foo struct {\n}\n"
	if out != want {
		t.Errorf("got:\n%q\nwant:\n%q", out, want)
	}
}

func TestCodecStructsFindsOperationReceivers(t *testing.T) {
	src := `package x
type Foo struct{}
func (f Foo) Operation() string { return "X" }
type Bar struct{}
func (b *Bar) Operation() string { return "Y" }
type NotACodec struct{}
`
	got := codecStructs(src)
	if len(got) != 2 {
		t.Fatalf("got %v, want [Foo Bar]", got)
	}
}

func TestReportHasUnresolvedRow(t *testing.T) {
	tests := []struct {
		name string
		rows []struct{ Verdict int }
		want bool
	}{
		{"no rows", nil, false},
		{"all resolved states", []struct{ Verdict int }{{0}, {1}, {2}, {3}}, false},
		{"one unresolved row (VerdictUnresolved=4)", []struct{ Verdict int }{{0}, {4}, {2}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reportHasUnresolvedRow(tt.rows); got != tt.want {
				t.Errorf("reportHasUnresolvedRow(%v) = %v, want %v", tt.rows, got, tt.want)
			}
		})
	}
}

// TestLoadReportFNamesSkipsUnresolvedReport locks in the task-28 fix: a
// report whose IDA side never actually resolved (diff.VerdictUnresolved,
// numeric 4) must not win WriterName->IDAName priority, even when it is the
// ONLY report for that writer. Modeled directly on the real case that
// motivated the guard: gms_v92/CashItemUseVegaScroll.json is the sole
// report for that writer and carries a Verdict:4 row; without the skip it
// would silently overwrite the hand-verified, arm-suffixed
// "...SendConsumeCashItemUseRequest#VegaScroll" doc comment with a weaker,
// unsuffixed fname. A refactor that drops the skip should fail this test.
func TestLoadReportFNamesSkipsUnresolvedReport(t *testing.T) {
	dir := t.TempDir()
	writeRawAuditReport(t, dir, "gms_v92", "CashItemUseVegaScroll.json", `{
		"WriterName": "CashItemUseVegaScroll",
		"IDAName": "CWvsContext::SendConsumeCashItemUseRequest",
		"Rows": [{"Verdict": 4}, {"Verdict": 2}]
	}`)
	// A normal, fully-resolved report for a different writer must still
	// resolve normally — the skip must not be over-broad.
	writeRawAuditReport(t, dir, "gms_v95", "Ping.json", `{
		"WriterName": "Ping",
		"IDAName": "CClientSocket::OnAliveReq",
		"Rows": [{"Verdict": 0}]
	}`)

	out, err := loadReportFNames(dir)
	if err != nil {
		t.Fatalf("loadReportFNames: %v", err)
	}
	if got, ok := out.fname["CashItemUseVegaScroll"]; ok {
		t.Errorf("CashItemUseVegaScroll should have been skipped (unresolved row), but resolved to %q", got)
	}
	if got, want := out.fname["Ping"], "CClientSocket::OnAliveReq"; got != want {
		t.Errorf("Ping fname = %q, want %q", got, want)
	}
}

// task-226: two codecs that share a family AND a struct name across the two
// directions ("character"+"SkillMacro") collapse to one WriterName key. Each
// direction must still resolve to its OWN report's fname, disambiguated by the
// reports' AtlasFile — otherwise whichever report loads first wins and the
// other file reads as permanent DRIFT. A helper struct sharing the file must
// stay unresolved rather than inheriting the file's fname.
func TestResolveFNamePrefersOwnFileOnCrossDirectionCollision(t *testing.T) {
	dir := t.TempDir()
	const cb = "libs/atlas-packet/character/clientbound/skill_macro.go"
	const sb = "libs/atlas-packet/character/serverbound/skill_macro.go"
	writeRawAuditReport(t, dir, "gms_v95", "CharacterSkillMacro.json", `{
		"WriterName": "CharacterSkillMacro",
		"IDAName": "CWvsContext::OnMacroSysDataInit",
		"AtlasFile": "`+cb+`",
		"Rows": [{"Verdict": 0}]
	}`)
	writeRawAuditReport(t, dir, "gms_v95", "CharacterSkillMacroHandle.json", `{
		"WriterName": "CharacterSkillMacroHandle",
		"IDAName": "CMacroSysMan::FlushToSvr",
		"AtlasFile": "`+sb+`",
		"Rows": [{"Verdict": 0}]
	}`)

	out, err := loadReportFNames(dir)
	if err != nil {
		t.Fatalf("loadReportFNames: %v", err)
	}
	if got, ok := out.resolveFName("CharacterSkillMacro", cb); !ok || got != "CWvsContext::OnMacroSysDataInit" {
		t.Errorf("clientbound SkillMacro = %q (ok=%v), want CWvsContext::OnMacroSysDataInit", got, ok)
	}
	if got, ok := out.resolveFName("CharacterSkillMacro", sb); !ok || got != "CMacroSysMan::FlushToSvr" {
		t.Errorf("serverbound SkillMacro = %q (ok=%v), want CMacroSysMan::FlushToSvr", got, ok)
	}
	if got, ok := out.resolveFName("CharacterSkillMacroEntry", sb); ok {
		t.Errorf("helper struct sharing the file must stay unresolved, got %q", got)
	}
}

func writeRawAuditReport(t *testing.T, auditsDir, version, filename, content string) {
	t.Helper()
	vdir := filepath.Join(auditsDir, version)
	if err := os.MkdirAll(vdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vdir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
