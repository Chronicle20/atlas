package cmd

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

// opRow is a small test helper building a RowOp matrix.MatrixRow with the
// given op name and per-version states.
func opRow(op string, cells map[string]matrix.State) matrix.MatrixRow {
	c := make(map[string]matrix.Cell, len(cells))
	for vk, st := range cells {
		c[vk] = matrix.Cell{State: st}
	}
	return matrix.MatrixRow{
		Kind:      matrix.RowOp,
		Op:        op,
		Direction: opregistry.DirServerbound,
		Cells:     c,
	}
}

var teleportRockFamily = featureFamiliesDoc{
	Families: map[string][]string{
		"teleport_rock": {"USE_TELEPORT_ROCK", "TROCK_ADD_MAP", "MAP_TRANSFER_RESULT"},
	},
}

// (a) One member verified, another n-a, WITHOUT a matching evidence entry —
// must produce a problem.
func TestNAConsistency_InconsistentWithoutEvidence(t *testing.T) {
	m := matrix.Matrix{Rows: []matrix.MatrixRow{
		opRow("USE_TELEPORT_ROCK", map[string]matrix.State{"gms_v48": matrix.StateNA, "gms_v61": matrix.StateVerified}),
		opRow("TROCK_ADD_MAP", map[string]matrix.State{"gms_v48": matrix.StateVerified, "gms_v61": matrix.StateVerified}),
		opRow("MAP_TRANSFER_RESULT", map[string]matrix.State{"gms_v48": matrix.StateVerified, "gms_v61": matrix.StateVerified}),
	}}
	res := naConsistencyCheck(m, teleportRockFamily, naEvidenceDoc{}, []string{"gms_v48", "gms_v61"})
	if len(res.Problems) != 1 {
		t.Fatalf("want 1 problem, got %d: %v", len(res.Problems), res.Problems)
	}
	if !strings.Contains(res.Problems[0], "USE_TELEPORT_ROCK") || !strings.Contains(res.Problems[0], "gms_v48") {
		t.Errorf("problem does not name the offending cell: %q", res.Problems[0])
	}
	if !strings.Contains(res.Problems[0], "add positive absence evidence") {
		t.Errorf("problem missing remediation hint: %q", res.Problems[0])
	}
	if len(res.Notes) != 0 {
		t.Errorf("want no consumed-evidence notes, got %v", res.Notes)
	}
}

// (b) Same layout WITH a matching evidence entry — no problem, and the entry
// is reported as consumed.
func TestNAConsistency_InconsistentWithEvidence(t *testing.T) {
	m := matrix.Matrix{Rows: []matrix.MatrixRow{
		opRow("USE_TELEPORT_ROCK", map[string]matrix.State{"gms_v48": matrix.StateNA, "gms_v61": matrix.StateVerified}),
		opRow("TROCK_ADD_MAP", map[string]matrix.State{"gms_v48": matrix.StateVerified, "gms_v61": matrix.StateVerified}),
		opRow("MAP_TRANSFER_RESULT", map[string]matrix.State{"gms_v48": matrix.StateVerified, "gms_v61": matrix.StateVerified}),
	}}
	ev := naEvidenceDoc{Entries: []naEvidenceEntry{
		{Op: "USE_TELEPORT_ROCK", Version: "gms_v48", Evidence: "binary-wide search found zero hits; see task-124"},
	}}
	res := naConsistencyCheck(m, teleportRockFamily, ev, []string{"gms_v48", "gms_v61"})
	if len(res.Problems) != 0 {
		t.Fatalf("want 0 problems, got %d: %v", len(res.Problems), res.Problems)
	}
	if len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "USE_TELEPORT_ROCK") || !strings.Contains(res.Notes[0], "gms_v48") {
		t.Errorf("want one consumed note naming USE_TELEPORT_ROCK x gms_v48, got %v", res.Notes)
	}
}

// An evidence entry with empty evidence text does not count as proof — it
// must still produce a problem.
func TestNAConsistency_EmptyEvidenceTextStillFails(t *testing.T) {
	m := matrix.Matrix{Rows: []matrix.MatrixRow{
		opRow("USE_TELEPORT_ROCK", map[string]matrix.State{"gms_v48": matrix.StateNA}),
		opRow("TROCK_ADD_MAP", map[string]matrix.State{"gms_v48": matrix.StateVerified}),
	}}
	ev := naEvidenceDoc{Entries: []naEvidenceEntry{
		{Op: "USE_TELEPORT_ROCK", Version: "gms_v48", Evidence: "   "},
	}}
	res := naConsistencyCheck(m, teleportRockFamily, ev, []string{"gms_v48"})
	if len(res.Problems) != 1 {
		t.Fatalf("want 1 problem (empty evidence text is not proof), got %d: %v", len(res.Problems), res.Problems)
	}
}

// (c) A family where every member is verified on a version — no problem.
func TestNAConsistency_AllVerified(t *testing.T) {
	m := matrix.Matrix{Rows: []matrix.MatrixRow{
		opRow("USE_TELEPORT_ROCK", map[string]matrix.State{"gms_v61": matrix.StateVerified}),
		opRow("TROCK_ADD_MAP", map[string]matrix.State{"gms_v61": matrix.StateVerified}),
		opRow("MAP_TRANSFER_RESULT", map[string]matrix.State{"gms_v61": matrix.StateVerified}),
	}}
	res := naConsistencyCheck(m, teleportRockFamily, naEvidenceDoc{}, []string{"gms_v61"})
	if len(res.Problems) != 0 {
		t.Fatalf("want 0 problems (all verified), got %d: %v", len(res.Problems), res.Problems)
	}
	if len(res.Notes) != 0 {
		t.Errorf("want no notes (nothing n-a to record evidence for), got %v", res.Notes)
	}
}

// (d) A family with an n-a member but NO verified sibling on that version —
// legitimate absence (the feature isn't present at all on that version), so
// no problem and nothing to record.
func TestNAConsistency_AllAbsentIsLegitimate(t *testing.T) {
	m := matrix.Matrix{Rows: []matrix.MatrixRow{
		opRow("USE_TELEPORT_ROCK", map[string]matrix.State{"gms_v48": matrix.StateNA}),
		opRow("TROCK_ADD_MAP", map[string]matrix.State{"gms_v48": matrix.StateNA}),
		opRow("MAP_TRANSFER_RESULT", map[string]matrix.State{"gms_v48": matrix.StateIncomplete}),
	}}
	res := naConsistencyCheck(m, teleportRockFamily, naEvidenceDoc{}, []string{"gms_v48"})
	if len(res.Problems) != 0 {
		t.Fatalf("want 0 problems (no verified sibling to prove inconsistency), got %d: %v", len(res.Problems), res.Problems)
	}
	if len(res.Notes) != 0 {
		t.Errorf("want no notes, got %v", res.Notes)
	}
}

// A version not carrying the family at all (no cells) must not panic and
// must not produce a problem.
func TestNAConsistency_MissingVersionCellIsSkipped(t *testing.T) {
	m := matrix.Matrix{Rows: []matrix.MatrixRow{
		opRow("USE_TELEPORT_ROCK", map[string]matrix.State{"gms_v61": matrix.StateVerified}),
		opRow("TROCK_ADD_MAP", map[string]matrix.State{"gms_v61": matrix.StateVerified}),
	}}
	res := naConsistencyCheck(m, teleportRockFamily, naEvidenceDoc{}, []string{"gms_v61", "jms_v185"})
	if len(res.Problems) != 0 {
		t.Fatalf("want 0 problems, got %d: %v", len(res.Problems), res.Problems)
	}
}

func TestLoadFeatureFamiliesMissingFile(t *testing.T) {
	d, err := loadFeatureFamilies("testdata/nonexistent-feature-families.yaml")
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if len(d.Families) != 0 {
		t.Errorf("want empty families, got %v", d.Families)
	}
}

func TestLoadNAEvidenceMissingFile(t *testing.T) {
	d, err := loadNAEvidence("testdata/nonexistent-na-evidence.yaml")
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if len(d.Entries) != 0 {
		t.Errorf("want empty entries, got %v", d.Entries)
	}
}
