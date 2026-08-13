package job

import "testing"

// TestParentIdentity_Roots -- the five identities with no advancement
// parent. Every other identity must have one.
func TestParentIdentity_Roots(t *testing.T) {
	roots := []Identity{Beginner, MapleLeafBrigadier, Noblesse, Legend, Evan}
	for _, r := range roots {
		if p, ok := ParentIdentity(r); ok {
			t.Errorf("ParentIdentity(%d) = (%d, true), want (0, false) -- %d is a root", r, p, r)
		}
	}
	rootSet := map[Identity]bool{}
	for _, r := range roots {
		rootSet[r] = true
	}
	for id := range identityNames {
		if rootSet[id] {
			continue
		}
		if _, ok := ParentIdentity(id); !ok {
			t.Errorf("ParentIdentity(%d) has no entry -- every non-root identity needs one", id)
		}
	}
}

// TestParentIdentity_GmLineIsRootedAtBeginner pins FR-3.2: task-182's
// display convention now lives here. constants.go models Gm/SuperGm as
// independent roots (the REGISTRY view); the game presents them as an
// advancement line from Beginner (the ADVANCEMENT/DISPLAY view).
func TestParentIdentity_GmLineIsRootedAtBeginner(t *testing.T) {
	if p, ok := ParentIdentity(Gm); !ok || p != Beginner {
		t.Fatalf("ParentIdentity(Gm) = (%v, %v), want (Beginner, true)", p, ok)
	}
	if p, ok := ParentIdentity(SuperGm); !ok || p != Gm {
		t.Fatalf("ParentIdentity(SuperGm) = (%v, %v), want (Gm, true)", p, ok)
	}
}

// TestParentWire_v48GmLine -- the PRD's motivating case. At gms 48.1 wire
// id 500 is Gm and 510 is SuperGm, so the wire-level edges are 500 -> 0 and
// 510 -> 500.
func TestParentWire_v48GmLine(t *testing.T) {
	v48 := newSet_gms_48_1()

	gmWire, ok := v48.Wire(Gm)
	if !ok || gmWire != 500 {
		t.Fatalf("v48 Wire(Gm) = (%v, %v), want (500, true)", gmWire, ok)
	}
	if p, ok := v48.ParentWire(Gm); !ok || p != 0 {
		t.Fatalf("v48 ParentWire(Gm) = (%v, %v), want (0, true) -- Beginner is wire id 0", p, ok)
	}
	if p, ok := v48.ParentWire(SuperGm); !ok || p != 500 {
		t.Fatalf("v48 ParentWire(SuperGm) = (%v, %v), want (500, true)", p, ok)
	}
}

// TestParentWire_v72PirateAndGmAreIndependentRoots -- at gms 72.1 wire id
// 500 is Pirate and Gm has moved to 900. Both are depth-1 children of
// Beginner (wire id 0) and neither borrows the other's edge.
func TestParentWire_v72PirateAndGmAreIndependentRoots(t *testing.T) {
	v72 := newSet_gms_72_1()

	if w, ok := v72.Wire(Pirate); !ok || w != 500 {
		t.Fatalf("v72 Wire(Pirate) = (%v, %v), want (500, true)", w, ok)
	}
	if w, ok := v72.Wire(Gm); !ok || w != 900 {
		t.Fatalf("v72 Wire(Gm) = (%v, %v), want (900, true)", w, ok)
	}
	if p, ok := v72.ParentWire(Pirate); !ok || p != 0 {
		t.Fatalf("v72 ParentWire(Pirate) = (%v, %v), want (0, true)", p, ok)
	}
	if p, ok := v72.ParentWire(Gm); !ok || p != 0 {
		t.Fatalf("v72 ParentWire(Gm) = (%v, %v), want (0, true)", p, ok)
	}
}

// TestParentWire_EdgeAlwaysPointsAtAnAvailableJob is FR-3.4's invariant
// across every version: if ParentWire returns a wire id, that wire id must
// belong to an identity that is itself available at this version. The API
// must never emit an edge pointing at a job absent from its own response.
func TestParentWire_EdgeAlwaysPointsAtAnAvailableJob(t *testing.T) {
	for name, s := range allVersionSets() {
		for _, id := range s.AvailableIdentities() {
			w, ok := s.ParentWire(id)
			if !ok {
				continue
			}
			pid, resolved := s.Resolve(w)
			if !resolved {
				t.Errorf("%s: ParentWire(%d) = %d, which resolves to no identity", name, id, w)
				continue
			}
			if !s.Available(pid) {
				t.Errorf("%s: ParentWire(%d) = %d (identity %d), which is not available", name, id, w, pid)
			}
		}
	}
}

// TestParentWire_D7PolicyGuard pins design D7. ParentWire makes an entry a
// ROOT when its parent is unavailable; it does NOT walk up to the nearest
// available ancestor. The two policies are indistinguishable on today's
// version set -- every available identity's parent is also available -- so
// the choice is currently unobservable. This test makes the day it becomes
// observable a test failure that forces a decision, rather than a silent
// rendering change.
func TestParentWire_D7PolicyGuard(t *testing.T) {
	for name, s := range allVersionSets() {
		for _, id := range s.AvailableIdentities() {
			p, hasParent := ParentIdentity(id)
			if !hasParent {
				continue
			}
			if !s.Available(p) {
				t.Errorf("%s: available identity %d has unavailable parent %d -- literal-root and nearest-available-ancestor now disagree; design D7 must be re-decided before this is silently rendered", name, id, p)
			}
		}
	}
}
