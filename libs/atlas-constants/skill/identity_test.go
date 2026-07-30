package skill

import "testing"

// TestSet_ResolveWire_v48HideNotCorkscrew pins the PRD-motivating bug fix at
// the Set level (not just the generator's join map): at GMS v48, wire id
// 5101004 must resolve to SuperGmHide, not BrawlerCorkscrewBlow (which is
// what 5101004 means from GMS v72 onward -- see
// docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv).
func TestSet_ResolveWire_v48HideNotCorkscrew(t *testing.T) {
	v48 := newSet_gms_48_1()

	id, ok := v48.Resolve(5101004)
	if !ok || id != SuperGmHide {
		t.Fatalf("v48 Resolve(5101004) = (%v, %v), want (SuperGmHide, true)", id, ok)
	}

	wireId, ok := v48.Wire(SuperGmHide)
	if !ok || wireId != 5101004 {
		t.Fatalf("v48 Wire(SuperGmHide) = (%v, %v), want (5101004, true)", wireId, ok)
	}

	// BrawlerCorkscrewBlow must NOT resolve at v48 -- v48 has no Brawler
	// class, so 5101004 has exactly one meaning in this version's Set.
	if _, ok := v48.Wire(BrawlerCorkscrewBlow); ok {
		t.Fatal("v48 Wire(BrawlerCorkscrewBlow) should not resolve -- Brawler did not exist pre-v62")
	}
}

// TestSet_ResolveWire_v72CorkscrewNotHide is the post-Pirate counterpart:
// the same wire id now means the canonical post-v83 identity.
func TestSet_ResolveWire_v72CorkscrewNotHide(t *testing.T) {
	v72 := newSet_gms_72_1()

	id, ok := v72.Resolve(5101004)
	if !ok || id != BrawlerCorkscrewBlow {
		t.Fatalf("v72 Resolve(5101004) = (%v, %v), want (BrawlerCorkscrewBlow, true)", id, ok)
	}

	// SuperGmHide still resolves at v72 -- but to its own canonical wire id
	// 9101004 (the GM/SuperGM ids moved to the 900/910 job range once
	// Pirate/Brawler took over 500/510), not to 5101004.
	wireId, ok := v72.Wire(SuperGmHide)
	if !ok || wireId != 9101004 {
		t.Fatalf("v72 Wire(SuperGmHide) = (%v, %v), want (9101004, true)", wireId, ok)
	}
}

// TestSet_Resolve_UnknownWireId -- a zero-value Set (and an unbound wire id
// on a real Set) must report ok=false, never panic or silently return the
// zero Identity as if it were meaningful.
func TestSet_Resolve_UnknownWireId(t *testing.T) {
	var zero Set
	if _, ok := zero.Resolve(1); ok {
		t.Fatal("zero-value Set.Resolve should report ok=false")
	}
	if _, ok := zero.Wire(SuperGmHide); ok {
		t.Fatal("zero-value Set.Wire should report ok=false")
	}

	v48 := newSet_gms_48_1()
	if _, ok := v48.Resolve(999999999); ok {
		t.Fatal("v48 Resolve of an unbound wire id should report ok=false")
	}
}
