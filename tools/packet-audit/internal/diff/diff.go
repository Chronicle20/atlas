package diff

import (
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
			r.Verdict = VerdictBlocker
			r.Note = "atlas: short — missing trailing field"
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
// calls whose guard evaluates false.
func Flatten(calls []atlaspacket.Call, ctx atlaspacket.GuardContext) []atlaspacket.Call {
	var out []atlaspacket.Call
	for _, c := range calls {
		if c.Guard != nil && !c.Guard.Eval(ctx) {
			continue
		}
		out = append(out, c)
	}
	return out
}
