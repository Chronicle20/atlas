package idasrc

import (
	"context"
	"testing"
)

func TestExportSourceResolve(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CLogin::OnCheckPasswordResult")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(f.Calls) != 8 {
		t.Errorf("calls: got %d, want 8", len(f.Calls))
	}
	if f.Calls[7].Op != Decode2 {
		t.Errorf("calls[7]: got %v, want Decode2", f.Calls[7].Op)
	}
	if f.Direction != DirClientbound {
		t.Errorf("direction: got %v", f.Direction)
	}
}

func TestExportSourceDispatcherPerMob(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CMob::OnDamaged")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// JSON entry has 2 calls; "per-mob" dispatcher prepends 1 (Decode4 mobId).
	if len(f.Calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (1 prefix + 2 leaf)", len(f.Calls))
	}
	if f.Calls[0].Op != Decode4 {
		t.Errorf("calls[0]: got %v, want Decode4 (dwMobId prefix)", f.Calls[0].Op)
	}
	if f.Calls[1].Op != Decode1 {
		t.Errorf("calls[1]: got %v, want Decode1 (damageType)", f.Calls[1].Op)
	}
	if f.Calls[2].Op != Decode4 {
		t.Errorf("calls[2]: got %v, want Decode4 (damage)", f.Calls[2].Op)
	}
}

func TestExportSourceDispatcherPerPet(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CPet::OnAction")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// "per-pet" prepends 2 (Decode4 characterId + Decode1 slot); 3 leaf calls.
	if len(f.Calls) != 5 {
		t.Fatalf("calls: got %d, want 5 (2 prefix + 3 leaf)", len(f.Calls))
	}
	if f.Calls[0].Op != Decode4 {
		t.Errorf("calls[0]: got %v, want Decode4 (characterId)", f.Calls[0].Op)
	}
	if f.Calls[1].Op != Decode1 {
		t.Errorf("calls[1]: got %v, want Decode1 (slot)", f.Calls[1].Op)
	}
	if f.Calls[2].Op != Decode1 {
		t.Errorf("calls[2]: got %v, want Decode1 (actionType)", f.Calls[2].Op)
	}
}

func TestExportSourceDispatcherPerPetRemote(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CUserRemote::OnPetActivated")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// "per-pet-remote" prepends only 1 (Decode4 characterId); 2 leaf calls.
	if len(f.Calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (1 prefix + 2 leaf)", len(f.Calls))
	}
	if f.Calls[0].Op != Decode4 {
		t.Errorf("calls[0]: got %v, want Decode4 (characterId)", f.Calls[0].Op)
	}
	if f.Calls[1].Op != Decode1 {
		t.Errorf("calls[1]: got %v, want Decode1 (slot — leaf, not prefix)", f.Calls[1].Op)
	}
}

func TestExportSourceServerboundIgnoresDispatcherAnnotation(t *testing.T) {
	// The "CPet::DoAction" entry has no dispatcher annotation — its calls
	// pass through verbatim. Sanity-checks that serverbound entries (which
	// shouldn't carry a dispatcher prefix) round-trip correctly.
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CPet::DoAction")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("calls: got %d, want 2 (no prefix, 2 leaf)", len(f.Calls))
	}
	if f.Direction != DirServerbound {
		t.Errorf("direction: got %v, want serverbound", f.Direction)
	}
}

// TestDelegateInlinesSubFunction verifies task-065 item 8: a call with
// op="Delegate" and a ref splices the referenced FName's resolved Calls into
// the parent's call list at that position, replacing the placeholder.
func TestDelegateInlinesSubFunction(t *testing.T) {
	src, err := NewExportSource("testdata/delegate_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CMobPool::OnMobEnterField")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// 2 parent calls + 3 inlined leaf calls = 5.
	if len(f.Calls) != 5 {
		t.Fatalf("calls: got %d, want 5 (2 parent + 3 inlined) — %+v", len(f.Calls), f.Calls)
	}
	wantOps := []Primitive{Decode4, Decode4, Decode2, Decode2, Decode1}
	for i, want := range wantOps {
		if f.Calls[i].Op != want {
			t.Errorf("calls[%d].Op: got %v, want %v", i, f.Calls[i].Op, want)
		}
	}
}

// TestDelegateANDsGuards verifies that when a Delegate call carries a guard,
// each inlined call's guard becomes "(outer-guard) && (inner-guard)" so the
// conditional-delegate semantic is preserved.
func TestDelegateANDsGuards(t *testing.T) {
	src, err := NewExportSource("testdata/delegate_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "Guarded::Parent")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("calls: got %d, want 2", len(f.Calls))
	}
	// calls[0]: Decode1 kind — unconditional.
	if f.Calls[0].Guard != "" {
		t.Errorf("calls[0].Guard: got %q, want \"\"", f.Calls[0].Guard)
	}
	// calls[1]: Decode4 payload — inlined under (kind > 0) && (version > 0).
	want := "(kind > 0) && (version > 0)"
	if f.Calls[1].Guard != want {
		t.Errorf("calls[1].Guard: got %q, want %q", f.Calls[1].Guard, want)
	}
}

// TestDelegateCycleDetected verifies that A → B → A terminates with an error
// rather than infinite recursion.
func TestDelegateCycleDetected(t *testing.T) {
	src, err := NewExportSource("testdata/delegate_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Resolve(context.Background(), "Cycle::A")
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	// The error chain should mention either "cycle" or both FNames.
	msg := err.Error()
	if !contains(msg, "cycle") || !contains(msg, "Cycle::") {
		t.Errorf("expected cycle error mentioning Cycle::*; got %q", msg)
	}
}

// TestDelegateDiamondAllowed verifies that the same leaf reachable via two
// different parent calls (NOT on the same descent path) inlines cleanly.
// This isn't a cycle — the cycle detector only trips on a re-entry to an
// fname currently on the resolve stack.
func TestDelegateDiamondAllowed(t *testing.T) {
	src, err := NewExportSource("testdata/delegate_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "Diamond::Root")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// Two parent Delegates each inline 1 leaf call → 2 total.
	if len(f.Calls) != 2 {
		t.Fatalf("calls: got %d, want 2 (1 + 1 from diamond)", len(f.Calls))
	}
	if f.Calls[0].Op != Decode1 || f.Calls[1].Op != Decode1 {
		t.Errorf("ops: got %v %v, want Decode1 Decode1", f.Calls[0].Op, f.Calls[1].Op)
	}
}

// TestDelegateRequiresRef verifies the error message when op="Delegate" but no
// ref field is set.
func TestDelegateRequiresRef(t *testing.T) {
	src, err := NewExportSource("testdata/delegate_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Resolve(context.Background(), "BadRef::Parent")
	if err == nil {
		t.Fatal("expected error for Delegate without ref, got nil")
	}
	if !contains(err.Error(), "Delegate") || !contains(err.Error(), "ref") {
		t.Errorf("expected error mentioning Delegate and ref; got %q", err.Error())
	}
}

// TestParsePrimEncodeDecodeEquivalence verifies task-065 item 7: parsePrim
// accepts both Encode×N and Decode×N op names and normalizes them to the
// same Primitive enum value. This is the binding that lets IDA Send*
// entries (which record Encode×N) and IDA OnX entries (which record
// Decode×N) flow through the same diff engine and compare against atlas's
// analyzer output (which itself normalizes Read/Write to Encode×N).
func TestParsePrimEncodeDecodeEquivalence(t *testing.T) {
	cases := []struct {
		enc, dec string
		want     Primitive
	}{
		{"Encode1", "Decode1", Decode1},
		{"Encode2", "Decode2", Decode2},
		{"Encode4", "Decode4", Decode4},
		{"Encode8", "Decode8", Decode8},
		{"EncodeStr", "DecodeStr", DecodeStr},
		{"EncodeBuf", "DecodeBuf", DecodeBuf},
		{"EncodeBuffer", "DecodeBuffer", DecodeBuf}, // legacy aliases
	}
	for _, c := range cases {
		e, err := parsePrim(c.enc)
		if err != nil {
			t.Errorf("parsePrim(%q): %v", c.enc, err)
			continue
		}
		d, err := parsePrim(c.dec)
		if err != nil {
			t.Errorf("parsePrim(%q): %v", c.dec, err)
			continue
		}
		if e != c.want || d != c.want {
			t.Errorf("parsePrim(%q)=%v parsePrim(%q)=%v; both want %v", c.enc, e, c.dec, d, c.want)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestDispatcherPrefixUnknownKind(t *testing.T) {
	// Forward-compat: unrecognized kinds yield no prefix (warn-and-continue,
	// not error). A future dispatcher kind can be added without breaking
	// existing JSON entries that name it before its support lands.
	if p := dispatcherPrefix("per-something-new"); p != nil {
		t.Errorf("expected nil prefix for unknown kind; got %d entries", len(p))
	}
	if p := dispatcherPrefix(""); p != nil {
		t.Errorf("expected nil prefix for empty kind; got %d entries", len(p))
	}
}
