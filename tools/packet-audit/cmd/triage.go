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

// triageOpts is the injectable configuration for the triage driver. Every value
// affecting the worklist is fixed up front, so re-running with identical
// opts+client produces a byte-identical report. The flag wrapper also carries
// Version/IDAURL/IDATimeout (used only to build the client + default paths),
// which the core does not need.
type triageOpts struct {
	Version      string // target version key (used only by the flag wrapper for defaults)
	Baseline     string // baseline export JSON (read-only)
	AuditDir     string // committed audit dir for this version (read-only)
	Report       string // markdown worklist output
	DescentDepth int
}

// triageCategory is the actionable bucket a non-✅ packet lands in.
type triageCategory string

const (
	// catCandidateReal: the handler does NOT branch on an early read (a flat
	// read-order compare is valid) AND at least one non-OK audit Row is a genuine
	// non-width-tolerable op/length mismatch. A candidate real wire bug — work it.
	catCandidateReal triageCategory = "candidate-real-divergence"
	// catPerModeBranch: the client handler BRANCHES on an early read (switch / if
	// on a leave-type byte / loop body). The flat whole-function export read-order
	// is the UNION of all branches; Atlas writes ONE branch — so the flat audit
	// comparison is invalid and must NOT be flagged as a real bug. Verify
	// per-branch (via dispatch).
	catPerModeBranch triageCategory = "per-mode-branch"
	// catRepresentation: NOT conditional AND every non-OK Row is width-tolerable
	// (equal-width or a fixed-vs-buffer pair under FieldEquivalent) — a
	// representation/encoding-choice difference, not a wire bug.
	catRepresentation triageCategory = "representation"
	// catUnverifiable: the base decompile failed, or the faithful read-order is
	// dominated by Unresolved spans — the comparison cannot be trusted.
	catUnverifiable triageCategory = "unverifiable"
)

// triageResult is one classified non-✅ packet outcome.
type triageResult struct {
	FName    string
	Address  string
	Verdict  string // rendered verdict symbol from the worst audit row
	ClientRd []string
	Category triageCategory
	Reason   string
}

// triageRow mirrors the fields the triage classifier consumes from one audit
// `<Packet>.json` Row. AtlasOp/IDAOp are integer primitives (atlaspacket /
// idasrc enum ordinals); a row is non-OK when Verdict ∈ {2,3,4}.
type triageRow struct {
	Index   int    `json:"Index"`
	AtlasOp int    `json:"AtlasOp"`
	IDAOp   int    `json:"IDAOp"`
	Verdict int    `json:"Verdict"`
	Note    string `json:"Note"`
}

// triagePacket groups the rows and the authoritative BranchDepth for one
// non-✅ packet read from its audit `.json`. BranchDepth ≥ 1 means the
// handler branches on an early read; the flat read-order compare is invalid.
type triagePacket struct {
	Rows        []triageRow
	BranchDepth int
}

// triageRun classifies every NON-✅ audited packet into an actionable category
// and writes a deterministic markdown worklist. Read-only: it never mutates the
// baseline or the audit. Returns a process exit code.
func triageRun(opts triageOpts, client idasrc.MCPClient, stdout io.Writer) int {
	// 1. Parse the audit dir → NON-✅ FName set (+ their packets + bad-audit notes).
	pkts, badAudit, err := parseTriagePackets(opts.AuditDir)
	if err != nil {
		fmt.Fprintln(stdout, "triage: read audit dir:", err)
		return 3
	}

	// 2. Load the baseline, index entries by FName.
	src, err := idasrc.NewExportSource(opts.Baseline)
	if err != nil {
		fmt.Fprintln(stdout, "triage: load baseline:", err)
		return 3
	}
	byFName := map[string]idasrc.BaselineEntry{}
	for _, e := range src.Entries() {
		byFName[e.FName] = e
	}

	ctx := context.Background()
	var results []triageResult

	// bad-audit packets cannot be classified (no FName) — surface as unverifiable.
	for _, name := range badAudit {
		results = append(results, triageResult{
			FName:    name,
			Category: catUnverifiable,
			Reason:   "audit .md missing or unparseable; FName could not be derived",
		})
	}

	// Deterministic FName order.
	fnames := make([]string, 0, len(pkts))
	for fn := range pkts {
		fnames = append(fnames, fn)
	}
	sort.Strings(fnames)

	for _, fn := range fnames {
		pkt := pkts[fn]
		entry, ok := byFName[fn]
		if !ok {
			results = append(results, triageResult{
				FName:    fn,
				Category: catUnverifiable,
				Reason:   "not present in baseline",
			})
			continue
		}

		res := triageResult{FName: fn, Address: entry.Address, Verdict: "❌"}

		// 3a. Decompile the raw text (conditionality) AND resolve the faithful
		// read-order (evidence + Unresolved-dominance). A base soft-fail/error →
		// unverifiable.
		text, derr := client.DecompileFunction(ctx, entry.Address)
		if derr != nil {
			res.Category = catUnverifiable
			res.Reason = "decompile failed"
			if !idasrc.IsDecompilationFailed(derr) {
				res.Reason = "decompile failed: " + derr.Error()
			}
			results = append(results, res)
			continue
		}

		f, rerr := idasrc.ResolveLive(ctx, client, entry.Address, entry.Direction,
			idasrc.HarvestOpts{DescentDepth: opts.DescentDepth})
		if rerr != nil {
			res.Category = catUnverifiable
			res.Reason = "base resolve failed"
			if !idasrc.IsDecompilationFailed(rerr) {
				res.Reason = "base resolve failed: " + rerr.Error()
			}
			results = append(results, res)
			continue
		}
		res.ClientRd = readOps(f.Calls)

		// 3b. Unresolved-dominance: if the faithful read-order is mostly Unresolved
		// spans, the flat comparison cannot be trusted.
		if unresolvedDominates(f.Calls) {
			res.Category = catUnverifiable
			res.Reason = "faithful read-order dominated by Unresolved spans"
			results = append(results, res)
			continue
		}

		// 3c-pre1. '#'-entry: the FName contains '#', meaning this is a per-dispatch
		// slice of a switch/dispatch handler (e.g. CUIGuildBBS::OnGuildBBSPacket#BBSThread).
		// The base handler branches; any flat audit compare is invalid regardless of
		// whether the slice's own decompile looks flat. Classify before conditionality.
		if strings.Contains(fn, "#") {
			res.Category = catPerModeBranch
			res.Reason = "per-mode # entry — base handler branches; verify per-dispatch"
			results = append(results, res)
			continue
		}

		// 3c-pre2. Audit BranchDepth: the audit JSON BranchDepth field is the
		// authoritative branch signal (more reliable than the decompile heuristic).
		// BranchDepth ≥ 1 means the handler branches on an early read; the flat
		// read-order compare is invalid regardless of what the decompile looks like.
		if pkt.BranchDepth >= 1 {
			res.Category = catPerModeBranch
			res.Reason = fmt.Sprintf("audit BranchDepth %d — handler branches; flat compare invalid", pkt.BranchDepth)
			results = append(results, res)
			continue
		}

		// 3c-pre3. Repeating-run: if the faithful read-order primitives contain a
		// block that repeats three consecutive times (with L*3 >= 6 reads), the
		// decompiler unrolled a loop/array. No 'for' construct survives for
		// ReadsAreConditional to catch, but the repeated read block is proof of a
		// loop — the flat compare is invalid.
		if idasrc.HasRepeatingRun(callPrimitives(f.Calls)) {
			res.Category = catPerModeBranch
			res.Reason = "repeating read run — loop/array unrolled by decompiler; flat compare invalid"
			results = append(results, res)
			continue
		}

		// 3c-pre4. Empty faithful read-order: no Decode calls were extracted from
		// the decompile. This is an extraction failure, not a divergence.
		if len(f.Calls) == 0 {
			res.Category = catUnverifiable
			res.Reason = "no faithful reads extracted"
			results = append(results, res)
			continue
		}

		// 3c-pre5. Any Unresolved in the faithful read-order: the read-order
		// contains a gap that cannot be compared. The comparison cannot be trusted.
		for _, c := range f.Calls {
			if c.Op == idasrc.Unresolved {
				res.Category = catUnverifiable
				res.Reason = "faithful read contains Unresolved span"
				break
			}
		}
		if res.Category == catUnverifiable {
			results = append(results, res)
			continue
		}

		// 3c. Conditionality — the crux. A handler whose reads branch on an early
		// value cannot be flat-compared; classify per-mode-branch (never a real
		// bug). Catches switch-dispatch, if-on-byte (DropDestroy), and loop bodies.
		if idasrc.ReadsAreConditional(text, entry.Direction) {
			res.Category = catPerModeBranch
			res.Reason = "client handler branches on an early read; flat audit compare invalid — verify per-branch"
			results = append(results, res)
			continue
		}

		// 3d. Flat handler: representation iff EVERY non-OK row is width-tolerable;
		// otherwise candidate-real-divergence.
		if allNonOKTolerable(pkt.Rows) {
			res.Category = catRepresentation
			res.Reason = "flat handler; every divergent field is width-tolerable (fixed↔buffer or equal width)"
		} else {
			res.Category = catCandidateReal
			res.Reason = "flat handler with a non-tolerable op/length mismatch — candidate real wire divergence"
		}
		results = append(results, res)
	}

	return writeTriageReport(opts, results, stdout)
}

// isNonOKVerdict reports whether an audit row verdict is actionable
// (2 blocker, 3 deferred, 4 unresolved).
func isNonOKVerdict(v int) bool { return v == 2 || v == 3 || v == 4 }

// allNonOKTolerable reports whether EVERY non-OK row is width-tolerable under
// FieldEquivalent (mapping the integer AtlasOp/IDAOp ordinals onto idasrc
// Primitives — both enums share the Decode/Encode{1,2,4,8,Str,Buf} ordering).
// A row with no comparable ops on either side (e.g. an atlas-extra trailing
// field with no client op) is NOT tolerable: the client provably never reads it
// in a flat handler, which is exactly a candidate real divergence.
func allNonOKTolerable(rows []triageRow) bool {
	sawNonOK := false
	for _, r := range rows {
		if !isNonOKVerdict(r.Verdict) {
			continue
		}
		sawNonOK = true
		a, aok := primitiveFromOrdinal(r.AtlasOp)
		b, bok := primitiveFromOrdinal(r.IDAOp)
		if !aok || !bok {
			// A missing op on either side means one side has no field here (atlas
			// short / atlas extra). That is a length divergence, not a tolerable
			// representation difference.
			return false
		}
		if !idasrc.FieldEquivalent(a, b) {
			return false
		}
	}
	// Defensive: if somehow no non-OK rows were present, treat as not-tolerable so
	// we never silently downgrade (the packet was non-✅ for a reason).
	return sawNonOK
}

// primitiveFromOrdinal maps an audit integer op ordinal (atlaspacket.Primitive
// for AtlasOp, idasrc.Primitive for IDAOp — both share the
// {1,2,4,8,Str,Buf}=0..5 ordering) to an idasrc.Primitive. Ordinals outside
// 0..5 (notably a sentinel for "no op on this side") return ok=false.
func primitiveFromOrdinal(n int) (idasrc.Primitive, bool) {
	switch n {
	case 0:
		return idasrc.Decode1, true
	case 1:
		return idasrc.Decode2, true
	case 2:
		return idasrc.Decode4, true
	case 3:
		return idasrc.Decode8, true
	case 4:
		return idasrc.DecodeStr, true
	case 5:
		return idasrc.DecodeBuf, true
	}
	return idasrc.Unresolved, false
}

// readOps renders a faithful read-order as a compact op-string slice for the
// worklist evidence (e.g. ["Decode1","Decode4"]). Unresolved entries render as
// "Unresolved".
func readOps(calls []idasrc.FieldCall) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, c.Op.RawOp())
	}
	return out
}

// callPrimitives extracts the Op primitive from each FieldCall for use with
// idasrc.HasRepeatingRun. Unresolved entries are preserved as-is (they
// participate in the block-equality comparison so a repeat of mixed
// concrete+Unresolved blocks is still detectable).
func callPrimitives(calls []idasrc.FieldCall) []idasrc.Primitive {
	out := make([]idasrc.Primitive, len(calls))
	for i, c := range calls {
		out[i] = c.Op
	}
	return out
}

// unresolvedDominates reports whether the faithful read-order is mostly
// Unresolved (strictly more Unresolved entries than concrete reads). An
// all-empty order (no reads at all) is not dominated.
func unresolvedDominates(calls []idasrc.FieldCall) bool {
	if len(calls) == 0 {
		return false
	}
	unres, concrete := 0, 0
	for _, c := range calls {
		if c.Op == idasrc.Unresolved {
			unres++
		} else {
			concrete++
		}
	}
	return unres > concrete
}

// parseTriagePackets scans an audit dir for per-packet `<Packet>.json` files. A
// packet is NON-✅ when any Row has Verdict ∈ {2,3,4}. For each such packet it
// reads the sibling `<Packet>.md` header to derive the FName, mapping FName ->
// its triagePacket (rows + BranchDepth). Returns the NON-✅ packets and the list
// of packets whose `.md` was missing/unparseable (bad-audit, derived from the
// json base name).
func parseTriagePackets(auditDir string) (pkts map[string]triagePacket, badAudit []string, err error) {
	jsons, err := filepath.Glob(filepath.Join(auditDir, "*.json"))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(jsons)
	pkts = map[string]triagePacket{}
	for _, jp := range jsons {
		b, rerr := os.ReadFile(jp)
		if rerr != nil {
			return nil, nil, rerr
		}
		var doc struct {
			BranchDepth int         `json:"BranchDepth"`
			Rows        []triageRow `json:"Rows"`
		}
		if jerr := json.Unmarshal(b, &doc); jerr != nil {
			badAudit = append(badAudit, strings.TrimSuffix(filepath.Base(jp), ".json"))
			continue
		}
		isNonOK := false
		for _, r := range doc.Rows {
			if isNonOKVerdict(r.Verdict) {
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
		pkts[fname] = triagePacket{Rows: doc.Rows, BranchDepth: doc.BranchDepth}
	}
	return pkts, badAudit, nil
}

// writeTriageReport writes the deterministic markdown worklist (entries grouped
// by category, sorted by FName within each) and prints the stdout roll-up.
// Returns a process exit code.
func writeTriageReport(opts triageOpts, results []triageResult, stdout io.Writer) int {
	order := []triageCategory{catCandidateReal, catPerModeBranch, catRepresentation, catUnverifiable}
	buckets := map[triageCategory][]triageResult{}
	for _, r := range results {
		buckets[r.Category] = append(buckets[r.Category], r)
	}
	for _, c := range order {
		s := buckets[c]
		sort.Slice(s, func(i, j int) bool { return s[i].FName < s[j].FName })
		buckets[c] = s
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# triage worklist\n\n")
	fmt.Fprintf(&b, "candidate-real-divergence %d / per-mode-branch %d / representation %d / unverifiable %d\n\n",
		len(buckets[catCandidateReal]), len(buckets[catPerModeBranch]),
		len(buckets[catRepresentation]), len(buckets[catUnverifiable]))
	for _, c := range order {
		fmt.Fprintf(&b, "## %s\n\n", c)
		for _, r := range buckets[c] {
			ops := strings.Join(r.ClientRd, ", ")
			addr := r.Address
			if addr == "" {
				addr = "?"
			}
			verdict := r.Verdict
			if verdict == "" {
				verdict = "❌"
			}
			fmt.Fprintf(&b, "- %s (FName `%s` @%s) — verdict %s — client-read [%s] — note: %s\n",
				r.FName, r.FName, addr, verdict, ops, r.Reason)
		}
		fmt.Fprintln(&b)
	}

	if dir := filepath.Dir(opts.Report); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stdout, "triage: mkdir report:", err)
			return 3
		}
	}
	if err := os.WriteFile(opts.Report, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(stdout, "triage: write report:", err)
		return 3
	}

	fmt.Fprintf(stdout, "triage: candidate-real-divergence %d / per-mode-branch %d / representation %d / unverifiable %d\n",
		len(buckets[catCandidateReal]), len(buckets[catPerModeBranch]),
		len(buckets[catRepresentation]), len(buckets[catUnverifiable]))
	return 0
}
