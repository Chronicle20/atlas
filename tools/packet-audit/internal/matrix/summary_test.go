package matrix

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

func summaryFixture() Matrix {
	return Matrix{Rows: []MatrixRow{
		{Kind: RowOp, Op: "VERIFIED_OP", Direction: opregistry.DirClientbound, Cells: map[string]Cell{
			"gms_v83": {State: StateVerified, Opcode: 0x001}}},
		{Kind: RowOp, Op: "OPEN_OP", Direction: opregistry.DirClientbound, Cells: map[string]Cell{
			"gms_v83": {State: StateIncomplete, Note: "no audit report", Opcode: 0x002}}},
		{Kind: RowOp, Op: "STALE_OP", Direction: opregistry.DirClientbound, Cells: map[string]Cell{
			"gms_v83": {State: StateIncomplete, Note: "evidence stale (decompile hash drift)", Opcode: 0x003}}},
		{Kind: RowOp, Op: "PARTIAL_OP", Direction: opregistry.DirServerbound, Cells: map[string]Cell{
			"gms_v83": {State: StatePartial, Note: "tool ✅ without byte-test", Opcode: 0x004}}},
		{Kind: RowOp, Op: "CONFLICT_OP", Direction: opregistry.DirClientbound, Cells: map[string]Cell{
			"gms_v83": {State: StateConflict, Note: "template-wiring gap", Opcode: 0x005}}},
		{Kind: RowSubStruct, Packet: "npc/clientbound/Detail", Cells: map[string]Cell{
			"gms_v83": {State: StateNA, Note: dispositionNote, Opcode: -1}}},
	}}
}

func TestSummarizeBuckets(t *testing.T) {
	s := Summarize(summaryFixture(), "gms_v83")
	if s.Verified != 1 || s.Incomplete != 2 || s.Partial != 1 || s.Conflict != 1 || s.NACount != 1 {
		t.Fatalf("counts wrong: %+v", s)
	}
	// Total (non-n-a) = verified+partial+incomplete+conflict = 1+1+2+1 = 5.
	if s.Total != 5 {
		t.Fatalf("total = %d; want 5", s.Total)
	}
	if s.VerifiedPct < 19.9 || s.VerifiedPct > 20.1 {
		t.Fatalf("verified%% = %.2f; want 20.0", s.VerifiedPct)
	}
	// Bucket sizes: unverified = incomplete+partial+family = 3; conflicts=1; na=1.
	if len(s.Unverified) != 3 || len(s.Conflicts) != 1 || len(s.NA) != 1 {
		t.Fatalf("bucket sizes: unverified=%d conflicts=%d na=%d", len(s.Unverified), len(s.Conflicts), len(s.NA))
	}
	// Stale = the one incomplete cell whose note signals drift.
	if len(s.Stale) != 1 || s.Stale[0].Op != "STALE_OP" {
		t.Fatalf("stale detection wrong: %+v", s.Stale)
	}
}

func TestRenderSupportMarkdownDeterministic(t *testing.T) {
	a := RenderSupportMarkdown(Summarize(summaryFixture(), "gms_v83"))
	b := RenderSupportMarkdown(Summarize(summaryFixture(), "gms_v83"))
	if a != b {
		t.Fatal("support summary render not deterministic")
	}
	if !strings.Contains(a, "n-a (deliberate)") || !strings.Contains(a, "Unverified (open gaps)") || !strings.Contains(a, "Conflicts (🟥)") {
		t.Fatalf("missing a gap section:\n%s", a)
	}
	// No wall-clock date.
	if strings.Contains(a, "Z") && strings.Contains(a, "T") && strings.Contains(a, ":") {
		for _, line := range strings.Split(a, "\n") {
			if strings.Contains(line, "T") && strings.Contains(line, "Z") && strings.Contains(line, ":") {
				t.Errorf("timestamp-looking line: %q", line)
			}
		}
	}
}
