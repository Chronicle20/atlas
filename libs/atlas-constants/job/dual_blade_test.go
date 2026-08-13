package job

import "testing"

// dualBlade is the five Dual Blade job identities, in advancement order
// (task-204). GMS introduced the branch at v0.88, so the first version
// column that releases it is gms 92.1.
var dualBlade = []Identity{
	BladeRecruit, BladeAcolyte, BladeSpecialist, BladeLord, BladeMaster,
}

// TestAvailable_DualBladeReleaseWindow is the regression this task exists
// for: before task-204 no identity's canonicalToken landed in the 430-434
// range, so classOf never returned "DualBlade", availability.csv's
// DualBlade rows were inert, and GET /data/job-availability emitted nothing
// for the branch -- which made atlas-ui's Jobs page (availability ∩ WZ
// presence) drop it silently on every version, including gms 92/95 where it
// genuinely shipped.
//
// jms 185.1 releases it too; that is the one place the otherwise very
// similar gms 87 and jms 185 columns diverge on this branch.
func TestAvailable_DualBladeReleaseWindow(t *testing.T) {
	released := map[string]bool{
		"gms 12.1": false, "gms 48.1": false, "gms 61.1": false, "gms 72.1": false,
		"gms 79.1": false, "gms 83.1": false, "gms 84.1": false, "gms 87.1": false,
		"gms 92.1": true, "gms 95.1": true, "jms 185.1": true,
	}
	for name, s := range allVersionSets() {
		for _, id := range dualBlade {
			if got := s.Available(id); got != released[name] {
				t.Errorf("%s: Available(%d) = %v, want %v", name, id, got, released[name])
			}
		}
	}
}

// TestResolveWire_DualBladeStubBoundButUnreleasedAtGms87 -- presence is not
// release, the same split CygnusStage4 exercises. gms 87's Skill.wz DOES
// carry job images 430-434, but as an incomplete stub: 17 of the 26 skill
// images the released columns ship, and 4300000 has no WZ name
// (divergences.csv gms,87,1,job,430). Resolve/Wire must keep answering for
// the ids; only Available flips.
func TestResolveWire_DualBladeStubBoundButUnreleasedAtGms87(t *testing.T) {
	s := newSet_gms_87_1()
	for _, id := range dualBlade {
		w, bound := s.Wire(id)
		if !bound {
			t.Fatalf("gms 87.1: Wire(%d) should resolve -- 430-434 are present in its Skill.wz", id)
		}
		if got, ok := s.Resolve(w); !ok || got != id {
			t.Fatalf("gms 87.1: Resolve(%d) = (%v, %v), want (%d, true)", w, got, ok, id)
		}
		if s.Available(id) {
			t.Errorf("gms 87.1: Available(%d) = true, want false -- the v87 WZ set is an unreleased stub", id)
		}
	}
}

// TestParentWire_DualBladeIsRootedAtRogue pins the advancement shape on the
// three columns that release it: a linear five-step chain hanging off Rogue,
// never off Beginner. Evidence for the Rogue root is quest 2351 ("First
// Mission: Infiltration"), whose demandSummary is "Make a job advancement as
// a #bRogue#k" -- see the parents table's Dual Blade block.
//
// Asserting on ParentWire (not ParentIdentity) is deliberate: it is what
// GET /data/job-availability actually serves, so this also pins that the
// whole chain stays available together. A hole anywhere in it would make the
// jobs below the hole render as extra roots.
func TestParentWire_DualBladeIsRootedAtRogue(t *testing.T) {
	chain := []struct{ child, parent Identity }{
		{BladeRecruit, Rogue},
		{BladeAcolyte, BladeRecruit},
		{BladeSpecialist, BladeAcolyte},
		{BladeLord, BladeSpecialist},
		{BladeMaster, BladeLord},
	}
	for _, name := range []string{"gms 92.1", "gms 95.1", "jms 185.1"} {
		s := allVersionSets()[name]
		for _, e := range chain {
			wantWire, ok := s.Wire(e.parent)
			if !ok {
				t.Fatalf("%s: Wire(%d) unbound -- cannot check the edge into it", name, e.parent)
			}
			got, ok := s.ParentWire(e.child)
			if !ok || got != wantWire {
				t.Errorf("%s: ParentWire(%d) = (%v, %v), want (%d, true)", name, e.child, got, ok, wantWire)
			}
		}
	}
}

// TestDualBladeSitsInsideTheThiefRange guards the ordering hazard called out
// in gen/availability.go's classOf: 430-434 are numerically inside the
// Explorer thief block. Today no arm claims 4xx broadly, so the DualBlade
// arm is unambiguous -- but if one is ever added ahead of it, Dual Blade
// would silently un-gate at gms 12-87. This test fails the moment the
// classification stops separating the two branches.
func TestDualBladeSitsInsideTheThiefRange(t *testing.T) {
	s := newSet_gms_83_1()
	for _, id := range []Identity{Assassin, Hermit, NightLord, Bandit, ChiefBandit, Shadower} {
		if !s.Available(id) {
			t.Errorf("gms 83.1: Available(%d) = false -- the Explorer thief branch must stay ungated", id)
		}
	}
	for _, id := range dualBlade {
		if id < Rogue || id > 499 {
			t.Errorf("Dual Blade identity %d left the 4xx thief range -- classOf's range arm needs re-deriving", id)
		}
	}
}
