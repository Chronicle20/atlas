package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"gopkg.in/yaml.v3"
)

// gate-check (task-169 FR-3.1b) is the BEHAVIORAL guard for the off-by-one
// version-gate class. gate-lint (FR-3.1a) catches the risky code SHAPE; this
// catches the risky COVERAGE: a version-gated wire divergence is only
// trustworthy if a VERIFIED byte-fixture pins the encoding on BOTH sides of the
// boundary. If only one straddling version is fixture-verified, a regression on
// the unpinned side (or a wrong boundary) would pass unnoticed.
//
// It reads docs/packets/gates.yaml — a machine-readable registry of the gated
// divergences (packet, boundary, the two adjacent straddling version keys) —
// and asserts, per gate, that some matrix row for that packet is `verified` at
// both the lower and upper version key. Read-only. Default mode reports and
// exits 0; `--check` exits non-zero when any `full` gate is missing a side.

// gateEntry is one row of gates.yaml. See the header comment in gates.yaml for
// the authoring contract.
type gateEntry struct {
	Packet    string `yaml:"packet"`
	Direction string `yaml:"direction"`
	Field     string `yaml:"field"`
	Boundary  string `yaml:"boundary"`
	Lower     string `yaml:"lower_version_key"`
	Upper     string `yaml:"upper_version_key"`
	Expect    string `yaml:"expect"` // "" / "full" (both sides required) or "partial" (one side may be unpinned)
	Reason    string `yaml:"reason"` // required when expect: partial
}

type gatesDoc struct {
	Gates []gateEntry `yaml:"gates"`
}

type gateCheckConfig struct {
	GatesPath  string
	StatusPath string
	Check      bool
}

func defaultGateCheckConfig() gateCheckConfig {
	return gateCheckConfig{
		GatesPath:  filepath.Join("docs", "packets", "gates.yaml"),
		StatusPath: filepath.Join("docs", "packets", "audits", "status.json"),
	}
}

func runGateCheck(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit gate-check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfg := defaultGateCheckConfig()
	fs.StringVar(&cfg.GatesPath, "gates", cfg.GatesPath, "gates.yaml path")
	fs.StringVar(&cfg.StatusPath, "status", cfg.StatusPath, "status.json path")
	fs.BoolVar(&cfg.Check, "check", false, "exit non-zero when any full gate is missing a verified side")
	if err := fs.Parse(args); err != nil {
		return 3
	}
	return gateCheckRun(cfg, os.Stdout, stderr)
}

// gateCheckResult classifies one gate.
type gateCheckResult struct {
	gate         gateEntry
	lowerOK      bool
	upperOK      bool
	packetKnown  bool // at least one row matched packet+direction
	configErr    string
}

func (r gateCheckResult) ok() bool {
	if r.configErr != "" {
		return false
	}
	if r.gate.Expect == "partial" {
		return r.lowerOK || r.upperOK
	}
	return r.lowerOK && r.upperOK
}

// gateCheckRun is the testable core.
func gateCheckRun(cfg gateCheckConfig, out, stderr io.Writer) int {
	doc, err := loadGates(cfg.GatesPath)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit gate-check: %v\n", err)
		return 3
	}
	m, err := matrix.LoadMatrix(cfg.StatusPath)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit gate-check: %v\n", err)
		return 3
	}

	known := map[string]bool{}
	for _, vk := range matrix.VersionKeys {
		known[vk] = true
	}

	results := make([]gateCheckResult, 0, len(doc.Gates))
	for _, g := range doc.Gates {
		results = append(results, evalGate(g, m, known))
	}

	var failed []gateCheckResult
	for _, r := range results {
		if !r.ok() {
			failed = append(failed, r)
		}
	}
	sort.SliceStable(failed, func(i, j int) bool {
		return failed[i].gate.Packet < failed[j].gate.Packet
	})

	w := out
	if cfg.Check {
		w = stderr
	}
	for _, r := range failed {
		fmt.Fprintln(w, describeGateFailure(r))
	}

	total := len(results)
	partial := 0
	for _, r := range results {
		if r.gate.Expect == "partial" {
			partial++
		}
	}

	if cfg.Check {
		if len(failed) > 0 {
			fmt.Fprintf(stderr, "packet-audit gate-check: %d of %d gate(s) lack a verified byte-fixture on a straddling version (see above).\n", len(failed), total)
			return 1
		}
		fmt.Fprintf(out, "gate-check: all %d gate(s) have verified byte-fixtures on both straddling versions (%d partial-by-design).\n", total, partial)
		return 0
	}
	fmt.Fprintf(out, "gate-check: %d gate(s), %d failing, %d partial-by-design (report-only; --check to gate).\n", total, len(failed), partial)
	return 0
}

// evalGate resolves a gate against the matrix using EXISTS-a-verified-row
// semantics: the gate's fixture pair exists if some row for (packet, direction)
// is verified at the lower key AND some row is verified at the upper key. Using
// "exists a row" gracefully handles packets that appear as multiple rows
// (op + sub-struct variants).
func evalGate(g gateEntry, m matrix.Matrix, known map[string]bool) gateCheckResult {
	r := gateCheckResult{gate: g}
	if g.Expect != "" && g.Expect != "full" && g.Expect != "partial" {
		r.configErr = fmt.Sprintf("unknown expect %q (want full|partial)", g.Expect)
		return r
	}
	if g.Expect == "partial" && g.Reason == "" {
		r.configErr = "expect: partial requires a reason"
		return r
	}
	if !known[g.Lower] {
		r.configErr = fmt.Sprintf("unknown lower_version_key %q", g.Lower)
		return r
	}
	if !known[g.Upper] {
		r.configErr = fmt.Sprintf("unknown upper_version_key %q", g.Upper)
		return r
	}
	for _, row := range m.Rows {
		if row.Packet != g.Packet {
			continue
		}
		if g.Direction != "" && string(row.Direction) != g.Direction {
			continue
		}
		r.packetKnown = true
		if c, ok := row.Cells[g.Lower]; ok && c.State == matrix.StateVerified {
			r.lowerOK = true
		}
		if c, ok := row.Cells[g.Upper]; ok && c.State == matrix.StateVerified {
			r.upperOK = true
		}
	}
	if !r.packetKnown {
		r.configErr = fmt.Sprintf("no matrix row for packet %q direction %q", g.Packet, g.Direction)
	}
	return r
}

func describeGateFailure(r gateCheckResult) string {
	tag := fmt.Sprintf("%s [%s]", r.gate.Packet, r.gate.Boundary)
	if r.configErr != "" {
		return fmt.Sprintf("  ✗ %s — config error: %s", tag, r.configErr)
	}
	var missing []string
	if !r.lowerOK {
		missing = append(missing, r.gate.Lower)
	}
	if !r.upperOK {
		missing = append(missing, r.gate.Upper)
	}
	return fmt.Sprintf("  ✗ %s — no verified byte-fixture at: %v (field: %s)", tag, missing, r.gate.Field)
}

func loadGates(path string) (gatesDoc, error) {
	var doc gatesDoc
	b, err := os.ReadFile(path)
	if err != nil {
		return doc, fmt.Errorf("reading %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return doc, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(doc.Gates) == 0 {
		return doc, fmt.Errorf("%s: no gates defined", path)
	}
	return doc, nil
}
