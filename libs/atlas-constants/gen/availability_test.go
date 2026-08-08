package main

import "testing"

// TestAvailability_v61PirateStubPresentNotAvailable is the task-187 Task 5
// PRD-motivating regression: v61's WZ snapshot already contains a Pirate job
// stub (auto-bound wire id 500 -> Pirate, task 4's semantics), but Pirate was
// not actually released until v0.62 (meymink anchor) -- so it must be
// present (semantics) but NOT available (release-gated).
func TestAvailability_v61PirateStubPresentNotAvailable(t *testing.T) {
	sem, err := BuildSemantics("gms", 61, 1)
	if err != nil {
		t.Fatal(err)
	}
	avail, err := BuildAvailability("gms", 61, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Pirate exists in v61 WZ semantics (stub) ...
	if _, ok := sem.JobByName["Pirate"]; !ok {
		t.Fatal("v61 semantics should contain Pirate stub")
	}
	// ... but is NOT released (availability)
	if avail.Job["Pirate"] {
		t.Fatal("v61 Pirate must be present-but-unavailable")
	}
}

// TestAvailability_v72PirateAvailable is the companion positive case: the
// earliest provisioned post-Pirate column must show Pirate as released.
func TestAvailability_v72PirateAvailable(t *testing.T) {
	avail, err := BuildAvailability("gms", 72, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !avail.Job["Pirate"] {
		t.Fatal("v72 Pirate must be available (released at v0.62)")
	}
}

// TestAvailability_SubsetOfSemantics is the task-187 Task 5 Step 2 subset
// invariant: every identity computed as available at a version must be
// present in that version's semantic map.
func TestAvailability_SubsetOfSemantics(t *testing.T) {
	if err := ValidateAvailabilitySubset("gms", 84, 1); err != nil {
		t.Fatal(err)
	}
}

// TestAvailability_AllProvisionedVersions smoke-tests BuildAvailability +
// ValidateAvailabilitySubset for every provisioned version, including the
// newly-reconciled gms_12 column.
func TestAvailability_AllProvisionedVersions(t *testing.T) {
	for _, v := range provisionedVersions {
		if _, err := BuildAvailability(v.Region, v.Major, v.Minor); err != nil {
			t.Fatalf("BuildAvailability(%s,%d,%d): %v", v.Region, v.Major, v.Minor, err)
		}
		if err := ValidateAvailabilitySubset(v.Region, v.Major, v.Minor); err != nil {
			t.Fatalf("ValidateAvailabilitySubset(%s,%d,%d): %v", v.Region, v.Major, v.Minor, err)
		}
	}
}

// TestClassOf_KnownBoundaries pins classOf's verified token ranges (see
// task-5-report.md for the full identities.yaml dump this was checked
// against) so a future edit to the ranges fails loudly here rather than
// silently mis-gating a whole class.
func TestClassOf_KnownBoundaries(t *testing.T) {
	cases := []struct {
		domain string
		token  uint64
		want   string
	}{
		{"job", 900, "GM"},            // Gm
		{"job", 910, "SuperGM"},       // SuperGm
		{"job", 500, "Pirate"},        // Pirate
		{"job", 522, "Pirate"},        // Corsair
		{"job", 1000, "Cygnus"},       // Noblesse
		{"job", 1512, "CygnusStage4"}, // ThunderBreakerStage4 (task-202: split out of Cygnus)
		{"job", 2000, "Aran"},         // Legend
		{"job", 2112, "Aran"},         // AranStage4
		{"job", 2001, "Evan"},         // Evan
		{"job", 2218, "Evan"},         // EvanStage10
		{"job", 0, ""},                // Beginner -- stable
		{"job", 800, ""},              // MapleLeafBrigadier -- stable (not a release class)
		{"skill", 9001000, "GM"},      // GmHaste
		{"skill", 9101004, "SuperGM"}, // SuperGmHide
		{"skill", 5000000, "Pirate"},  // PirateBulletTime
		{"skill", 5220001, "Pirate"},  // CorsairElementalBoost
		{"skill", 10001000, "Cygnus"}, // NoblesseThreeSnails
		{"skill", 15111007, "Cygnus"}, // ThunderBreakerStage3SharkWave
		{"skill", 20001004, "Aran"},   // LegendMonsterRiding
		{"skill", 21121008, "Aran"},   // AranStage4HerosWill
		{"skill", 20010012, "Evan"},   // EvanBlessOfNymph
		{"skill", 22181003, "Evan"},   // EvanStage10SoulStone
		{"skill", 1000, ""},           // BeginnerThreeSnails -- stable
		{"skill", 8001000, ""},        // MapleLeafBrigadierAntiMacro -- stable
	}
	for _, c := range cases {
		if got := classOf(c.domain, c.token); got != c.want {
			t.Errorf("classOf(%q, %d) = %q, want %q", c.domain, c.token, got, c.want)
		}
	}
}

// TestClassOf_CygnusStage4IsItsOwnClass pins task-202 FR-2.1/2.2. The whole
// 1000-1599 range used to map to a single "Cygnus" label, so there was no
// way to express "Cygnus, but not tier 4" -- and Cygnus 4th job is empty in
// the WZ at every supported version (docs/tasks/task-202-.../investigation.md
// Finding 3: 1112/1212/1312/1412/1512 all have a PRESENT but EMPTY skill
// node, contrast 1111 with 218 children).
//
// The five tokens are matched by an explicit list, not by arithmetic. The
// arithmetic form (t%10 == 2 && t >= 1100) is exact today but is a fact
// about today's five branches, not a rule; a sixth Cygnus branch must be
// added deliberately rather than inherited by accident.
func TestClassOf_CygnusStage4IsItsOwnClass(t *testing.T) {
	for _, tok := range []uint64{1112, 1212, 1312, 1412, 1512} {
		if got := classOf("job", tok); got != "CygnusStage4" {
			t.Errorf("classOf(job, %d) = %q, want CygnusStage4", tok, got)
		}
		// FR-2.2: the floor-by-10000 relationship must carry the split into
		// the skill domain for free.
		if got := classOf("skill", tok*10000+1000); got != "CygnusStage4" {
			t.Errorf("classOf(skill, %d) = %q, want CygnusStage4", tok*10000+1000, got)
		}
	}
}

// TestClassOf_CygnusTiers1To3Unchanged is the no-regression guard on the
// split: everything else in 1000-1599 must still be plain Cygnus.
func TestClassOf_CygnusTiers1To3Unchanged(t *testing.T) {
	for _, tok := range []uint64{1000, 1100, 1110, 1111, 1200, 1210, 1211, 1300, 1310, 1311, 1400, 1410, 1411, 1500, 1510, 1511} {
		if got := classOf("job", tok); got != "Cygnus" {
			t.Errorf("classOf(job, %d) = %q, want Cygnus", tok, got)
		}
	}
}
