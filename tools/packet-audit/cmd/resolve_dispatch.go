package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// resolveDispatchOpts is the injectable configuration for the resolve-dispatch
// driver. Unlike infer (read-only), this command MUTATES the baseline in place
// for high-confidence picks. Deterministic for identical (opts, client).
type resolveDispatchOpts struct {
	Baseline      string  // baseline export JSON (mutated in place for high-confidence picks)
	Worklist      string  // markdown confirmation worklist output path
	MinConfidence float64 // auto-accept threshold (default 0.6)
	DescentDepth  int
}

// worklistItem is one low-confidence pick the agent must confirm in IDA.
type worklistItem struct {
	FName      string            `json:"fname"`
	Address    string            `json:"address"`
	Proposed   []idasrc.Selector `json:"proposed"`
	Confidence float64           `json:"confidence"`
	Candidates []int64           `json:"candidates,omitempty"`
}

// resolveDispatchRun infers per-base dispatch selectors for #Mode entries,
// auto-accepts high-confidence picks (writing them into the baseline), and emits
// a confirmation worklist (markdown + sibling .json) of the low-confidence picks
// for the agent to resolve in IDA. A base decompile soft-fail marks that base's
// entries undecompilable and the run continues. Returns a process exit code.
func resolveDispatchRun(opts resolveDispatchOpts, client idasrc.MCPClient, stdout io.Writer) int {
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "resolve-dispatch: load baseline:", err)
		return 3
	}
	entries := src.Entries()

	byAddr := map[string][]int{}
	var addrOrder []string
	for i := range entries {
		a := entries[i].Address
		if _, ok := byAddr[a]; !ok {
			addrOrder = append(addrOrder, a)
		}
		byAddr[a] = append(byAddr[a], i)
	}
	sort.Strings(addrOrder)

	ctx := context.Background()
	accepted := map[string]idasrc.DispatchUpdate{}
	var worklist []worklistItem
	undecompilable := 0

	for _, addr := range addrOrder {
		idxs := byAddr[addr]
		// Only #Mode entries need a selector; flat entries are validated directly.
		var modeIdxs []int
		for _, i := range idxs {
			if strings.Contains(entries[i].FName, "#") {
				modeIdxs = append(modeIdxs, i)
			}
		}
		if len(modeIdxs) == 0 {
			continue
		}

		dir := entries[modeIdxs[0]].Direction
		f, rerr := idasrc.ResolveLive(ctx, client, addr, dir, idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil {
			undecompilable += len(modeIdxs)
			continue
		}

		shapes := make([]idasrc.EntryShape, len(modeIdxs))
		for k, i := range modeIdxs {
			shapes[k] = idasrc.EntryShape{FName: entries[i].FName, Hand: entries[i].HandCalls}
		}
		for _, a := range idasrc.InferDispatchJoint(f, shapes) {
			if len(a.Dispatch) > 0 && a.Confidence >= opts.MinConfidence && len(a.Candidates) < 2 {
				accepted[a.FName] = idasrc.DispatchUpdate{
					Dispatch: a.Dispatch,
					Note:     fmt.Sprintf("inferred-high-confidence (%.2f) @%s", a.Confidence, addr),
				}
			} else {
				worklist = append(worklist, worklistItem{
					FName:      a.FName,
					Address:    addr,
					Proposed:   a.Dispatch,
					Confidence: a.Confidence,
					Candidates: a.Candidates,
				})
			}
		}
	}

	if len(accepted) > 0 {
		if err := idasrc.WriteDispatch(opts.Baseline, accepted); err != nil {
			fmt.Fprintln(stdout, "resolve-dispatch: write baseline:", err)
			return 3
		}
	}
	if code := writeWorklist(opts, worklist, stdout); code != 0 {
		return code
	}

	// (worklist was already sorted by writeWorklist; the roll-up only needs counts.)
	fmt.Fprintf(stdout, "resolve-dispatch: %d auto-accepted (>=%.2f), %d to confirm, %d undecompilable\n",
		len(accepted), opts.MinConfidence, len(worklist), undecompilable)
	return 0
}

// writeWorklist writes the markdown confirmation worklist and a sibling .json.
// The agent reads the .json, confirms each pick against the IDA decompile at
// Address, and applies the confirmed selector via a follow-up WriteDispatch.
func writeWorklist(opts resolveDispatchOpts, items []worklistItem, stdout io.Writer) int {
	sort.Slice(items, func(i, j int) bool { return items[i].FName < items[j].FName })
	var b strings.Builder
	fmt.Fprintf(&b, "# resolve-dispatch confirmation worklist\n\n%d low-confidence picks to confirm in IDA.\n\n", len(items))
	for _, it := range items {
		fmt.Fprintf(&b, "- `%s` @%s — proposed %v (conf %.2f) candidates %v\n",
			it.FName, it.Address, it.Proposed, it.Confidence, it.Candidates)
	}
	if dir := filepath.Dir(opts.Worklist); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "resolve-dispatch: mkdir:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Worklist, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(stdout, "resolve-dispatch: write worklist:", err)
		return 3
	}
	jsonPath := strings.TrimSuffix(opts.Worklist, filepath.Ext(opts.Worklist)) + ".json"
	jb, _ := json.MarshalIndent(items, "", "  ")
	jb = append(jb, '\n')
	if err := os.WriteFile(jsonPath, jb, 0o644); err != nil {
		fmt.Fprintln(stdout, "resolve-dispatch: write worklist json:", err)
		return 3
	}
	return 0
}
