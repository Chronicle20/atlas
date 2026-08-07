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
		{"job", 1512, "Cygnus"},       // ThunderBreakerStage4
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
