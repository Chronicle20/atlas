package diff

import (
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

type Verdict int

const (
	VerdictMatch      Verdict = iota // ✅
	VerdictMinor                     // ⚠️
	VerdictBlocker                   // ❌
	VerdictDeferred                  // 🔍
	VerdictUnresolved                // 🚫 — IDA read-order unknown (export gap)
)

func (v Verdict) Symbol() string {
	return [...]string{"✅", "⚠️", "❌", "🔍", "🚫"}[v]
}

// Severity returns a priority rank for verdict aggregation — higher rank = worse /
// more-actionable. The rank is intentionally DECOUPLED from the iota ordinal so
// that appending new verdicts (e.g. VerdictUnresolved) never accidentally masks a
// real Blocker.
//
// Ranking (highest first):
//
//	Blocker(❌) > Minor(⚠️) > Unresolved(🚫) > Deferred(🔍) > Match(✅)
func (v Verdict) Severity() int {
	switch v {
	case VerdictBlocker:
		return 4
	case VerdictMinor:
		return 3
	case VerdictUnresolved:
		return 2
	case VerdictDeferred:
		return 1
	default: // VerdictMatch (and any future low-priority verdict)
		return 0
	}
}

// WorstVerdict returns the single most-actionable verdict from rows, using the
// Severity ranking rather than ordinal comparison. An empty slice returns
// VerdictMatch.
func WorstVerdict(rows []Row) Verdict {
	w := VerdictMatch
	for _, r := range rows {
		if r.Verdict.Severity() > w.Severity() {
			w = r.Verdict
		}
	}
	return w
}

type Row struct {
	Index      int
	AtlasOp    atlaspacket.Primitive
	AtlasKind  atlaspacket.Kind
	IDAOp      idasrc.Primitive
	IDAComment string
	Verdict    Verdict
	Note       string
}

func Diff(atlas []atlaspacket.Call, ida idasrc.Fields) []Row {
	atlas = expandRepeatRuns(atlas, ida)
	atlas = absorbBufferGroups(atlas, ida)
	atlas = coalesceAtlas(atlas, ida)
	var rows []Row
	n := max(len(atlas), len(ida.Calls))
	for i := 0; i < n; i++ {
		var r Row
		r.Index = i
		if i < len(atlas) {
			r.AtlasOp = atlas[i].Op
			r.AtlasKind = atlas[i].Kind
		}
		if i < len(ida.Calls) {
			r.IDAOp = ida.Calls[i].Op
			r.IDAComment = ida.Calls[i].Comment
		}
		switch {
		case i >= len(atlas):
			// IDA's loop-body calls (guard "loop X") are optional at runtime:
			// they only fire when the just-read count is > 0. Atlas correctly
			// omits the loop body when its count is always 0 (e.g. balloons).
			// Downgrade these trailing entries from blocker to minor.
			if i < len(ida.Calls) && strings.HasPrefix(ida.Calls[i].Guard, "loop ") {
				r.Verdict = VerdictMinor
				r.Note = "loop body — atlas emits zero iterations (count==0)"
			} else {
				r.Verdict = VerdictBlocker
				r.Note = "atlas: short — missing trailing field"
			}
		case i >= len(ida.Calls):
			// Trailing opaque buffer absorbs the rest: when the client's LAST read
			// is a DecodeBuf (e.g. PARTYDATA::Decode reads a fixed 298/378-byte
			// block via memcpy), it consumes the remainder of the packet — so any
			// trailing Atlas fields ARE that buffer's content, not an over-write.
			// The audit cannot verify the buffer's exact length (opaque), consistent
			// with the existing EncodeBuf↔* tolerance, so these are matched.
			if len(ida.Calls) > 0 && ida.Calls[len(ida.Calls)-1].Op == idasrc.DecodeBuf {
				r.Verdict = VerdictMatch
				r.Note = "absorbed by trailing opaque buffer"
			} else if i == len(ida.Calls) && i == len(atlas)-1 &&
				atlas[i].Kind == atlaspacket.KindWrite && primWidth(atlas[i].Op) == 1 {
				// A single trailing 1-byte primitive write that no client version
				// reads is harmless padding (a flag byte like MovementAffectingStat=0
				// that the client never decodes — it simply stops reading). This
				// cannot desync the client (nothing follows it), so it is a Minor
				// over-write, not a Blocker. Narrowly scoped to ONE trailing byte so
				// genuine multi-field truncations still surface as blockers.
				r.Verdict = VerdictMinor
				r.Note = "atlas: trailing padding byte — client stops reading (harmless over-write)"
			} else {
				r.Verdict = VerdictBlocker
				r.Note = "atlas: extra — client never reads this field"
			}
		case ida.Calls[i].Op == idasrc.Unresolved:
			r.Verdict = VerdictUnresolved
			r.Note = "IDA read-order unresolved: " + ida.Calls[i].Comment
		case atlas[i].Kind == atlaspacket.KindRecurse && atlas[i].Opaque:
			// Pass-3 opaque boundary: registered type with no encode method whose
			// layout could not be synthesized. STABLE deferred row keyed on
			// "opaque type" so the docs opaque-type registry can curate it.
			r.Verdict = VerdictDeferred
			r.Note = "opaque type: " + atlas[i].RecurseType + " — register boundary (see opaque registry)"
		case atlas[i].Kind == atlaspacket.KindRecurse:
			r.Verdict = VerdictDeferred
			r.Note = "sub-struct: " + atlas[i].RecurseType + " — see _substruct/"
		case atlas[i].Kind == atlaspacket.KindRepeat:
			r.Verdict = VerdictDeferred
			r.Note = "loop body — see follow-up scan"
		case !widthEquivalent(atlas[i].Op, ida.Calls[i]):
			r.Verdict = VerdictBlocker
			r.Note = "width mismatch"
		default:
			r.Verdict = VerdictMatch
		}
		rows = append(rows, r)
	}
	return rows
}

// expandRepeatRuns replicates an inlined loop body (marked with RepeatLen on its
// head) to match the client's N-element run. Atlas writes a fixed-count array via
// `for range` (e.g. guild TitleChange's 5 ranks, written once and flattened to one
// iteration); the client reads N copies. The pre-pass greedily counts how many
// consecutive L-field client blocks width-match the body and emits the Atlas body
// that many times. Only fires when at least one full iteration matches, so a body
// that doesn't align is left untouched (no manufactured match). Walks both sides
// with independent indices, assuming the pre-loop fields are 1:1 (they are for
// these packets).
func expandRepeatRuns(atlas []atlaspacket.Call, ida idasrc.Fields) []atlaspacket.Call {
	has := false
	for _, a := range atlas {
		if a.RepeatLen > 0 {
			has = true
			break
		}
	}
	if !has {
		return atlas
	}
	out := make([]atlaspacket.Call, 0, len(atlas))
	ai, ii := 0, 0
	for ai < len(atlas) {
		a := atlas[ai]
		L := a.RepeatLen
		// Only a TRAILING loop body (nothing follows it on the Atlas side) is
		// safe to expand: the client's remaining reads are then all iterations of
		// it, so greedy matching cannot over-consume a distinct following field.
		if L > 0 && ai+L == len(atlas) && ii+L <= len(ida.Calls) {
			body := atlas[ai : ai+L]
			k := 0
			for ii+(k+1)*L <= len(ida.Calls) && bodyMatchesClient(body, ida.Calls[ii+k*L:ii+(k+1)*L]) {
				k++
			}
			if k >= 1 {
				for r := 0; r < k; r++ {
					for _, b := range body {
						b.RepeatLen = 0
						out = append(out, b)
					}
				}
				ai += L
				ii += k * L
				continue
			}
		}
		out = append(out, a)
		ai++
		if ii < len(ida.Calls) {
			ii++
		}
	}
	return out
}

// bodyMatchesClient reports whether each Atlas loop-body field width-matches the
// corresponding client read in a candidate iteration.
func bodyMatchesClient(body []atlaspacket.Call, client []idasrc.FieldCall) bool {
	if len(body) != len(client) {
		return false
	}
	for i := range body {
		if body[i].Kind != atlaspacket.KindWrite || !widthEquivalent(body[i].Op, client[i]) {
			return false
		}
	}
	return true
}

// absorbBufferGroups collapses an expanded sub-struct field group (marked by
// GroupLen on its head, set by FlattenWithRegistry) into a single opaque buffer
// when the client reads that whole struct as one fixed DecodeBuf. The client's
// GW_Friend/GW_ItemSlot/avatar structs are read as a single fixed-size buffer,
// while Atlas serializes them field-by-field; without this collapse the trailing
// group fields show as false "atlas extra". It walks both sides with independent
// indices and only fires when the client read at the group's position is a
// DecodeBuf — so the field-by-field client convention (where the group should
// stay expanded) is left untouched, and no match is manufactured.
func absorbBufferGroups(atlas []atlaspacket.Call, ida idasrc.Fields) []atlaspacket.Call {
	hasGroup := false
	for _, a := range atlas {
		if a.GroupLen > 1 {
			hasGroup = true
			break
		}
	}
	if !hasGroup {
		return atlas
	}
	out := make([]atlaspacket.Call, 0, len(atlas))
	ai, ii := 0, 0
	for ai < len(atlas) {
		a := atlas[ai]
		if a.GroupLen > 1 && ai+a.GroupLen <= len(atlas) &&
			ii < len(ida.Calls) && ida.Calls[ii].Op == idasrc.DecodeBuf {
			merged := a
			merged.Kind = atlaspacket.KindWrite
			merged.Op = atlaspacket.EncodeBuf
			merged.RecurseType = ""
			merged.GroupLen = 0
			out = append(out, merged)
			ai += a.GroupLen
			ii++
			continue
		}
		out = append(out, a)
		ai++
		if ii < len(ida.Calls) {
			ii++
		}
	}
	return out
}

// coalesceAtlas is a conservative pre-pass that merges runs of adjacent
// fixed-width Atlas writes whose summed byte width exactly equals the next IDA
// fixed-width read — the composite-run equivalence class (task-080 §4.7), e.g.
// WriteInt16 + WriteShort(0) ≡ Decode4, or WriteInt16 + WriteShort + WriteInt
// laid out as an opaque point read.
//
// It is deliberately narrow to avoid false matches:
//   - It only ever merges KindWrite calls with a positive fixed width (1/2/4/8);
//     a KindRecurse/KindRepeat or an opaque/string op (width <= 0) immediately
//     stops the run.
//   - It merges only when the running Atlas sum lands EXACTLY on the current
//     IDA read's fixed width, AND the run spans 2+ Atlas calls collapsing into a
//     single IDA call (so the IDA side has strictly fewer calls at that point).
//     If the very first Atlas call already equals the IDA width, nothing is
//     merged — the existing per-index compare handles the 1:1 case.
//   - On any overshoot it leaves that region untouched, so a genuine width
//     mismatch still surfaces as a blocker.
//
// The synthesized merged call carries the IDA-side width and inherits the run's
// first call (line/guard) so downstream rows stay attributable.
func coalesceAtlas(atlas []atlaspacket.Call, ida idasrc.Fields) []atlaspacket.Call {
	out := make([]atlaspacket.Call, 0, len(atlas))
	ai, ii := 0, 0
	for ai < len(atlas) {
		a := atlas[ai]
		// Only attempt to coalesce a fixed-width write against a fixed-width IDA read.
		aw := primWidth(a.Op)
		if a.Kind != atlaspacket.KindWrite || aw <= 0 || ii >= len(ida.Calls) {
			out = append(out, a)
			ai++
			if ii < len(ida.Calls) {
				ii++
			}
			continue
		}
		iw := idaWidth(ida.Calls[ii].Op)
		if iw <= 0 || aw >= iw {
			// IDA side is opaque/string, or this single write already meets/exceeds
			// the read — let the per-index compare (incl. widthEquivalent) handle it.
			out = append(out, a)
			ai++
			ii++
			continue
		}
		// Try to grow a run of adjacent fixed-width writes that sums to iw.
		sum := aw
		j := ai + 1
		for j < len(atlas) && sum < iw {
			nw := primWidth(atlas[j].Op)
			if atlas[j].Kind != atlaspacket.KindWrite || nw <= 0 {
				break
			}
			sum += nw
			j++
		}
		if sum == iw && j-ai >= 2 {
			merged := a // inherit line/guard from the run's first call
			merged.Op = primFromWidth(iw)
			out = append(out, merged)
			ai = j
			ii++
			continue
		}
		// No exact composite; leave this call as-is for the normal compare.
		out = append(out, a)
		ai++
		ii++
	}
	return out
}

// primFromWidth maps a fixed byte width back to the Atlas primitive of that
// width. Only the fixed widths produced by primWidth (1/2/4/8) are valid here.
func primFromWidth(w int) atlaspacket.Primitive {
	switch w {
	case 1:
		return atlaspacket.Encode1
	case 2:
		return atlaspacket.Encode2
	case 4:
		return atlaspacket.Encode4
	case 8:
		return atlaspacket.Encode8
	}
	return atlaspacket.EncodeBuf
}

// widthEquivalent reports whether an Atlas write and an IDA read occupy the
// same number of wire bytes even when their op-labels differ — the opaque-buffer
// / width-label equivalence class (task-080 §4.7).
//
// The analyzer tracks no byte length for either an Atlas EncodeBuf or an IDA
// DecodeBuf (both map to a sentinel "opaque" width). Cases from the audit —
// WriteByteArray(N) ≡ DecodeBuf(N), WriteLong ≡ EncodeBuffer(8) / 8-byte buf,
// WriteInt64 point ≡ EncodeBuffer(&pt,8) — all pit a fixed-width primitive on
// one side against an opaque buffer on the other. Because no declared length is
// available, the differ cannot prove a mismatch, so it treats a fixed-width
// primitive (1/2/4/8) as equivalent to an opaque buffer rather than flagging a
// false-positive width blocker. Two equal fixed widths still match exactly;
// genuinely different fixed widths (e.g. byte vs int16) still mismatch.
func widthEquivalent(a atlaspacket.Primitive, ida idasrc.FieldCall) bool {
	aw := primWidth(a)
	iw := idaWidth(ida.Op)
	if aw == iw {
		return true
	}
	// A fixed-width Atlas primitive vs an IDA opaque buffer (DecodeBuf, width -2):
	// the buffer's length is not captured, so accept it as the same field.
	if aw > 0 && ida.Op == idasrc.DecodeBuf {
		return true
	}
	// Symmetric direction: an Atlas opaque buffer (EncodeBuf, width -2) vs a
	// fixed-width IDA read. Same rationale — no declared Atlas buffer length.
	if a == atlaspacket.EncodeBuf && iw > 0 {
		return true
	}
	return false
}

func primWidth(p atlaspacket.Primitive) int {
	switch p {
	case atlaspacket.Encode1:
		return 1
	case atlaspacket.Encode2:
		return 2
	case atlaspacket.Encode4:
		return 4
	case atlaspacket.Encode8:
		return 8
	case atlaspacket.EncodeStr:
		return -1
	case atlaspacket.EncodeBuf:
		return -2
	}
	return 0
}

func idaWidth(p idasrc.Primitive) int {
	switch p {
	case idasrc.Decode1:
		return 1
	case idasrc.Decode2:
		return 2
	case idasrc.Decode4:
		return 4
	case idasrc.Decode8:
		return 8
	case idasrc.DecodeStr:
		return -1
	case idasrc.DecodeBuf:
		return -2
	}
	return 0
}

// Flatten resolves an atlas call list against a GuardContext: drops guarded
// calls whose guard evaluates false. Inlines KindRepeat bodies so that loop
// bodies appear in the flattened sequence, matching the IDA export's
// convention of emitting one entry per loop-body field with `guard: "loop X"`.
func Flatten(calls []atlaspacket.Call, ctx atlaspacket.GuardContext) []atlaspacket.Call {
	return FlattenWithRegistry(calls, ctx, nil)
}

// FlattenWithRegistry is like Flatten but also inlines KindRecurse calls by
// looking up the sub-struct's pre-analyzed Call list in reg. When reg is nil
// or a type is unknown, KindRecurse entries pass through unchanged (legacy path).
// Cycle detection prevents infinite recursion when a type transitively refers to itself.
func FlattenWithRegistry(calls []atlaspacket.Call, ctx atlaspacket.GuardContext, reg *atlaspacket.TypeRegistry) []atlaspacket.Call {
	return flattenWithRegistryGuarded(coalesceAdjacentRepeats(calls), ctx, reg, map[string]bool{})
}

// coalesceAdjacentRepeats merges consecutive KindRepeat loops with identical body
// shapes into one. The column-major block-writers (party.WritePartyData) encode
// each field column as TWO adjacent loops — `for member { WriteX }` then
// `for pad { WriteX(0) }` — that together fill one fixed array; the client reads
// each column as a single field. Without merging, the pad loop is an extra
// flattened entry that shifts every later column by one. Only ADJACENT loops with
// the same flattened body are merged, so genuinely distinct loops are untouched.
func coalesceAdjacentRepeats(calls []atlaspacket.Call) []atlaspacket.Call {
	out := make([]atlaspacket.Call, 0, len(calls))
	for _, c := range calls {
		if len(c.Body) > 0 {
			c.Body = coalesceAdjacentRepeats(c.Body)
		}
		if c.Kind == atlaspacket.KindRepeat && len(out) > 0 {
			prev := out[len(out)-1]
			if prev.Kind == atlaspacket.KindRepeat && repeatBodyEqual(prev.Body, c.Body) {
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

// repeatBodyEqual reports whether two loop bodies have the same flattened shape
// (length + per-call Kind/Op/RecurseType), ignoring guards and field names.
func repeatBodyEqual(a, b []atlaspacket.Call) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind || a[i].Op != b[i].Op || a[i].RecurseType != b[i].RecurseType {
			return false
		}
	}
	return true
}

// flattenWithRegistryGuarded is the internal recursion helper for FlattenWithRegistry.
// The visited set tracks types currently on the recursion stack so self-referential
// KindRecurse chains (e.g. Movement → Element → Movement via field-type resolution)
// don't infinite-loop. Marks are added on entry and removed on exit so DAG re-visits
// across separate branches still expand correctly.
func flattenWithRegistryGuarded(calls []atlaspacket.Call, ctx atlaspacket.GuardContext, reg *atlaspacket.TypeRegistry, visited map[string]bool) []atlaspacket.Call {
	var out []atlaspacket.Call
	for _, c := range calls {
		if c.Guard != nil && !c.Guard.Eval(ctx) {
			continue
		}
		if c.Kind == atlaspacket.KindRepeat {
			body := flattenWithRegistryGuarded(c.Body, ctx, reg, visited)
			// Mark the loop-body head so the diff can replicate it to match a
			// client N-element run (fixed-count loop expansion).
			if len(body) > 0 {
				body[0].RepeatLen = len(body)
			}
			out = append(out, body...)
			continue
		}
		if c.Kind == atlaspacket.KindRecurse && reg != nil {
			if visited[c.RecurseType] {
				// Cycle detected — emit the KindRecurse call unchanged so the
				// diff engine surfaces it as a deferred entry rather than looping.
				out = append(out, c)
				continue
			}
			if sub, ok := reg.Calls(c.RecurseType); ok {
				visited[c.RecurseType] = true
				flat := flattenWithRegistryGuarded(sub, ctx, reg, visited)
				delete(visited, c.RecurseType)
				// Mark this sub-struct's flattened span so a client-side opaque
				// DecodeBuf can absorb the whole group (see absorbBufferGroups).
				// Set on the group head; the OUTERMOST recurse wins because the
				// parent expansion runs after this returns.
				if len(flat) > 1 {
					flat[0].GroupLen = len(flat)
				}
				out = append(out, flat...)
				continue
			}
			// Unresolved recurse target. If the registry knows it as a Pass-3
			// opaque boundary (no encode method, layout not decomposable), tag the
			// passed-through call so the diff engine emits a STABLE "opaque" row
			// the docs opaque-type registry can key on, rather than a generic
			// unresolved-recurse row.
			if reg.IsOpaque(c.RecurseType) {
				c.Opaque = true
			}
		}
		out = append(out, c)
	}
	return out
}
