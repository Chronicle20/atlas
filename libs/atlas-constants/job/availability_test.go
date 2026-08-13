package job

import "testing"

// cygnusStage4 is the five Cygnus 4th-job identities, unreleased at every
// supported version (task-202 FR-2.1).
var cygnusStage4 = []Identity{
	DawnWarriorStage4, BlazeWizardStage4, WindArcherStage4,
	NightWalkerStage4, ThunderBreakerStage4,
}

// allVersionSets is every provisioned version column, in the order
// docs/tasks/task-187-version-aware-id-semantics uses.
func allVersionSets() map[string]Set {
	return map[string]Set{
		"gms 12.1":  newSet_gms_12_1(),
		"gms 48.1":  newSet_gms_48_1(),
		"gms 61.1":  newSet_gms_61_1(),
		"gms 72.1":  newSet_gms_72_1(),
		"gms 79.1":  newSet_gms_79_1(),
		"gms 83.1":  newSet_gms_83_1(),
		"gms 84.1":  newSet_gms_84_1(),
		"gms 87.1":  newSet_gms_87_1(),
		"gms 92.1":  newSet_gms_92_1(),
		"gms 95.1":  newSet_gms_95_1(),
		"jms 185.1": newSet_jms_185_1(),
	}
}

// TestAvailable_CygnusStage4NeverReleased pins FR-2.1 across every column.
func TestAvailable_CygnusStage4NeverReleased(t *testing.T) {
	for name, s := range allVersionSets() {
		for _, id := range cygnusStage4 {
			if s.Available(id) {
				t.Errorf("%s: Available(%d) = true, want false -- Cygnus 4th job shipped no skills at any supported version", name, id)
			}
		}
	}
}

// TestResolveWire_CygnusStage4StillPresent -- presence != release. The
// identities are genuinely in the WZ (1112.img exists, with an empty skill
// node), so Resolve/Wire must keep answering for them wherever the Cygnus
// class itself is bound. Deleting them from identities.yaml instead would
// have conflated the two axes task-187 built.
func TestResolveWire_CygnusStage4StillPresent(t *testing.T) {
	v83 := newSet_gms_83_1()
	for _, id := range cygnusStage4 {
		w, ok := v83.Wire(id)
		if !ok {
			t.Fatalf("v83 Wire(%d) should still resolve -- presence is not release", id)
		}
		if got, ok := v83.Resolve(w); !ok || got != id {
			t.Fatalf("v83 Resolve(%d) = (%v, %v), want (%d, true)", w, got, ok, id)
		}
	}
}

// TestResolveWire_SuperGmNotBoundAtJms185 pins task-202 FR-2.3: the JMS
// v185 Skill.wz root has no 910 image at all (900/Gm is present; 910/SuperGm
// is absent), confirmed against the GMS v95 Skill.wz which has both. The
// presence layer (identities.yaml's per-version wire join) already excludes
// SuperGm from jms 185 independently of availability.csv's released flag --
// this test guards that exclusion against regressing if the presence data
// is ever regenerated from a corrected/updated JMS WZ. See
// docs/tasks/task-202-version-correct-job-hierarchy/availability-audit.md.
func TestResolveWire_SuperGmNotBoundAtJms185(t *testing.T) {
	s := newSet_jms_185_1()
	if _, bound := s.Wire(SuperGm); bound {
		t.Fatalf("jms 185.1: Wire(SuperGm) resolved -- expected no binding (910.img is absent from the JMS v185 Skill.wz root)")
	}
	if s.Available(SuperGm) {
		t.Fatalf("jms 185.1: Available(SuperGm) = true, want false")
	}
}

// TestAvailable_CygnusTiers1To3NoRegression is the guard on the split: the
// tiers that DID ship must be unaffected -- available from gms 79 onward,
// unavailable at gms 72 and earlier.
func TestAvailable_CygnusTiers1To3NoRegression(t *testing.T) {
	tiers := []Identity{
		Noblesse,
		DawnWarriorStage1, DawnWarriorStage2, DawnWarriorStage3,
		BlazeWizardStage1, BlazeWizardStage2, BlazeWizardStage3,
		WindArcherStage1, WindArcherStage2, WindArcherStage3,
		NightWalkerStage1, NightWalkerStage2, NightWalkerStage3,
		ThunderBreakerStage1, ThunderBreakerStage2, ThunderBreakerStage3,
	}
	released := map[string]bool{
		"gms 12.1": false, "gms 48.1": false, "gms 61.1": false, "gms 72.1": false,
		"gms 79.1": true, "gms 83.1": true, "gms 84.1": true, "gms 87.1": true,
		"gms 92.1": true, "gms 95.1": true, "jms 185.1": true,
	}
	for name, s := range allVersionSets() {
		for _, id := range tiers {
			// An identity with no wire binding at this version cannot be
			// available regardless of release class; only assert where the
			// version actually binds it.
			if _, bound := s.Wire(id); !bound {
				continue
			}
			if got := s.Available(id); got != released[name] {
				t.Errorf("%s: Available(%d) = %v, want %v", name, id, got, released[name])
			}
		}
	}
}
