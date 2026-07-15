package matrix

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// The three distinct StatePartial notes must render as distinct glyphs in
// STATUS.md, while the totals still count them all as a single partial (🟡)
// class — status.json is unaffected (render-only, FR-4.2 / task-169 T2.2).
func TestPartialNotesRenderDistinctly(t *testing.T) {
	m := Matrix{Rows: []MatrixRow{
		{Kind: RowOp, Op: "NEEDS_FIXTURE", Direction: opregistry.DirClientbound, Cells: map[string]Cell{
			"gms_v83": {State: StatePartial, Note: "tier-1: needs byte-fixture test to verify", Opcode: 0x001}}},
		{Kind: RowOp, Op: "DIFF_ONLY", Direction: opregistry.DirClientbound, Cells: map[string]Cell{
			"gms_v83": {State: StatePartial, Note: "tool ✅ without byte-test", Opcode: 0x002}}},
		{Kind: RowOp, Op: "PINNED", Direction: opregistry.DirClientbound, Cells: map[string]Cell{
			"gms_v83": {State: StatePartial, Note: "evidence-pinned deferral", Opcode: 0x003}}},
	}}
	got := RenderMarkdown(m, []string{"gms_v83"})

	glyphs := []string{"🟡ᶠ", "🟡ᵈ", "🟡ᵖ"}
	for _, g := range glyphs {
		// Each glyph must appear at least twice: once in the legend, once in a cell.
		if strings.Count(got, g) < 2 {
			t.Errorf("partial glyph %q should appear in both legend and a cell; count=%d", g, strings.Count(got, g))
		}
	}
	// The three glyphs must be distinct from each other.
	for i := range glyphs {
		for j := i + 1; j < len(glyphs); j++ {
			if glyphs[i] == glyphs[j] {
				t.Fatalf("partial glyphs not distinct: %q == %q", glyphs[i], glyphs[j])
			}
		}
	}

	// Totals unchanged: all three grade StatePartial, so the 🟡 column reads 3.
	var totalsLine string
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "| v83 |") {
			totalsLine = line
		}
	}
	if totalsLine == "" {
		t.Fatal("no v83 totals line")
	}
	// | v83 | ✅ | 🧩 | 🟡 | ❌ | ⬜ | 🟥 | % |  -> partial column is the 3rd count.
	cols := strings.Split(totalsLine, "|")
	// cols: ["", " v83 ", " ✅ ", " 🧩 ", " 🟡 ", ...]
	if got := strings.TrimSpace(cols[4]); got != "3" {
		t.Errorf("partial (🟡) total = %q; want 3 (totals must count all partials as one class)", got)
	}
}
