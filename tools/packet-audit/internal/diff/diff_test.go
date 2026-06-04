package diff

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

func TestDiffAlignedExact(t *testing.T) {
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode4},
	}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode1, Comment: "byte"},
		{Op: idasrc.Decode4, Comment: "int32"},
	}}
	rows := Diff(atlas, ida)
	if len(rows) != 2 {
		t.Fatalf("rows=%d", len(rows))
	}
	for _, r := range rows {
		if r.Verdict != VerdictMatch {
			t.Errorf("row %+v: verdict=%v", r, r.Verdict)
		}
	}
}

func TestDiffWidthMismatch(t *testing.T) {
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
	}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode2},
	}}
	rows := Diff(atlas, ida)
	if len(rows) != 1 || rows[0].Verdict != VerdictBlocker {
		t.Fatalf("expected blocker; got %+v", rows)
	}
}

func TestDiffShortAtlas(t *testing.T) {
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
	}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode1}, {Op: idasrc.Decode4},
	}}
	rows := Diff(atlas, ida)
	if len(rows) != 2 || rows[1].Verdict != VerdictBlocker {
		t.Fatalf("expected blocker on missing atlas row; got %+v", rows)
	}
}

func TestDiffOpaqueBufferWidthEquivalence(t *testing.T) {
	// Atlas writes an 8-byte fixed primitive (WriteLong); IDA reads an opaque
	// buffer (DecodeBuffer). The analyzer tracks no byte length on either side,
	// so it cannot prove a mismatch — these are byte-equal opaque-buffer cases.
	atlas := []atlaspacket.Call{{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode8}}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{{Op: idasrc.DecodeBuf}}}
	rows := Diff(atlas, ida)
	if len(rows) != 1 || rows[0].Verdict != VerdictMatch {
		t.Fatalf("expected match for opaque buf == Encode8; got %+v", rows)
	}
}

func TestDiffCompositeRunEqualsWiderRead(t *testing.T) {
	// Atlas writes int16 + int16 (e.g. WriteInt16 + WriteShort(0)); IDA reads a
	// single int32. The two adjacent fixed-width writes sum to the wider read.
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2},
	}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{{Op: idasrc.Decode4}}}
	rows := Diff(atlas, ida)
	if len(rows) != 1 {
		t.Fatalf("composite 2+2 should coalesce into a single row vs Decode4, got %d rows: %+v", len(rows), rows)
	}
	if rows[0].Verdict != VerdictMatch {
		t.Fatalf("composite 2+2 should match Decode4, got verdict %v: %+v", rows[0].Verdict, rows)
	}
}

func TestDiffCompositeRunOvershootDoesNotCoalesce(t *testing.T) {
	// Atlas writes int16 + int32 (sum 6) against IDA Decode4 + Decode4. The first
	// Atlas run would have to sum to 4 to coalesce, but Encode2 alone is 2 and the
	// next call Encode4 overshoots to 6 — never landing exactly on the IDA width.
	// The conservative pre-pass must NOT merge this region, so the genuine width
	// mismatch (Encode2 vs Decode4) still surfaces as a blocker.
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode4},
	}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode4},
		{Op: idasrc.Decode4},
	}}
	rows := Diff(atlas, ida)
	hasBlocker := false
	for _, r := range rows {
		if r.Verdict == VerdictBlocker {
			hasBlocker = true
		}
	}
	if !hasBlocker {
		t.Fatalf("overshooting run (Encode2+Encode4 vs Decode4) must not coalesce; expected a blocker, got %+v", rows)
	}
}

func TestFlattenDropsInactiveGuards(t *testing.T) {
	g, _ := atlaspacket.ParseGuard(`t.MajorVersion() >= 95`)
	calls := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2, Guard: g},
	}
	v95 := Flatten(calls, atlaspacket.GuardContext{Region: "GMS", MajorVersion: 95})
	v83 := Flatten(calls, atlaspacket.GuardContext{Region: "GMS", MajorVersion: 83})
	if len(v95) != 2 || len(v83) != 1 {
		t.Errorf("flatten: v95=%d v83=%d", len(v95), len(v83))
	}
}
