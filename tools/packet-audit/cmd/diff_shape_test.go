package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

func fc(ops ...idasrc.Primitive) []idasrc.FieldCall {
	out := make([]idasrc.FieldCall, len(ops))
	for i, o := range ops {
		out[i] = idasrc.FieldCall{Op: o}
	}
	return out
}

func TestClassifyDiff(t *testing.T) {
	cases := []struct {
		name       string
		hand, live []idasrc.FieldCall
		wantPos    string // "leading" | "trailing" | "interior" | "none"
		wantDelta  int    // len(live) - len(hand)
	}{
		{"leading-extra-live", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode1, idasrc.Decode4, idasrc.Decode2), "leading", 1},
		{"trailing-extra-live", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode4, idasrc.Decode2, idasrc.Decode1), "trailing", 1},
		{"interior", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode4, idasrc.Decode1, idasrc.Decode2), "interior", 1},
		{"identical", fc(idasrc.Decode4, idasrc.Decode2),
			fc(idasrc.Decode4, idasrc.Decode2), "none", 0},
	}
	for _, tc := range cases {
		d := classifyDiff(tc.hand, tc.live)
		if d.position != tc.wantPos || d.delta != tc.wantDelta {
			t.Errorf("%s: got {pos:%q delta:%d}, want {pos:%q delta:%d}",
				tc.name, d.position, d.delta, tc.wantPos, tc.wantDelta)
		}
	}
}

func TestDiffShapeRun_EmitsDivergentRows(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x100": fooDecomp}}
	dir := t.TempDir()
	report := filepath.Join(dir, "d.md")
	code := diffShapeRun(diffShapeOpts{Baseline: "testdata/diffshape_mini.json", Report: report, DescentDepth: 4}, fc, io.Discard)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := func() string { b, _ := os.ReadFile(report); return string(b) }()
	if !strings.Contains(s, "Foo::OnBar#Short") {
		t.Fatalf("divergent entry missing from diff-shape report:\n%s", s)
	}
	if !strings.Contains(s, "delta") {
		t.Fatalf("report missing delta annotation:\n%s", s)
	}
	if strings.Contains(s, "Foo::OnBar#A") {
		t.Fatalf("verified entry wrongly included:\n%s", s)
	}
}

func TestDiffShape_DeterministicAndReadOnly(t *testing.T) {
	fc := &validateFakeMCP{decomp: map[string]string{"0x100": fooDecomp}}
	dir := t.TempDir()
	r1 := filepath.Join(dir, "a.md")
	r2 := filepath.Join(dir, "b.md")
	before, _ := os.ReadFile("testdata/diffshape_mini.json")
	_ = diffShapeRun(diffShapeOpts{Baseline: "testdata/diffshape_mini.json", Report: r1, DescentDepth: 4}, fc, io.Discard)
	_ = diffShapeRun(diffShapeOpts{Baseline: "testdata/diffshape_mini.json", Report: r2, DescentDepth: 4}, fc, io.Discard)
	a, _ := os.ReadFile(r1)
	b, _ := os.ReadFile(r2)
	if string(a) != string(b) {
		t.Fatal("diff-shape report not deterministic")
	}
	after, _ := os.ReadFile("testdata/diffshape_mini.json")
	if string(before) != string(after) {
		t.Fatal("diff-shape mutated the baseline")
	}
}
