package job

import "testing"

// TestSet_ResolveWire_v48GmNotPirate pins the PRD-motivating bug fix at the
// Set level: at GMS v48 (pre-Pirate), job wire id 500 must resolve to Gm,
// not Pirate (which is what 500 means from GMS v72 onward -- see
// docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv).
func TestSet_ResolveWire_v48GmNotPirate(t *testing.T) {
	v48 := newSet_gms_48_1()

	id, ok := v48.Resolve(500)
	if !ok || id != Gm {
		t.Fatalf("v48 Resolve(500) = (%v, %v), want (Gm, true)", id, ok)
	}

	wireId, ok := v48.Wire(Gm)
	if !ok || wireId != 500 {
		t.Fatalf("v48 Wire(Gm) = (%v, %v), want (500, true)", wireId, ok)
	}

	if _, ok := v48.Wire(Pirate); ok {
		t.Fatal("v48 Wire(Pirate) should not resolve -- Pirate did not exist pre-v62")
	}
}

// TestSet_ResolveWire_v72PirateNotGm is the post-Pirate counterpart: the
// same wire id now means the canonical post-v83 identity, and the
// pre-Pirate Gm/SuperGm slots have moved to 900/910.
func TestSet_ResolveWire_v72PirateNotGm(t *testing.T) {
	v72 := newSet_gms_72_1()

	id, ok := v72.Resolve(500)
	if !ok || id != Pirate {
		t.Fatalf("v72 Resolve(500) = (%v, %v), want (Pirate, true)", id, ok)
	}

	id2, ok2 := v72.Resolve(900)
	if !ok2 || id2 != Gm {
		t.Fatalf("v72 Resolve(900) = (%v, %v), want (Gm, true)", id2, ok2)
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
	if _, ok := zero.Wire(Gm); ok {
		t.Fatal("zero-value Set.Wire should report ok=false")
	}
}
