package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// inferOpts is the injectable configuration for the infer driver. Every value
// affecting the proposal JSON is fixed up front, so re-running with identical
// opts+client produces byte-identical output. The flag wrapper also carries
// Version/IDAURL/IDATimeout (used only to build the client + default paths),
// which the core does not need.
type inferOpts struct {
	Baseline      string  // baseline export JSON (read-only)
	Out           string  // proposal JSON output path
	MinConfidence float64 // threshold for "high confidence" in the roll-up (default 0.6)
	DescentDepth  int
}

// inferProposal is one entry's inferred dispatch proposal. candidates is omitted
// when there is no near-tie (InferDispatch returns nil unless genuinely ambiguous).
type inferProposal struct {
	Dispatch   []idasrc.Selector `json:"dispatch"`
	Confidence float64           `json:"confidence"`
	Candidates []int64           `json:"candidates,omitempty"`
}

// inferProposalFile is the deterministic output document: FName -> proposal.
type inferProposalFile struct {
	Proposals map[string]inferProposal `json:"proposals"`
}

// inferRun proposes a dispatch selector for each baseline entry by decompiling its
// base by address (once per address, cached) via ResolveLive, matching each entry's
// hand-authored reads to the best switch case (InferDispatch). It writes a
// deterministic proposal JSON (FName -> {dispatch, confidence, candidates}) and a
// stdout roll-up (high-confidence / ambiguous / undecompilable counts). It NEVER
// mutates the baseline.
//
// A base decompilation soft-fail (or any other ResolveLive error) marks every entry
// at that address undecompilable (no proposal) and the run CONTINUES — one bad
// address never aborts. Returns a process exit code.
func inferRun(opts inferOpts, client idasrc.MCPClient, stdout io.Writer) int {
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "infer: load baseline:", err)
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

	ctx := context.Background()
	proposals := map[string]inferProposal{}
	var ambiguous []string
	undecompilable := 0

	for _, addr := range addrOrder {
		idxs := byAddr[addr]
		dir := entries[idxs[0]].Direction
		f, rerr := idasrc.ResolveLive(ctx, client, addr, dir, idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil {
			// Soft-fail or any other error: every entry at this address is
			// undecompilable; no proposal; continue (never abort).
			undecompilable += len(idxs)
			continue
		}
		// Joint per-base assignment: each entry at this address maps one-to-one to a
		// DISTINCT switch case, resolving the conflicts/near-ties that independent
		// per-entry inference produces when siblings share a base.
		shapes := make([]idasrc.EntryShape, len(idxs))
		for k, i := range idxs {
			shapes[k] = idasrc.EntryShape{FName: entries[i].FName, Hand: entries[i].HandCalls}
		}
		for _, a := range idasrc.InferDispatchJoint(f, shapes) {
			proposals[a.FName] = inferProposal{Dispatch: a.Dispatch, Confidence: a.Confidence, Candidates: a.Candidates}
			if len(a.Candidates) >= 2 {
				ambiguous = append(ambiguous, a.FName)
			}
		}
	}

	if code := writeProposals(opts, proposals, stdout); code != 0 {
		return code
	}

	// Roll-up: high-confidence vs ambiguous vs undecompilable. "ambiguous" is the
	// genuine near-tie set; the remaining proposals below the threshold (but not
	// ambiguous) still count as not-high-confidence implicitly via the difference.
	high := 0
	for _, p := range proposals {
		if len(p.Candidates) < 2 && p.Confidence >= opts.MinConfidence {
			high++
		}
	}
	sort.Strings(ambiguous)
	fmt.Fprintf(stdout, "infer: %d high-confidence (>=%.2f), %d ambiguous, %d undecompilable\n",
		high, opts.MinConfidence, len(ambiguous), undecompilable)
	for _, fn := range ambiguous {
		fmt.Fprintf(stdout, "  ambiguous: %s\n", fn)
	}
	return 0
}

// writeProposals writes the deterministic proposal JSON. Map key ordering is
// normalized by encoding/json (sorts keys), so the output is byte-stable for a
// given set of proposals. Returns a process exit code.
func writeProposals(opts inferOpts, proposals map[string]inferProposal, stdout io.Writer) int {
	doc := inferProposalFile{Proposals: proposals}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(stdout, "infer: marshal proposals:", err)
		return 3
	}
	b = append(b, '\n')

	if dir := filepath.Dir(opts.Out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "infer: mkdir:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Out, b, 0o644); err != nil {
		fmt.Fprintln(stdout, "infer: write proposals:", err)
		return 3
	}
	return 0
}
