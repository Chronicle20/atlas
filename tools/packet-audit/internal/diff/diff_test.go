package diff

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
	"github.com/Chronicle20/atlas/tools/packet-audit/internal/idasrc"
)

// buildFixtureRegistry materializes one atlaspacket testdata fixture (.go.txt)
// into a temp package directory as a real .go file and runs NewTypeRegistry over
// it. This exercises the production registry path (Pass 1/2/3) without coupling
// to libs/atlas-packet, so a fixture can model a struct shape in isolation.
func buildFixtureRegistry(t *testing.T, fixture string) *atlaspacket.TypeRegistry {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(thisFile), "..", "atlaspacket", fixture)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixture, err)
	}
	dir := t.TempDir()
	base := strings.TrimSuffix(filepath.Base(fixture), ".txt") // foo.go.txt -> foo.go
	dst := filepath.Join(dir, base)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture copy: %v", err)
	}
	reg, err := atlaspacket.NewTypeRegistry(dir)
	if err != nil {
		t.Fatalf("NewTypeRegistry: %v", err)
	}
	return reg
}

func countOp(calls []atlaspacket.Call, op atlaspacket.Primitive) int {
	n := 0
	for _, c := range calls {
		if c.Kind == atlaspacket.KindWrite && c.Op == op {
			n++
		}
	}
	return n
}

// TestRegistryDescendsDecomposableType pins Pass-3: a sub-struct field whose
// type has NO encode method but decomposes into flat known primitives (two
// int32) must be INLINED when its parent's recurse marker is flattened —
// surfacing 2 Encode4 writes rather than leaving an unresolved deferred recurse.
func TestRegistryDescendsDecomposableType(t *testing.T) {
	reg := buildFixtureRegistry(t, "testdata/substruct_no_encode.go.txt")
	calls, ok := reg.Calls("Outer")
	if !ok {
		t.Fatal("Outer not registered (expected Encode-derived calls)")
	}
	flat := FlattenWithRegistry(calls, atlaspacket.GuardContext{Region: "GMS", MajorVersion: 95}, reg)
	if got := countOp(flat, atlaspacket.Encode4); got != 2 {
		t.Fatalf("expected 2 inlined Encode4 from sub-struct, got %d (flat=%+v)", got, flat)
	}
}

// TestRegistryFlagsOpaqueFieldAndDefers pins the Opaque path (Pass-3's negative
// branch): a method-less struct (Opaq) carrying an unmappable field (float64)
// must NOT be synthesized — it stays Calls==nil, is flagged Opaque, and a recurse
// into it surfaces as the STABLE "opaque type:" deferred row rather than being
// silently inlined or passed through as a generic unresolved recurse.
func TestRegistryFlagsOpaqueFieldAndDefers(t *testing.T) {
	reg := buildFixtureRegistry(t, "testdata/substruct_no_encode.go.txt")

	// (a) The float64-bearing struct is the register boundary, not synthesized.
	if !reg.IsOpaque("Opaq") {
		t.Fatal("expected Opaq to be flagged Opaque (float64 field is unmappable)")
	}
	if _, ok := reg.Calls("Opaq"); ok {
		t.Fatal("Opaq must NOT have synthesized Calls — it should be left opaque")
	}

	// (b) Diffing a recurse into Opaq yields a VerdictDeferred row whose note
	// names the opaque type.
	calls, ok := reg.Calls("OuterOpaque")
	if !ok {
		t.Fatal("OuterOpaque not registered (expected Encode-derived calls)")
	}
	flat := FlattenWithRegistry(calls, atlaspacket.GuardContext{Region: "GMS", MajorVersion: 95}, reg)
	rows := Diff(flat, idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode1},
		{Op: idasrc.DecodeBuf},
	}})
	var found bool
	for _, r := range rows {
		if r.Verdict == VerdictDeferred && strings.Contains(r.Note, "opaque type:") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a VerdictDeferred row noting \"opaque type:\"; got rows=%+v", rows)
	}
}

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

func TestDiffUnresolvedRow(t *testing.T) {
	atlas := []atlaspacket.Call{{Op: atlaspacket.Encode4}}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{{Op: idasrc.Unresolved, Comment: "vtable"}}}
	rows := Diff(atlas, ida)
	if rows[0].Verdict != VerdictUnresolved {
		t.Errorf("verdict = %v (%s), want VerdictUnresolved", rows[0].Verdict, rows[0].Verdict.Symbol())
	}
}

func TestVerdictUnresolvedSymbol(t *testing.T) {
	if VerdictUnresolved.Symbol() != "🚫" {
		t.Errorf("symbol = %q, want 🚫", VerdictUnresolved.Symbol())
	}
}

// TestWorstVerdictBlockerBeatsUnresolved verifies that WorstVerdict returns
// VerdictBlocker when rows contain both a VerdictBlocker and a VerdictUnresolved
// (or VerdictDeferred) — ordinal max would incorrectly pick VerdictUnresolved/
// VerdictDeferred since their iota values exceed VerdictBlocker.
func TestWorstVerdictBlockerBeatsUnresolved(t *testing.T) {
	rows := []Row{
		{Verdict: VerdictUnresolved},
		{Verdict: VerdictBlocker},
	}
	got := WorstVerdict(rows)
	if got != VerdictBlocker {
		t.Errorf("WorstVerdict = %v (%s), want VerdictBlocker", got, got.Symbol())
	}
}

func TestWorstVerdictBlockerBeatsDeferred(t *testing.T) {
	rows := []Row{
		{Verdict: VerdictDeferred},
		{Verdict: VerdictBlocker},
		{Verdict: VerdictMatch},
	}
	got := WorstVerdict(rows)
	if got != VerdictBlocker {
		t.Errorf("WorstVerdict = %v (%s), want VerdictBlocker", got, got.Symbol())
	}
}

func TestWorstVerdictMinorBeatsUnresolved(t *testing.T) {
	// Minor (⚠️) is more actionable than Unresolved (🚫) — an Unresolved row is an
	// IDA export gap, not a wire bug, so a Minor finding wins the aggregate.
	rows := []Row{
		{Verdict: VerdictMinor},
		{Verdict: VerdictUnresolved},
	}
	got := WorstVerdict(rows)
	if got != VerdictMinor {
		t.Errorf("WorstVerdict = %v (%s), want VerdictMinor", got, got.Symbol())
	}
}

func TestWorstVerdictEmptyIsMatch(t *testing.T) {
	got := WorstVerdict(nil)
	if got != VerdictMatch {
		t.Errorf("WorstVerdict(nil) = %v, want VerdictMatch", got)
	}
}

// TestAbsorbBufferGroupsCollapsesSubstruct pins the sub-struct buffer-absorb:
// an expanded sub-struct field group (GroupLen on its head) collapses into one
// opaque buffer when the client reads it as a single DecodeBuf — but stays
// expanded when the client reads the fields individually.
func TestAbsorbBufferGroupsCollapsesSubstruct(t *testing.T) {
	// Atlas: [byte, <group of 3: int32,bytes,byte>, byte] (e.g. mode + GW_* + flag)
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode4, GroupLen: 3},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.EncodeBuf},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
	}

	// (a) client reads the group as ONE buffer → all rows match.
	bufClient := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode1, Comment: "mode"},
		{Op: idasrc.DecodeBuf, Comment: "GW_* struct"},
		{Op: idasrc.Decode1, Comment: "flag"},
	}}
	for _, r := range Diff(atlas, bufClient) {
		if r.Verdict != VerdictMatch {
			t.Fatalf("buffer-client: row %d not match: %s %s", r.Index, r.Verdict.Symbol(), r.Note)
		}
	}

	// (b) client reads the fields individually → group stays expanded, all match.
	fieldClient := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode1}, {Op: idasrc.Decode4}, {Op: idasrc.DecodeBuf},
		{Op: idasrc.Decode1}, {Op: idasrc.Decode1},
	}}
	rows := Diff(atlas, fieldClient)
	if len(rows) != 5 {
		t.Fatalf("field-client: expected 5 rows (group stays expanded), got %d", len(rows))
	}
	for _, r := range rows {
		if r.Verdict != VerdictMatch {
			t.Fatalf("field-client: row %d not match: %s %s", r.Index, r.Verdict.Symbol(), r.Note)
		}
	}
}

// TestExpandRepeatRuns pins the fixed-count loop expansion: a trailing Atlas loop
// body (RepeatLen on its head) replicates to match the client's N-element run,
// but a non-trailing loop is left untouched (no over-consumption).
func TestExpandRepeatRuns(t *testing.T) {
	// Trailing loop: Atlas [mode, <string body, RepeatLen=1>] vs client [mode, str×5].
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.EncodeStr, RepeatLen: 1},
	}
	client := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode1}, {Op: idasrc.DecodeStr}, {Op: idasrc.DecodeStr},
		{Op: idasrc.DecodeStr}, {Op: idasrc.DecodeStr}, {Op: idasrc.DecodeStr},
	}}
	rows := Diff(atlas, client)
	if len(rows) != 6 {
		t.Fatalf("expected 6 rows (body replicated ×5), got %d", len(rows))
	}
	for _, r := range rows {
		if r.Verdict != VerdictMatch {
			t.Fatalf("row %d not match: %s %s", r.Index, r.Verdict.Symbol(), r.Note)
		}
	}
}

// TestTrailingPaddingByte pins the narrow trailing-padding rule: a single 1-byte
// primitive write past the client's last read (e.g. OnSetTemporaryStat's unread
// MovementAffectingStat=0) is a Minor over-write, not a Blocker — nothing follows
// it, so it cannot desync the client.
func TestTrailingPaddingByte(t *testing.T) {
	// Atlas [int32, bytes, int16, byte] vs client [int32, bytes, int16].
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode4},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.EncodeBuf},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1},
	}
	client := idasrc.Fields{Calls: []idasrc.FieldCall{
		{Op: idasrc.Decode4}, {Op: idasrc.DecodeBuf}, {Op: idasrc.Decode2},
	}}
	rows := Diff(atlas, client)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if rows[3].Verdict != VerdictMinor {
		t.Fatalf("trailing padding byte: want Minor, got %s (%s)", rows[3].Verdict.Symbol(), rows[3].Note)
	}
	for _, r := range rows[:3] {
		if r.Verdict != VerdictMatch {
			t.Fatalf("row %d not match: %s %s", r.Index, r.Verdict.Symbol(), r.Note)
		}
	}

	// Two trailing extra bytes must STAY a blocker (only one byte is padding).
	atlas2 := append(atlas, atlaspacket.Call{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode1})
	rows2 := Diff(atlas2, client)
	if rows2[3].Verdict != VerdictBlocker && rows2[4].Verdict != VerdictBlocker {
		t.Fatalf("two trailing extras should remain a blocker, got rows[3]=%s rows[4]=%s",
			rows2[3].Verdict.Symbol(), rows2[4].Verdict.Symbol())
	}
}
