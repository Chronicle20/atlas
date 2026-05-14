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
