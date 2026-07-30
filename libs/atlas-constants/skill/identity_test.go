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

// TestSet_Available_v61PirateSkillStubPresentNotAvailable is the skill-domain
// counterpart of the job-domain golden test: PirateFlashFist (wire 5001001)
// is present in v61's WZ semantics but must be present-but-unavailable
// (Pirate released v0.62, after v61).
func TestSet_Available_v61PirateSkillStubPresentNotAvailable(t *testing.T) {
	v61 := newSet_gms_61_1()
	if _, ok := v61.Wire(PirateFlashFist); !ok {
		t.Fatal("v61 PirateFlashFist should be present (Pirate skill stub in WZ semantics)")
	}
	if v61.Available(PirateFlashFist) {
		t.Fatal("v61 PirateFlashFist must be present-but-unavailable (Pirate released v0.62, after v61)")
	}
}

// TestSet_Name_v48 pins Name() returning the version-independent display
// name for a bound identity, and "" for one absent from the version.
func TestSet_Name_v48(t *testing.T) {
	v48 := newSet_gms_48_1()
	if got := v48.Name(SuperGmHide); got != "Super Gm Hide" {
		t.Fatalf("v48 Name(SuperGmHide) = %q, want %q", got, "Super Gm Hide")
	}
	if got := v48.Name(PirateBulletTime); got != "" {
		t.Fatalf("v48 Name(PirateBulletTime) = %q, want \"\" (absent from v48 semantics)", got)
	}
}

// TestSet_AvailableIdentities_SortedByWireId checks the accessor returns a
// non-empty, ascending-by-wire-id slice.
func TestSet_AvailableIdentities_SortedByWireId(t *testing.T) {
	v72 := newSet_gms_72_1()
	ids := v72.AvailableIdentities()
	if len(ids) == 0 {
		t.Fatal("v72 AvailableIdentities() should be non-empty")
	}
	var prev Id
	for i, id := range ids {
		w, ok := v72.Wire(id)
		if !ok {
			t.Fatalf("AvailableIdentities()[%d] = %v has no Wire() binding", i, id)
		}
		if i > 0 && w < prev {
			t.Fatalf("AvailableIdentities() not sorted ascending by wire id: %v (%d) before wire %d", id, w, prev)
		}
		prev = w
	}
}

// ---- Identity-keyed semantic predicate tests (task-187 Task 7) ----
//
// Each of these mirrors an existing Id-typed test in model_test.go/
// mount_test.go/point_reset_test.go, asserting the Identity-typed port
// agrees with the Id-typed original on the same identity/id pair.

func TestIsKeyDownSkillIdentity(t *testing.T) {
	if !IsKeyDownSkillIdentity(BrawlerCorkscrewBlow) {
		t.Fatal("BrawlerCorkscrewBlow is a keydown identity")
	}
	if !IsKeyDownSkillIdentity(GunslingerGrenade) {
		t.Fatal("GunslingerGrenade is a keydown identity")
	}
	if IsKeyDownSkillIdentity(SuperGmHide) {
		t.Fatal("SuperGmHide is not a keydown identity")
	}
	if IsKeyDownSkillIdentity(PirateFlashFist) {
		t.Fatal("PirateFlashFist is not a keydown identity")
	}
}

func TestIsShootSkillNotUsingShootingWeaponIdentity(t *testing.T) {
	if !IsShootSkillNotUsingShootingWeaponIdentity(BuccaneerEnergyOrb) {
		t.Fatal("BuccaneerEnergyOrb does not use a shooting weapon")
	}
	if IsShootSkillNotUsingShootingWeaponIdentity(BrawlerCorkscrewBlow) {
		t.Fatal("BrawlerCorkscrewBlow is not in the not-using-shooting-weapon list")
	}
}

func TestIsShootSkillNotConsumingBulletIdentity(t *testing.T) {
	// Every not-using-shooting-weapon identity is also not-consuming-bullet.
	if !IsShootSkillNotConsumingBulletIdentity(BuccaneerEnergyOrb) {
		t.Fatal("BuccaneerEnergyOrb (shoot-weapon-exempt) must also be bullet-exempt")
	}
	if !IsShootSkillNotConsumingBulletIdentity(HermitShadowMeso) {
		t.Fatal("HermitShadowMeso does not consume a bullet")
	}
	if IsShootSkillNotConsumingBulletIdentity(CorsairRapidFire) {
		t.Fatal("CorsairRapidFire is not in either not-consuming-bullet list")
	}
}

func TestIsTamedMountSkillIdentity(t *testing.T) {
	if !IsTamedMountSkillIdentity(EvanMonsterRiding) {
		t.Fatal("EvanMonsterRiding is a tamed-mount identity")
	}
	if IsTamedMountSkillIdentity(BeginnerSpaceShip) {
		t.Fatal("BeginnerSpaceShip is a skill-only mount, not a tamed mount")
	}
}

func TestSkillOnlyMountVehicleIdentity(t *testing.T) {
	got, ok := SkillOnlyMountVehicleIdentity(BeginnerSpaceShip, 3)
	if !ok || got != 1932003 {
		t.Fatalf("SkillOnlyMountVehicleIdentity(BeginnerSpaceShip, 3) = (%v, %v), want (1932003, true)", got, ok)
	}
	got2, ok2 := SkillOnlyMountVehicleIdentity(LegendBalrogMount, 0)
	if !ok2 || got2 != 1932010 {
		t.Fatalf("SkillOnlyMountVehicleIdentity(LegendBalrogMount, 0) = (%v, %v), want (1932010, true)", got2, ok2)
	}
	if _, ok3 := SkillOnlyMountVehicleIdentity(EvanMonsterRiding, 0); ok3 {
		t.Fatal("EvanMonsterRiding is a tamed mount, not a skill-only mount")
	}
}

func TestIsPointResetExcludedIdentity(t *testing.T) {
	if !IsPointResetExcludedIdentity(Identity(21110007)) {
		t.Fatal("Aran hidden combo 21110007 must be point-reset excluded")
	}
	if !IsPointResetExcludedIdentity(Identity(9101004)) { // GM-range skill
		t.Fatal("GM-range skill 9101004 must be point-reset excluded")
	}
	if IsPointResetExcludedIdentity(BrawlerCorkscrewBlow) {
		t.Fatal("BrawlerCorkscrewBlow must not be point-reset excluded")
	}
}
