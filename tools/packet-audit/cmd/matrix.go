package cmd

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/template"
)

// exitRuntime is the exit code for runtime errors (missing flags parsed, I/O errors, etc.).
// exitBlocker is the exit code for --check stale/conflict failures.
const (
	exitRuntime = 3
	exitBlocker = 1
)

type matrixOpts struct {
	RegistryDir  string
	AuditsDir    string
	TemplatesDir string
	ExportsDir   string
	EvidenceDir  string // consumed from Phase 2 on; empty = no evidence
	PacketLibDir string // consumed from Phase 3 on (marker scan); empty = no markers
	Versions     []string
	OutDir       string
	Check        bool
}

func runMatrix(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("packet-audit matrix", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o matrixOpts
	var versionsCSV string
	fs.StringVar(&o.RegistryDir, "registry-dir", "docs/packets/registry", "registry YAML dir")
	fs.StringVar(&o.AuditsDir, "audits-dir", "docs/packets/audits", "audit reports parent dir")
	fs.StringVar(&o.TemplatesDir, "templates-dir", "services/atlas-configurations/seed-data/templates", "tenant seed templates dir")
	fs.StringVar(&o.ExportsDir, "exports-dir", "docs/packets/ida-exports", "IDA export JSON dir")
	fs.StringVar(&o.EvidenceDir, "evidence-dir", "docs/packets/evidence", "evidence ledger dir")
	fs.StringVar(&o.PacketLibDir, "packet-lib", "libs/atlas-packet", "atlas-packet root for marker scanning")
	fs.StringVar(&versionsCSV, "versions", strings.Join(matrix.VersionKeys, ","), "comma-separated version keys")
	fs.StringVar(&o.OutDir, "out-dir", "docs/packets/audits", "output dir for STATUS.md/status.json")
	fs.BoolVar(&o.Check, "check", false, "CI mode: verify committed outputs are current; fail on conflicts/drift")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 3
	}
	// TrimSpace each entry and drop empties (handles "v83, v84" or trailing commas).
	raw := strings.Split(versionsCSV, ",")
	for _, v := range raw {
		if s := strings.TrimSpace(v); s != "" {
			o.Versions = append(o.Versions, s)
		}
	}
	return matrixRun(o, os.Stdout, stderr)
}

func matrixRun(o matrixOpts, stdout, stderr io.Writer) int {
	reg, err := opregistry.LoadDir(o.RegistryDir, o.Versions)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	in := matrix.Inputs{Registry: reg,
		Reports:        map[string]map[string]matrix.LoadedReport{},
		Routed:         map[string]map[matrix.RouteKey]bool{},
		RoutedAnywhere: map[matrix.RouteKey]bool{},
		Evidence:       map[matrix.EvKey]matrix.EvidenceStatus{},
		Tier1:          map[string]bool{},
		Markers:        map[matrix.EvKey]matrix.MarkerStatus{},
	}
	hashes := map[string]string{}
	exportPaths := map[string]string{}
	for _, vk := range o.Versions {
		reps, err := matrix.LoadReports(filepath.Join(o.AuditsDir, vk))
		if err != nil {
			fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
			return exitRuntime
		}
		in.Reports[vk] = reps
		in.Routed[vk] = map[matrix.RouteKey]bool{}
		tp := templatePathIn(o.TemplatesDir, vk)
		if t, err := template.Load(tp); os.IsNotExist(err) {
			// Missing template is a warning: grading continues without routing data.
			fmt.Fprintf(stderr, "packet-audit matrix: warning: no template for %s (%v)\n", vk, err)
		} else if err != nil {
			// Other errors (permission denied, corrupt JSON, etc.) are fatal.
			fmt.Fprintf(stderr, "packet-audit matrix: error loading template for %s: %v\n", vk, err)
			return exitRuntime
		} else {
			for op := range t.Writers() {
				k := matrix.RouteKey{Opcode: op, Dir: opregistry.DirClientbound}
				in.Routed[vk][k] = true
				in.RoutedAnywhere[k] = true
			}
			for op := range t.Handlers() {
				k := matrix.RouteKey{Opcode: op, Dir: opregistry.DirServerbound}
				in.Routed[vk][k] = true
				in.RoutedAnywhere[k] = true
			}
		}
		ep := exportPathIn(o.ExportsDir, vk)
		if raw, err := os.ReadFile(ep); os.IsNotExist(err) {
			// Missing export file: warn but continue (not all versions have exports yet).
			fmt.Fprintf(stderr, "packet-audit matrix: warning: no export file for %s (%s)\n", vk, ep)
		} else if err != nil {
			// Unreadable-but-existing export is a hard failure.
			fmt.Fprintf(stderr, "packet-audit matrix: error reading export for %s: %v\n", vk, err)
			return exitRuntime
		} else {
			hashes[vk] = fmt.Sprintf("%x", sha256.Sum256(raw))
			exportPaths[vk] = ep
		}
	}

	evStatus, evProblems, err := matrix.BuildEvidenceInputs(o.EvidenceDir, exportPaths)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	in.Evidence = evStatus
	// Design §13: an evidence record for a (packet, version) with no audit
	// report is dangling — a --check failure.
	for k := range evStatus {
		if _, ok := reportForPacket(in.Reports[k.Version], k.Packet); !ok {
			evProblems = append(evProblems,
				fmt.Sprintf("dangling evidence: %s × %s has no audit report", k.Packet, k.Version))
		}
	}

	m := matrix.Build(in, o.Versions)
	m.ExportHashes = hashes
	m.ToolSHA = toolTreeSHA()

	md := matrix.RenderMarkdown(m, o.Versions)
	js, err := matrix.RenderJSON(m)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	mdPath := filepath.Join(o.OutDir, "STATUS.md")
	jsPath := filepath.Join(o.OutDir, "status.json")

	if o.Check {
		return matrixCheck(m, md, js, mdPath, jsPath, evProblems, stderr)
	}
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	if err := os.WriteFile(jsPath, js, 0o644); err != nil {
		fmt.Fprintf(stderr, "packet-audit matrix: %v\n", err)
		return exitRuntime
	}
	fmt.Fprintf(stdout, "wrote %s and %s\n", mdPath, jsPath)
	return 0
}

// matrixCheck implements the full --check semantics (design §10.1):
// fails on stale committed outputs, evidence problems (drift/dangling), and
// any conflict cell (conflicts are blockers, never allowlisted).
func matrixCheck(m matrix.Matrix, md string, js []byte, mdPath, jsPath string, evProblems []string, stderr io.Writer) int {
	fail := false
	for _, p := range evProblems {
		fmt.Fprintf(stderr, "matrix --check: %s\n", p)
		fail = true
	}
	for _, r := range m.Rows {
		for vk, c := range r.Cells {
			if c.State == matrix.StateConflict {
				name := r.Op
				if name == "" {
					name = r.Packet
				}
				fmt.Fprintf(stderr, "matrix --check: conflict %s × %s — %s\n", name, vk, c.Note)
				fail = true
			}
		}
	}
	if cur, err := os.ReadFile(mdPath); err != nil || string(cur) != md {
		fmt.Fprintf(stderr, "matrix --check: %s is stale — regenerate and commit\n", mdPath)
		fail = true
	}
	if cur, err := os.ReadFile(jsPath); err != nil || string(cur) != string(js) {
		fmt.Fprintf(stderr, "matrix --check: %s is stale\n", jsPath)
		fail = true
	}
	if fail {
		return exitBlocker
	}
	return 0
}

// reportForPacket finds the LoadedReport (if any) for a given packet id within
// a version's report map. It checks each report using PacketID for normalization.
// Task 3.2 reuses this helper.
func reportForPacket(reps map[string]matrix.LoadedReport, pkt string) (matrix.LoadedReport, bool) {
	for _, r := range reps {
		if matrix.PacketID(r) == pkt {
			return r, true
		}
	}
	return matrix.LoadedReport{}, false
}

func templatePathIn(dir, vk string) string {
	return filepath.Join(dir, filepath.Base(matrix.TemplatePath(vk)))
}

func exportPathIn(dir, vk string) string {
	return filepath.Join(dir, filepath.Base(matrix.ExportPath(vk)))
}

// toolTreeSHA returns `git rev-parse HEAD:tools/packet-audit` (the tree SHA of
// the tool itself), or "unknown" outside a git checkout.
func toolTreeSHA() string {
	out, err := exec.Command("git", "rev-parse", "HEAD:tools/packet-audit").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
