package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/matrix"
	"gopkg.in/yaml.v3"
)

// doc-freshness (task-169 FR-2.3) is the machine-lintable guard against RC-B
// (doc drift silently reopening). It parses the `packet-process-facts` fenced
// block in docs/packets/PROCESS.md — the single source of truth every playbook
// links to — and asserts every fact matches the tool's ground truth:
//
//   - version_count / version_keys        vs matrix.VersionKeys
//   - dispatcher_lint_baseline_families   vs dispatcher-lint-baseline.yaml exempt_families
//   - family_cap_dispatchers              vs evidence/families.yaml dispatchers
//   - ci_gates                            vs the packet-matrix.yml workflow steps
//
// Read-only. Exit 0 clean; non-zero on any divergence (each printed to stderr).

// docFreshnessConfig parameterises the check so tests can point it at fixtures.
// Defaults match the repo layout (commands run from the repo root).
type docFreshnessConfig struct {
	ProcessMD    string
	BaselineYAML string
	FamiliesYAML string
	WorkflowYML  string
}

func defaultDocFreshnessConfig() docFreshnessConfig {
	return docFreshnessConfig{
		ProcessMD:    filepath.Join("docs", "packets", "PROCESS.md"),
		BaselineYAML: filepath.Join("docs", "packets", "dispatcher-lint-baseline.yaml"),
		FamiliesYAML: filepath.Join("docs", "packets", "evidence", "families.yaml"),
		WorkflowYML:  filepath.Join(".github", "workflows", "packet-matrix.yml"),
	}
}

// processFacts is the parsed `packet-process-facts` YAML block.
type processFacts struct {
	VersionCount                   int      `yaml:"version_count"`
	VersionKeys                    []string `yaml:"version_keys"`
	DispatcherLintBaselineFamilies []string `yaml:"dispatcher_lint_baseline_families"`
	FamilyCapDispatchers           []string `yaml:"family_cap_dispatchers"`
	CIGates                        []string `yaml:"ci_gates"`
	MatrixCheckHardGate            bool     `yaml:"matrix_check_hard_gate"`
}

// ciGateWorkflowSubstr maps each documented CI-gate id to a substring that must
// appear in the workflow's run commands. Kept in sync with PROCESS.md's
// ci_gates list and the workflow steps — both directions are asserted.
var ciGateWorkflowSubstr = map[string]string{
	"packet-audit-tests":  "go test ./...",
	"fname-doc-check":     "fname-doc --check",
	"operations-check":    "operations --check",
	"dispatcher-lint":     "dispatcher-lint",
	"doc-freshness-check": "doc-freshness --check",
	"gate-check":          "gate-check --check",
	"matrix-check":        "matrix --check",
}

func runDocFreshness(args []string, stderr io.Writer) int {
	for _, a := range args {
		switch a {
		case "--check":
			// Accepted for consistency with the other --check gates; the
			// command is always a check (nothing to regenerate).
		case "-h", "--help", "help":
			fmt.Fprintln(stderr, "usage: packet-audit doc-freshness [--check]")
			fmt.Fprintln(stderr, "asserts docs/packets/PROCESS.md packet-process-facts match the tool's ground truth; read-only.")
			return 0
		default:
			fmt.Fprintf(stderr, "packet-audit doc-freshness: unexpected argument %q\n", a)
			return 3
		}
	}
	return docFreshnessRun(defaultDocFreshnessConfig(), os.Stdout, stderr)
}

// docFreshnessRun is the testable core.
func docFreshnessRun(cfg docFreshnessConfig, out, stderr io.Writer) int {
	facts, err := parseProcessFacts(cfg.ProcessMD)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit doc-freshness: %v\n", err)
		return 3
	}

	var diffs []string

	// version_count / version_keys vs matrix.VersionKeys.
	if facts.VersionCount != len(matrix.VersionKeys) {
		diffs = append(diffs, fmt.Sprintf("version_count: doc says %d, ground truth (matrix.VersionKeys) is %d",
			facts.VersionCount, len(matrix.VersionKeys)))
	}
	if !equalStrings(facts.VersionKeys, matrix.VersionKeys) {
		diffs = append(diffs, fmt.Sprintf("version_keys: doc %v != matrix.VersionKeys %v",
			facts.VersionKeys, matrix.VersionKeys))
	}

	// dispatcher_lint_baseline_families vs dispatcher-lint-baseline.yaml.
	baseFams, err := loadBaselineExemptFamilies(cfg.BaselineYAML)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit doc-freshness: %v\n", err)
		return 3
	}
	if !equalStringSets(facts.DispatcherLintBaselineFamilies, baseFams) {
		diffs = append(diffs, fmt.Sprintf("dispatcher_lint_baseline_families: doc %v != dispatcher-lint-baseline.yaml exempt_families %v",
			facts.DispatcherLintBaselineFamilies, baseFams))
	}

	// family_cap_dispatchers vs evidence/families.yaml dispatchers.
	fams, err := matrix.LoadFamilies(cfg.FamiliesYAML)
	if err != nil {
		fmt.Fprintf(stderr, "packet-audit doc-freshness: loading families.yaml %s: %v\n", cfg.FamiliesYAML, err)
		return 3
	}
	if !equalStringSets(facts.FamilyCapDispatchers, fams.Dispatchers) {
		diffs = append(diffs, fmt.Sprintf("family_cap_dispatchers: doc %v != families.yaml dispatchers %v",
			facts.FamilyCapDispatchers, fams.Dispatchers))
	}

	// ci_gates vs the workflow (both directions over the known gate set).
	if gd, err := ciGateDivergences(facts.CIGates, cfg.WorkflowYML); err != nil {
		fmt.Fprintf(stderr, "packet-audit doc-freshness: %v\n", err)
		return 3
	} else {
		diffs = append(diffs, gd...)
	}

	if len(diffs) > 0 {
		fmt.Fprintf(stderr, "packet-audit doc-freshness: %d divergence(s) between %s and the tool's ground truth:\n",
			len(diffs), cfg.ProcessMD)
		for _, d := range diffs {
			fmt.Fprintln(stderr, "  -", d)
		}
		fmt.Fprintln(stderr, "Update the packet-process-facts block in PROCESS.md (or the diverged source) so they agree.")
		return 1
	}
	fmt.Fprintf(out, "doc-freshness: PROCESS.md packet-process-facts agree with the tool (%d versions, %d CI gates).\n",
		len(matrix.VersionKeys), len(facts.CIGates))
	return 0
}

// parseProcessFacts extracts and unmarshals the `packet-process-facts` fenced
// YAML block from PROCESS.md.
func parseProcessFacts(path string) (processFacts, error) {
	var f processFacts
	b, err := os.ReadFile(path)
	if err != nil {
		return f, fmt.Errorf("reading %s: %w", path, err)
	}
	lines := strings.Split(string(b), "\n")
	// Find the marker line, then collect until the closing ``` fence.
	start := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "# packet-process-facts" {
			start = i
			break
		}
	}
	if start < 0 {
		return f, fmt.Errorf("%s: no `# packet-process-facts` block found", path)
	}
	var body []string
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			break
		}
		body = append(body, lines[i])
	}
	if err := yaml.Unmarshal([]byte(strings.Join(body, "\n")), &f); err != nil {
		return f, fmt.Errorf("%s: parsing packet-process-facts: %w", path, err)
	}
	return f, nil
}

// loadBaselineExemptFamilies parses exempt_families from the dispatcher-lint
// baseline yaml.
func loadBaselineExemptFamilies(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var doc dispatcherLintBaseline
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return doc.ExemptFamilies, nil
}

// ciGateDivergences checks, over the known gate set, that every documented gate
// is present in the workflow AND every gate present in the workflow is
// documented.
func ciGateDivergences(documented []string, workflowPath string) ([]string, error) {
	wf, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", workflowPath, err)
	}
	wfStr := string(wf)
	docSet := map[string]bool{}
	for _, g := range documented {
		docSet[g] = true
	}
	var diffs []string
	// Documented gate must be a known id and present in the workflow.
	for _, g := range documented {
		substr, known := ciGateWorkflowSubstr[g]
		if !known {
			diffs = append(diffs, fmt.Sprintf("ci_gates: doc lists unknown gate %q (not in the known gate map)", g))
			continue
		}
		if !strings.Contains(wfStr, substr) {
			diffs = append(diffs, fmt.Sprintf("ci_gates: documented gate %q (%q) is absent from the workflow", g, substr))
		}
	}
	// A gate the workflow runs must be documented.
	for g, substr := range ciGateWorkflowSubstr {
		if strings.Contains(wfStr, substr) && !docSet[g] {
			diffs = append(diffs, fmt.Sprintf("ci_gates: workflow runs %q (%q) but PROCESS.md does not document it", g, substr))
		}
	}
	return diffs, nil
}

// equalStrings compares two slices order-sensitively.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// equalStringSets compares two slices as sets (order-insensitive), treating nil
// and empty as equal.
func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
	}
	for _, c := range seen {
		if c != 0 {
			return false
		}
	}
	return true
}
