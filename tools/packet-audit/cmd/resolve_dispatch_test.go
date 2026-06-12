package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

func TestResolveDispatch_AutoAcceptsHighConfidence(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.json")
	src, err := os.ReadFile("testdata/infer_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base, src, 0o644); err != nil {
		t.Fatal(err)
	}
	worklist := filepath.Join(dir, "worklist.md")

	// 0x100: switch on Decode1; case 1 -> Decode4 (#One), case 2 -> Decode2 (#Two).
	// Well-separated -> high confidence -> auto-accepted and written to the baseline.
	fc := &validateFakeMCP{decomp: map[string]string{"0x100": inferDecomp}}
	var out bytes.Buffer
	code := resolveDispatchRun(resolveDispatchOpts{Baseline: base, Worklist: worklist, MinConfidence: 0.6, DescentDepth: 4}, fc, &out)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}

	s, err := idasrc.NewExportSource(base)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]int64{}
	for _, e := range s.Entries() {
		if len(e.Dispatch) == 1 {
			cases[e.FName] = e.Dispatch[0].Case
		}
	}
	if cases["Foo::OnBar#One"] != 1 || cases["Foo::OnBar#Two"] != 2 {
		t.Fatalf("selectors not persisted as expected: %+v", cases)
	}
	if !strings.Contains(out.String(), "auto-accepted") {
		t.Fatalf("roll-up missing 'auto-accepted': %q", out.String())
	}
	// A worklist file is always written (even if empty).
	if _, err := os.Stat(worklist); err != nil {
		t.Fatalf("worklist not written: %v", err)
	}
}
