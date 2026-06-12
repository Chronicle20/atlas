package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// decomposeOpts is the injectable configuration for the decompose driver. Every
// value affecting the extended baseline + report is fixed up front, so re-running
// with identical opts+client produces byte-identical output. The flag wrapper
// also carries IDAURL/IDATimeout (used only to build the client + default paths),
// which the core does not need.
type decomposeOpts struct {
	Version      string // target version key (used only by the flag wrapper for defaults)
	Baseline     string // baseline export JSON (read-only)
	AuditDir     string // committed audit dir for this version
	Out          string // extended baseline JSON output
	Report       string // markdown report
	DescentDepth int
}

// decomposeClass is the per-FName classification recorded in the report.
type decomposeClass string

const (
	classUpgraded   decomposeClass = "upgraded"       // truncated read-order (F is a strict prefix-extension of H): replaced with faithful order
	classUnchanged  decomposeClass = "unchanged"      // faithful order matches hand reads under width tolerance (no truncation)
	classDivergence decomposeClass = "divergence"     // mid-stream mismatch or F shorter than H — candidate real bug; entry left untouched
	classNeedsDisp  decomposeClass = "needs-dispatch" // # entry: per-mode, needs a dispatch selector (validate/infer path)
	classError      decomposeClass = "error"          // base resolve failed (soft-fail or other) — entry left untouched
	classMissing    decomposeClass = "missing"        // non-✅ FName not present in the baseline
	classBadAudit   decomposeClass = "bad-audit"      // audit .md missing/unparseable — FName could not be derived
)

// decomposeResult is one classified FName outcome.
type decomposeResult struct {
	FName  string
	Class  decomposeClass
	Detail string
}

// reAuditHeader extracts the FName from an audit `.md` header line of the form
// `# <Packet> (← `<FName>`)`. The FName is the backtick-delimited token inside
// the parentheses.
var reAuditHeader = regexp.MustCompile("^#\\s+.*\\(\\s*←\\s*`([^`]+)`\\s*\\)")

// decomposeRun upgrades the baseline's truncated/opaque read-orders for NON-✅,
// SINGLE-SHAPE (non-`#`) packets using the live exporter, and writes an extended
// baseline + a report. It NEVER mutates the input baseline; `#`-mode entries are
// SKIPPED (they need a dispatch selector — handled by the validate/infer path).
//
// A base decompilation soft-fail (or any other ResolveLive error) records the
// FName as `error` and leaves the entry untouched; the run CONTINUES (one bad
// address never aborts). Returns a process exit code.
func decomposeRun(opts decomposeOpts, client idasrc.MCPClient, stdout io.Writer) int {
	// 1. Parse the audit dir → NON-✅ FName set (+ bad-audit notes).
	nonOK, badAudit, err := parseNonOKFNames(opts.AuditDir)
	if err != nil {
		fmt.Fprintln(stdout, "decompose: read audit dir:", err)
		return 3
	}

	// 2. Load the baseline, index entries by FName.
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "decompose: load baseline:", err)
		return 3
	}
	byFName := map[string]idasrc.BaselineEntry{}
	for _, e := range src.Entries() {
		byFName[e.FName] = e
	}

	ctx := context.Background()
	var results []decomposeResult
	// upgrades: FName -> the faithful resolved read-order, as export rawCalls.
	upgrades := map[string][]rawCallOut{}

	for _, name := range badAudit {
		results = append(results, decomposeResult{FName: name, Class: classBadAudit, Detail: "audit .md missing or unparseable"})
	}

	// 3. Process each NON-✅ FName.
	fnames := make([]string, 0, len(nonOK))
	for fn := range nonOK {
		fnames = append(fnames, fn)
	}
	sort.Strings(fnames)

	for _, fn := range fnames {
		entry, ok := byFName[fn]
		if !ok {
			results = append(results, decomposeResult{FName: fn, Class: classMissing, Detail: "not present in baseline"})
			continue
		}
		if strings.Contains(fn, "#") {
			results = append(results, decomposeResult{FName: fn, Class: classNeedsDisp, Detail: "per-mode entry needs a dispatch selector"})
			continue
		}

		f, rerr := idasrc.ResolveLive(ctx, client, entry.Address, entry.Direction,
			idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil {
			detail := "base decompilation failed"
			if !idasrc.IsDecompilationFailed(rerr) {
				detail = "base resolve failed: " + rerr.Error()
			}
			results = append(results, decomposeResult{FName: fn, Class: classError, Detail: detail})
			continue
		}

		switch classifyReads(entry.HandCalls, f.Calls) {
		case classUnchanged:
			// Faithful order matches hand reads under width tolerance: the
			// non-✅ verdict is a representation/elsewhere issue, NOT truncation.
			// Leave the entry untouched.
			results = append(results, decomposeResult{FName: fn, Class: classUnchanged,
				Detail: "faithful order matches hand reads (not truncation)"})
		case classUpgraded:
			// Strict prefix-extension: F is longer than H and every H[i] is
			// field-equivalent to F[i]. This is provably-safe truncation — the
			// export stopped short and the client reads more. Replace the
			// entry's calls (OUTPUT only) with the faithful order.
			upgrades[fn] = fieldsToRawCalls(f.Calls)
			results = append(results, decomposeResult{FName: fn, Class: classUpgraded,
				Detail: fmt.Sprintf("hand %d ops → faithful %d ops", len(entry.HandCalls), len(f.Calls))})
		default:
			// classDivergence: mid-stream mismatch at a common prefix position,
			// or faithful is SHORTER than hand (Atlas writes a field the client
			// does not read). Either case is a candidate real wire bug — NEVER
			// silently overwritten. Leave the entry untouched and flag for triage.
			results = append(results, decomposeResult{FName: fn, Class: classDivergence,
				Detail: fmt.Sprintf("hand %d ops vs faithful %d ops — mid-stream mismatch or over-read; needs human triage",
					len(entry.HandCalls), len(f.Calls))})
		}
	}

	// 4. Write the extended baseline (deep copy of input with upgraded calls
	//    replaced), then the report + roll-up. NEVER write to opts.Baseline.
	if code := writeExtendedBaseline(opts.Baseline, opts.Out, upgrades, stdout); code != 0 {
		return code
	}
	return writeDecomposeReport(opts, results, stdout)
}

// rawCallOut mirrors the baseline JSON "calls" element. Only the fields the
// faithful read-order can carry are emitted; empty optional fields are omitted so
// the output stays clean and deterministic.
type rawCallOut struct {
	Op      string `json:"op"`
	Comment string `json:"comment"`
	Guard   string `json:"guard,omitempty"`
}

// fieldsToRawCalls converts a faithful resolved read-order to export rawCalls
// (Primitive → op string via RawOp, carrying Comment + Guard).
func fieldsToRawCalls(calls []idasrc.FieldCall) []rawCallOut {
	out := make([]rawCallOut, 0, len(calls))
	for _, c := range calls {
		out = append(out, rawCallOut{Op: c.Op.RawOp(), Comment: c.Comment, Guard: c.Guard})
	}
	return out
}

// classifyReads classifies the relationship between a hand-authored read list (H)
// and the live faithful read list (F) into one of three classes:
//
//   - classUnchanged: same length AND fieldEquivalent(H[i], F[i]) for all i.
//     The non-✅ audit verdict is a representation/elsewhere issue, not truncation.
//     The entry is left untouched.
//
//   - classUpgraded: F is a STRICT PREFIX-EXTENSION of H — i.e. len(F) > len(H)
//     AND fieldEquivalent(H[i], F[i]) for ALL i in 0..len(H)-1. This is
//     unambiguously-safe truncation (the export stopped short; the client reads
//     more). The entry's calls are replaced with F in the output.
//
//   - classDivergence: any mismatch within the common prefix (fieldEquivalent
//     returns false for some i < min(len(H),len(F))), OR len(F) < len(H) (hand
//     read more than the live order — over-read or Atlas writes a field the client
//     does not consume). These are candidate real wire bugs and must NEVER be
//     silently overwritten; they are flagged for human triage.
//
// Only classUpgraded modifies the output entry; the other two classes leave the
// entry byte-identical to the input baseline.
func classifyReads(hand, faithful []idasrc.FieldCall) decomposeClass {
	v, _ := idasrc.ValidateShape(hand, faithful)
	if v == idasrc.ShapeVerified {
		// Full length+position match under width tolerance.
		return classUnchanged
	}

	// Check for a strict prefix-extension: F longer than H, all H positions
	// field-equivalent to the corresponding F positions.
	if len(faithful) > len(hand) {
		prefixOK := true
		for i := range hand {
			if !idasrc.FieldEquivalent(hand[i].Op, faithful[i].Op) {
				prefixOK = false
				break
			}
		}
		if prefixOK {
			return classUpgraded
		}
	}

	// Everything else: mid-stream mismatch, F shorter than H, or a prefix
	// mismatch when F is longer. Flag as divergence — do not overwrite.
	return classDivergence
}

// parseNonOKFNames scans an audit dir for per-packet `<Packet>.json` files. A
// packet is NON-✅ when any Row has Verdict ∈ {2 blocker, 3 deferred, 4 unresolved}.
// For each such packet it reads the sibling `<Packet>.md` header to derive the
// FName. Returns the NON-✅ FName set and the list of packets whose `.md` was
// missing/unparseable (bad-audit, derived from the json base name).
func parseNonOKFNames(auditDir string) (nonOK map[string]bool, badAudit []string, err error) {
	jsons, err := filepath.Glob(filepath.Join(auditDir, "*.json"))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(jsons)
	nonOK = map[string]bool{}
	for _, jp := range jsons {
		b, rerr := os.ReadFile(jp)
		if rerr != nil {
			return nil, nil, rerr
		}
		var doc struct {
			Rows []struct {
				Verdict int `json:"Verdict"`
			} `json:"Rows"`
		}
		if jerr := json.Unmarshal(b, &doc); jerr != nil {
			// Unparseable .json: treat as bad-audit keyed by base name.
			badAudit = append(badAudit, strings.TrimSuffix(filepath.Base(jp), ".json"))
			continue
		}
		isNonOK := false
		for _, r := range doc.Rows {
			if r.Verdict == 2 || r.Verdict == 3 || r.Verdict == 4 {
				isNonOK = true
				break
			}
		}
		if !isNonOK {
			continue
		}
		mdPath := strings.TrimSuffix(jp, ".json") + ".md"
		fname, ok := fnameFromAuditMD(mdPath)
		if !ok {
			badAudit = append(badAudit, strings.TrimSuffix(filepath.Base(jp), ".json"))
			continue
		}
		nonOK[fname] = true
	}
	return nonOK, badAudit, nil
}

// fnameFromAuditMD reads an audit `.md` and returns the FName from its
// `# <Packet> (← `<FName>`)` header. Returns false if the file is missing or the
// header is unparseable.
func fnameFromAuditMD(path string) (string, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if m := reAuditHeader.FindStringSubmatch(line); m != nil {
			return strings.TrimSpace(m[1]), true
		}
	}
	return "", false
}

// extendedBaseline round-trips the baseline JSON for output. The functions map is
// kept as json.RawMessage per FName so every non-upgraded entry is byte-preserved
// exactly as authored; only an upgraded entry's "calls" array is rewritten.
type extendedBaseline struct {
	Binary      string                     `json:"binary"`
	MD5         string                     `json:"md5"`
	GeneratedAt string                     `json:"generated_at"`
	Functions   map[string]json.RawMessage `json:"functions"`
}

// writeExtendedBaseline loads the input baseline, replaces only the upgraded
// FNames' "calls" arrays, and marshals deterministically to opts.Out. The input
// file is read-only — it is never written. Non-upgraded entries are preserved as
// their original raw JSON. Returns a process exit code.
func writeExtendedBaseline(baselinePath, outPath string, upgrades map[string][]rawCallOut, stdout io.Writer) int {
	b, err := os.ReadFile(baselinePath)
	if err != nil {
		fmt.Fprintln(stdout, "decompose: read baseline:", err)
		return 3
	}
	var doc extendedBaseline
	if err := json.Unmarshal(b, &doc); err != nil {
		fmt.Fprintln(stdout, "decompose: parse baseline:", err)
		return 3
	}

	for fname, calls := range upgrades {
		raw, ok := doc.Functions[fname]
		if !ok {
			// Should not happen (upgrades only come from baseline entries), but
			// never fabricate an entry.
			continue
		}
		// Decode the original entry into an ordered map so every sibling field
		// (address, direction, dispatch, notes, …) is preserved, then replace
		// only "calls".
		var fn map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fn); err != nil {
			fmt.Fprintln(stdout, "decompose: parse entry "+fname+":", err)
			return 3
		}
		cb, err := json.Marshal(calls)
		if err != nil {
			fmt.Fprintln(stdout, "decompose: marshal calls "+fname+":", err)
			return 3
		}
		fn["calls"] = cb
		nb, err := json.Marshal(fn)
		if err != nil {
			fmt.Fprintln(stdout, "decompose: marshal entry "+fname+":", err)
			return 3
		}
		doc.Functions[fname] = nb
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(stdout, "decompose: marshal baseline:", err)
		return 3
	}
	out = append(out, '\n')

	if dir := filepath.Dir(outPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "decompose: mkdir:", err)
			return 3
		}
	}
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		fmt.Fprintln(stdout, "decompose: write extended baseline:", err)
		return 3
	}
	return 0
}

// writeDecomposeReport writes the deterministic markdown report (per-FName
// classification bucketed by class, sorted within each section) and prints the
// roll-up counts to stdout. Returns a process exit code.
func writeDecomposeReport(opts decomposeOpts, results []decomposeResult, stdout io.Writer) int {
	order := []decomposeClass{classUpgraded, classUnchanged, classDivergence, classNeedsDisp, classError, classMissing, classBadAudit}
	buckets := map[decomposeClass][]decomposeResult{}
	for _, r := range results {
		buckets[r.Class] = append(buckets[r.Class], r)
	}
	for _, c := range order {
		s := buckets[c]
		sort.Slice(s, func(i, j int) bool { return s[i].FName < s[j].FName })
		buckets[c] = s
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# decompose report\n\n")
	fmt.Fprintf(&b, "upgraded %d / unchanged %d / divergence %d / needs-dispatch %d / error %d / missing %d / bad-audit %d\n\n",
		len(buckets[classUpgraded]), len(buckets[classUnchanged]), len(buckets[classDivergence]),
		len(buckets[classNeedsDisp]), len(buckets[classError]), len(buckets[classMissing]), len(buckets[classBadAudit]))
	for _, c := range order {
		fmt.Fprintf(&b, "## %s\n\n", c)
		for _, r := range buckets[c] {
			if r.Detail != "" {
				fmt.Fprintf(&b, "- %s — %s\n", r.FName, r.Detail)
			} else {
				fmt.Fprintf(&b, "- %s\n", r.FName)
			}
		}
		fmt.Fprintln(&b)
	}

	if dir := filepath.Dir(opts.Report); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "decompose: mkdir report:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Report, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(stdout, "decompose: write report:", err)
		return 3
	}

	fmt.Fprintf(stdout, "upgraded %d / unchanged %d / divergence %d / needs-dispatch %d / error %d / missing %d / bad-audit %d\n",
		len(buckets[classUpgraded]), len(buckets[classUnchanged]), len(buckets[classDivergence]),
		len(buckets[classNeedsDisp]), len(buckets[classError]), len(buckets[classMissing]), len(buckets[classBadAudit]))
	return 0
}
