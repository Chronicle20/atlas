package diff

import (
	"strings"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

type Verdict int

const (
	VerdictMatch    Verdict = iota // ✅
	VerdictMinor                   // ⚠️
	VerdictBlocker                 // ❌
	VerdictDeferred                // 🔍
)

func (v Verdict) Symbol() string {
	return [...]string{"✅", "⚠️", "❌", "🔍"}[v]
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
			r.Verdict = VerdictBlocker
			r.Note = "atlas: extra — client never reads this field"
		case atlas[i].Kind == atlaspacket.KindRecurse:
			r.Verdict = VerdictDeferred
			r.Note = "sub-struct: " + atlas[i].RecurseType + " — see _substruct/"
		case atlas[i].Kind == atlaspacket.KindRepeat:
			r.Verdict = VerdictDeferred
			r.Note = "loop body — see follow-up scan"
		case primWidth(atlas[i].Op) != idaWidth(ida.Calls[i].Op):
			r.Verdict = VerdictBlocker
			r.Note = "width mismatch"
		default:
			r.Verdict = VerdictMatch
		}
		rows = append(rows, r)
	}
	return rows
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
	return flattenWithRegistryGuarded(calls, ctx, reg, map[string]bool{})
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
			out = append(out, flattenWithRegistryGuarded(c.Body, ctx, reg, visited)...)
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
				out = append(out, flattenWithRegistryGuarded(sub, ctx, reg, visited)...)
				delete(visited, c.RecurseType)
				continue
			}
		}
		out = append(out, c)
	}
	return out
}
