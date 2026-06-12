package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// validateOpts is the injectable configuration for the validate driver. Every
// value affecting the output report is fixed up front, so re-running with
// identical opts+client produces a byte-identical report. The flag wrapper also
// carries Version/IDAURL/IDATimeout (used only to build the client + default
// paths), which the core does not need.
type validateOpts struct {
	Baseline     string // path to the version's baseline export JSON (with dispatch annotations)
	Report       string // output report path (markdown)
	Allowlist    string // path to _unimplemented.json (missing file = empty allowlist)
	DescentDepth int
}

// shapeResult is one validated baseline entry's outcome. Bucket, when non-empty,
// overrides the Verdict-derived bucket ("missing-mode" / "extra-mode" from the
// case<->mode bijection check); an empty Bucket classifies by Verdict.
type shapeResult struct {
	FName   string
	Verdict idasrc.ShapeVerdict
	Detail  string
	Bucket  string
}

// validateRun loads the baseline, groups entries by base address, decompiles
// each base ONCE via ResolveLive (cached by address), extracts each entry's
// wire shape by its Dispatch, validates it against the hand-authored reads, and
// writes a deterministic markdown report. It NEVER mutates the baseline.
//
// A base decompilation soft-fail (or any other ResolveLive error) marks every
// entry at that address unverifiable and the run CONTINUES — one bad address
// never aborts the whole validation. Returns a process exit code.
func validateRun(opts validateOpts, client idasrc.MCPClient, stdout io.Writer) int {
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "validate: load baseline:", err)
		return 3
	}
	entries := src.Entries()

	// Group entry indices by base address, in sorted-address order. The first
	// entry's Direction at each address is used for the (single) decompile.
	byAddr := map[string][]int{}
	var addrOrder []string
	for i := range entries {
		a := entries[i].Address
		if _, seen := byAddr[a]; !seen {
			addrOrder = append(addrOrder, a)
		}
		byAddr[a] = append(byAddr[a], i)
	}
	sort.Strings(addrOrder)

	allow, err := idasrc.LoadAllowlist(opts.Allowlist)
	if err != nil {
		fmt.Fprintln(stdout, "validate: load allowlist:", err)
		return 3
	}

	ctx := context.Background()
	results := make([]shapeResult, 0, len(entries))

	// Per-BASE-handler dispatch accumulation. A handler whose #Mode entries are
	// split across multiple addresses (the v95 decompiler outlines party/guild
	// case bodies into separate functions) must be diffed ONCE against the union
	// of its client cases — not once per address, which double-counts every case
	// and falsely reports a case bound at a sibling address as missing.
	type handlerAgg struct {
		disc        string
		clientCases map[int64]bool
		bound       map[int64]string // case -> a representative #Mode FName
		order       int
	}
	aggs := map[string]*handlerAgg{}
	aggOrder := 0

	for _, addr := range addrOrder {
		idxs := byAddr[addr]
		dir := entries[idxs[0]].Direction
		f, rerr := idasrc.ResolveLive(ctx, client, addr, dir, idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil {
			// Soft-fail or any other error: every entry at this address is
			// unverifiable; continue (never abort the run).
			detail := "base decompilation failed"
			if !idasrc.IsDecompilationFailed(rerr) {
				detail = "base resolve failed: " + rerr.Error()
			}
			for _, i := range idxs {
				results = append(results, shapeResult{
					FName:   entries[i].FName,
					Verdict: idasrc.ShapeUnverifiable,
					Detail:  detail,
				})
			}
			continue
		}
		for _, i := range idxs {
			e := entries[i]
			var verdict idasrc.ShapeVerdict
			var detail string
			// A `#Mode` entry's verdict depends on its dispatch and the live function's
			// shape:
			//   - With a selector: extract that branch and validate it; a selector that
			//     matches nothing is unverifiable (not a false divergence).
			//   - Empty dispatch + LEAF function (no multi-way dispatch): the whole
			//     function IS this entry's wire shape — validate it flat.
			//   - Empty dispatch + multi-way dispatcher: genuinely not extractable
			//     without a selector — unverifiable.
			// A non-`#` entry is always flat (its whole function is its wire shape).
			isMode := strings.Contains(e.FName, "#")
			switch {
			case isMode && len(e.Dispatch) > 0:
				live := idasrc.ExtractShape(f, e.Dispatch)
				if len(live) == 0 {
					verdict = idasrc.ShapeUnverifiable
					detail = "per-mode selector matched no reads"
				} else {
					verdict, detail = idasrc.ValidateShape(e.HandCalls, live)
				}
			case isMode && !f.HasMultiwayDispatch:
				if len(f.Calls) == 0 {
					// Leaf decompile yielded no reads — extraction failed, not a
					// zero-field wire shape. Unverifiable, never a false divergence.
					verdict = idasrc.ShapeUnverifiable
					detail = "leaf function yielded no extractable reads"
				} else {
					verdict, detail = idasrc.ValidateShape(e.HandCalls, f.Calls)
				}
			case isMode:
				verdict = idasrc.ShapeUnverifiable
				detail = "per-mode shape not extractable (no usable dispatch selector)"
			default:
				verdict, detail = idasrc.ValidateShape(e.HandCalls, idasrc.ExtractShape(f, e.Dispatch))
			}
			results = append(results, shapeResult{FName: e.FName, Verdict: verdict, Detail: detail})
		}

		// Accumulate this address's dispatch bindings + client case-set into the
		// owning base handler (computed once after the loop).
		for _, i := range idxs {
			e := entries[i]
			// Only numeric-case equality selectors are bijection bindings. Default
			// and verbatim ({Guard}) selectors have no numeric case (Case==0) and
			// must NOT pollute the equality-based case<->mode completeness check.
			if !strings.Contains(e.FName, "#") || len(e.Dispatch) == 0 ||
				e.Dispatch[0].Default || e.Dispatch[0].Guard != "" {
				continue
			}
			base := e.FName[:strings.Index(e.FName, "#")]
			a := aggs[base]
			if a == nil {
				a = &handlerAgg{disc: e.Dispatch[0].Discriminator, clientCases: map[int64]bool{}, bound: map[int64]string{}, order: aggOrder}
				aggOrder++
				aggs[base] = a
			}
			a.bound[e.Dispatch[0].Case] = e.FName
			if f.CaseLabels != nil {
				if cs := f.CaseLabels[e.Dispatch[0].Discriminator]; cs != nil {
					for _, c := range cs.Values() {
						a.clientCases[c] = true
					}
				}
			}
		}
	}

	// Per-handler bijection: diff each handler's unified client case-set against
	// its bound #Mode cases (across all addresses), minus the allowlist.
	bases := make([]string, 0, len(aggs))
	for b := range aggs {
		bases = append(bases, b)
	}
	sort.Slice(bases, func(i, j int) bool { return aggs[bases[i]].order < aggs[bases[j]].order })
	for _, base := range bases {
		a := aggs[base]
		clientVals := make([]int64, 0, len(a.clientCases))
		for c := range a.clientCases {
			clientVals = append(clientVals, c)
		}
		bindings := make([]idasrc.ModeBinding, 0, len(a.bound))
		for c, fn := range a.bound {
			bindings = append(bindings, idasrc.ModeBinding{FName: fn, Case: c})
		}
		bij := idasrc.Bijection(idasrc.NewCaseSet(clientVals), bindings)
		for _, mc := range bij.Missing {
			bucket := "missing-mode"
			detail := "client dispatch case has no Atlas #Mode writer"
			if allow != nil && allow.Suppressed(base, mc) {
				bucket = "allowlisted"
				detail = "client dispatch case intentionally unimplemented (allowlisted)"
			}
			results = append(results, shapeResult{
				FName:  fmt.Sprintf("%s#case<%d>", base, mc),
				Detail: detail,
				Bucket: bucket,
			})
		}
		for _, ex := range bij.Extra {
			results = append(results, shapeResult{
				FName:  ex.FName,
				Detail: fmt.Sprintf("Atlas #Mode case %d absent from client dispatch", ex.Case),
				Bucket: "extra-mode",
			})
		}
	}

	return writeReport(opts, results, stdout)
}


// writeReport writes the deterministic markdown report and prints the roll-up
// counts to stdout. Results are bucketed by verdict and sorted by FName within
// each section. Returns a process exit code.
func writeReport(opts validateOpts, results []shapeResult, stdout io.Writer) int {
	var verified, divergent, unverifiable, missing, extra, allowlisted []shapeResult
	for _, r := range results {
		switch r.Bucket {
		case "missing-mode":
			missing = append(missing, r)
			continue
		case "extra-mode":
			extra = append(extra, r)
			continue
		case "allowlisted":
			allowlisted = append(allowlisted, r)
			continue
		}
		switch r.Verdict {
		case idasrc.ShapeVerified:
			verified = append(verified, r)
		case idasrc.ShapeDivergent:
			divergent = append(divergent, r)
		default:
			unverifiable = append(unverifiable, r)
		}
	}
	byFName := func(s []shapeResult) {
		// Stable so equal-FName rows keep insertion order (deterministic output).
		sort.SliceStable(s, func(i, j int) bool { return s[i].FName < s[j].FName })
	}
	byFName(verified)
	byFName(divergent)
	byFName(unverifiable)
	byFName(missing)
	byFName(extra)
	byFName(allowlisted)

	var b strings.Builder
	fmt.Fprintf(&b, "# validate report\n\n")
	fmt.Fprintf(&b, "verified %d / divergent %d / missing-mode %d / extra-mode %d / unverifiable %d / allowlisted %d\n\n",
		len(verified), len(divergent), len(missing), len(extra), len(unverifiable), len(allowlisted))
	writeSection(&b, "verified", verified)
	writeSection(&b, "divergent", divergent)
	writeSection(&b, "missing-mode", missing)
	writeSection(&b, "extra-mode", extra)
	writeSection(&b, "unverifiable", unverifiable)
	writeSection(&b, "allowlisted", allowlisted)

	if dir := filepath.Dir(opts.Report); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "validate: mkdir:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Report, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(stdout, "validate: write report:", err)
		return 3
	}

	fmt.Fprintf(stdout, "verified %d / divergent %d / missing-mode %d / extra-mode %d / unverifiable %d / allowlisted %d\n",
		len(verified), len(divergent), len(missing), len(extra), len(unverifiable), len(allowlisted))
	return 0
}

// writeSection emits one "## <verdict>" section listing each result's FName and
// (when present) detail, in the already-sorted order.
func writeSection(b *strings.Builder, title string, rs []shapeResult) {
	fmt.Fprintf(b, "## %s\n\n", title)
	for _, r := range rs {
		if r.Detail != "" {
			fmt.Fprintf(b, "- %s — %s\n", r.FName, r.Detail)
		} else {
			fmt.Fprintf(b, "- %s\n", r.FName)
		}
	}
	fmt.Fprintln(b)
}
