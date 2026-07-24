package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
)

// Feature-family n-a consistency gate (task-124): "the receive side proves
// the send side", and absence needs positive proof, not a failed search. A
// cell can't be graded `n-a` ("op absent") on a version when a same-feature
// sibling op is `verified` on that same version, unless the n-a member
// carries a recorded positive-absence entry in
// docs/packets/feature-na-evidence.yaml. Check-only (matrix --check): it
// never changes grading or the rendered STATUS.md/status.json.
//
// See docs/packets/audits/VERIFYING_A_PACKET.md "Is this cell n-a? (proving
// absence)" for the authoring bar an evidence entry must clear.

// featureFamiliesDoc is the parsed shape of docs/packets/feature-families.yaml:
// named groups of ops that together form one logical feature (presence is
// correlated — verifying one is send/receive-side proof the feature exists).
type featureFamiliesDoc struct {
	Families map[string][]string `yaml:"families"`
}

const (
	defaultFeatureFamiliesPath   = "docs/packets/feature-families.yaml"
	defaultFeatureNAEvidencePath = "docs/packets/feature-na-evidence.yaml"
)

// loadFeatureFamilies reads and parses feature-families.yaml at path. A
// missing file is not an error — it returns an empty doc (no family is
// checked), so the matrix still runs in trees that predate this gate.
func loadFeatureFamilies(path string) (featureFamiliesDoc, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return featureFamiliesDoc{}, nil
	}
	if err != nil {
		return featureFamiliesDoc{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var d featureFamiliesDoc
	if err := yaml.Unmarshal(raw, &d); err != nil {
		return featureFamiliesDoc{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return d, nil
}

// naEvidenceEntry is one docs/packets/feature-na-evidence.yaml entry: the
// positive absence proof for one (op, version) cell that is n-a while a
// same-family sibling is verified.
type naEvidenceEntry struct {
	Op       string `yaml:"op"`
	Version  string `yaml:"version"`
	Evidence string `yaml:"evidence"`
}

// naEvidenceDoc is the parsed shape of docs/packets/feature-na-evidence.yaml.
type naEvidenceDoc struct {
	Entries []naEvidenceEntry `yaml:"entries"`
}

// loadNAEvidence reads and parses feature-na-evidence.yaml at path. A missing
// file is not an error — it returns an empty doc (no entries recorded, so
// every family-inconsistent n-a is reported as a problem).
func loadNAEvidence(path string) (naEvidenceDoc, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return naEvidenceDoc{}, nil
	}
	if err != nil {
		return naEvidenceDoc{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var d naEvidenceDoc
	if err := yaml.Unmarshal(raw, &d); err != nil {
		return naEvidenceDoc{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return d, nil
}

// naEvidenceKey is the (op, version) identity an evidence entry is pinned to.
type naEvidenceKey struct {
	Op      string
	Version string
}

// index builds an Op×Version -> evidence-text lookup, skipping malformed
// entries (empty op or version).
func (d naEvidenceDoc) index() map[naEvidenceKey]string {
	m := make(map[naEvidenceKey]string, len(d.Entries))
	for _, e := range d.Entries {
		if e.Op == "" || e.Version == "" {
			continue
		}
		m[naEvidenceKey{Op: e.Op, Version: e.Version}] = e.Evidence
	}
	return m
}

// naConsistencyResult is the outcome of evaluating the gate over a Matrix.
// Problems drive `matrix --check`'s non-zero exit (design pattern shared with
// evProblems/marker errs in matrixRun). Notes are informational — which
// evidence entries were actually consumed by a real family-inconsistent cell
// (mirrors dispatcher-lint's "baselined family" notes) — so a stale,
// unconsumed evidence entry is visible without failing the build.
type naConsistencyResult struct {
	Problems []string
	Notes    []string
}

// naConsistencyCheck evaluates the feature-family n-a consistency gate
// (task-124) over an already-built Matrix. For each declared family, for
// each version: if at least one member op is StateVerified AND at least one
// member op is StateNA on that version, every StateNA member must have a
// matching feature-na-evidence.yaml entry with non-empty evidence text —
// otherwise a problem string is appended. A family with no verified member on
// a version (the feature genuinely isn't present there at all) or with every
// member verified is untouched — this is legitimate absence, not the
// send/receive inconsistency the gate targets.
//
// versions fixes iteration order for deterministic output; when empty it is
// derived from the matrix rows (sorted).
func naConsistencyCheck(m matrix.Matrix, fams featureFamiliesDoc, ev naEvidenceDoc, versions []string) naConsistencyResult {
	// Index every op row by its Op name. A given op name may have more than
	// one row (e.g. distinct clientbound/serverbound rows sharing a name) —
	// family membership is by op identity, not direction, so all matching
	// rows' cells are consulted.
	rowsByOp := map[string][]matrix.MatrixRow{}
	for _, r := range m.Rows {
		if r.Kind != matrix.RowOp || r.Op == "" {
			continue
		}
		rowsByOp[r.Op] = append(rowsByOp[r.Op], r)
	}
	evIdx := ev.index()
	consumed := map[naEvidenceKey]bool{}

	famNames := make([]string, 0, len(fams.Families))
	for name := range fams.Families {
		famNames = append(famNames, name)
	}
	sort.Strings(famNames)

	vks := versions
	if len(vks) == 0 {
		seen := map[string]bool{}
		for _, r := range m.Rows {
			for vk := range r.Cells {
				seen[vk] = true
			}
		}
		for vk := range seen {
			vks = append(vks, vk)
		}
		sort.Strings(vks)
	}

	var problems []string
	for _, famName := range famNames {
		members := fams.Families[famName]
		for _, vk := range vks {
			verifiedOps := map[string]bool{}
			naOps := map[string]bool{}
			for _, op := range members {
				for _, r := range rowsByOp[op] {
					c, ok := r.Cells[vk]
					if !ok {
						continue
					}
					switch c.State {
					case matrix.StateVerified:
						verifiedOps[op] = true
					case matrix.StateNA:
						naOps[op] = true
					}
				}
			}
			if len(verifiedOps) == 0 || len(naOps) == 0 {
				// Either the whole family is unverified on this version (no
				// sibling proof to hold the n-a to), or nothing is n-a here.
				continue
			}
			verifiedList := sortedSetKeys(verifiedOps)
			naList := sortedSetKeys(naOps)
			for _, naOp := range naList {
				key := naEvidenceKey{Op: naOp, Version: vk}
				evidence, has := evIdx[key]
				if has && strings.TrimSpace(evidence) != "" {
					consumed[key] = true
					continue
				}
				problems = append(problems, fmt.Sprintf(
					"n-a consistency: %s × %s is n-a but sibling %s is verified on %s — add positive absence evidence to docs/packets/feature-na-evidence.yaml (or verify the cell)",
					naOp, vk, verifiedList[0], vk))
			}
		}
	}

	consumedKeys := make([]naEvidenceKey, 0, len(consumed))
	for k := range consumed {
		consumedKeys = append(consumedKeys, k)
	}
	sort.Slice(consumedKeys, func(i, j int) bool {
		if consumedKeys[i].Op != consumedKeys[j].Op {
			return consumedKeys[i].Op < consumedKeys[j].Op
		}
		return consumedKeys[i].Version < consumedKeys[j].Version
	})
	notes := make([]string, 0, len(consumedKeys))
	for _, k := range consumedKeys {
		notes = append(notes, fmt.Sprintf("n-a evidence consumed: %s × %s (docs/packets/feature-na-evidence.yaml)", k.Op, k.Version))
	}

	return naConsistencyResult{Problems: problems, Notes: notes}
}

func sortedSetKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// printNANotes writes the consumed-evidence notes to w, one per line — the
// visible signal that a family-inconsistent n-a is deliberately recorded
// (mirrors dispatcher-lint's baselined-family notes).
func printNANotes(w io.Writer, notes []string) {
	for _, n := range notes {
		_, _ = fmt.Fprintf(w, "note\t%s\n", n)
	}
}
