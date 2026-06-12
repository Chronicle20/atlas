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

type diffShapeOpts struct {
	Baseline     string
	Report       string
	DescentDepth int
}

// diffShapeRun emits a side-by-side hand-vs-live read-list report for every
// DIVERGENT baseline entry, with the divergence position classified. It is a
// pure diagnostic: it loads the baseline, resolves each base live once, extracts
// each entry's shape, and reports — it NEVER mutates the baseline or a verdict.
func diffShapeRun(opts diffShapeOpts, client idasrc.MCPClient, stdout io.Writer) int {
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "diff-shape: load baseline:", err)
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
	type row struct {
		fname string
		diff  shapeDiff
		hand  []idasrc.FieldCall
		live  []idasrc.FieldCall
	}
	var rows []row

	for _, addr := range addrOrder {
		idxs := byAddr[addr]
		dir := entries[idxs[0]].Direction
		f, rerr := idasrc.ResolveLive(ctx, client, addr, dir, idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil {
			continue // diagnostic skips unresolvable bases (validate reports them unverifiable)
		}
		for _, i := range idxs {
			e := entries[i]
			live := idasrc.ExtractShape(f, e.Dispatch)
			verdict, _ := idasrc.ValidateShape(e.HandCalls, live)
			if verdict != idasrc.ShapeDivergent {
				continue
			}
			rows = append(rows, row{fname: e.FName, diff: classifyDiff(e.HandCalls, live), hand: e.HandCalls, live: live})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].fname < rows[j].fname })

	var b strings.Builder
	fmt.Fprintf(&b, "# diff-shape report\n\n%d divergent entries\n\n", len(rows))
	for _, r := range rows {
		fmt.Fprintf(&b, "## %s — %s (delta %+d, prefix %d, suffix %d)\n",
			r.fname, r.diff.position, r.diff.delta, r.diff.prefix, r.diff.suffix)
		fmt.Fprintf(&b, "- hand: %s\n", opsLine(r.hand))
		fmt.Fprintf(&b, "- live: %s\n\n", opsLine(r.live))
	}

	if dir := filepath.Dir(opts.Report); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "diff-shape: mkdir:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Report, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(stdout, "diff-shape: write report:", err)
		return 3
	}
	fmt.Fprintf(stdout, "diff-shape: %d divergent entries\n", len(rows))
	return 0
}

func opsLine(cs []idasrc.FieldCall) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = c.Op.RawOp()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// shapeDiff describes where a hand read-list and a live read-list diverge.
// position is "none" (identical), "leading", "trailing", or "interior" depending
// on where the shared prefix/suffix leaves the differing span. delta is
// len(live)-len(hand). prefix/suffix are the matched-on-both lengths.
type shapeDiff struct {
	position string
	delta    int
	prefix   int
	suffix   int
}

// classifyDiff locates the divergence between two read lists via longest common
// prefix + longest common suffix (by exact op identity). Diagnostic only — it
// never affects a verdict. A list whose shorter side is fully covered by the
// shared prefix+suffix yields "leading"/"trailing" by which side the extra reads
// sit on; otherwise "interior".
func classifyDiff(hand, live []idasrc.FieldCall) shapeDiff {
	d := shapeDiff{delta: len(live) - len(hand)}
	if eqOps(hand, live) {
		d.position = "none"
		return d
	}
	n := min2(len(hand), len(live))
	p := 0
	for p < n && hand[p].Op == live[p].Op {
		p++
	}
	s := 0
	for s < n-p && hand[len(hand)-1-s].Op == live[len(live)-1-s].Op {
		s++
	}
	d.prefix, d.suffix = p, s
	switch {
	case p == 0 && s > 0:
		// only a shared suffix — the extra reads are at the front.
		d.position = "leading"
	case s == 0 && p > 0:
		// only a shared prefix — the extra reads are at the back.
		d.position = "trailing"
	default:
		// shared on both edges (diff sandwiched), or no shared edge at all.
		d.position = "interior"
	}
	return d
}

func eqOps(a, b []idasrc.FieldCall) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Op != b[i].Op {
			return false
		}
	}
	return true
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}
