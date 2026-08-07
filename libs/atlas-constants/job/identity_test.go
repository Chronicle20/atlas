package job

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

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

// TestSet_Available_v61PirateStubPresentNotAvailable is the compiled-Set
// counterpart of the generator's golden test: v61's Pirate job stub
// resolves (Task 4 presence) but must not be Available (task-187 Task 5
// release gating -- Pirate did not release until v0.62).
func TestSet_Available_v61PirateStubPresentNotAvailable(t *testing.T) {
	v61 := newSet_gms_61_1()

	if _, ok := v61.Resolve(500); !ok {
		t.Fatal("v61 wire 500 should resolve (Pirate stub present in WZ semantics)")
	}
	if v61.Available(Pirate) {
		t.Fatal("v61 Pirate must be present-but-unavailable (released v0.62, after v61)")
	}
}

// TestSet_Available_v72PirateAvailable is the positive counterpart: the
// earliest provisioned post-Pirate column must report Pirate available.
func TestSet_Available_v72PirateAvailable(t *testing.T) {
	v72 := newSet_gms_72_1()
	if !v72.Available(Pirate) {
		t.Fatal("v72 Pirate should be available (released v0.62)")
	}
	if !v72.Available(Gm) {
		t.Fatal("v72 Gm should be available (always-released, stable class)")
	}
}

// TestSet_Name_v48 pins Name() returning the version-independent display
// name for a bound identity, and "" for one absent from the version.
func TestSet_Name_v48(t *testing.T) {
	v48 := newSet_gms_48_1()
	if got := v48.Name(Gm); got != "Gm" {
		t.Fatalf("v48 Name(Gm) = %q, want %q", got, "Gm")
	}
	if got := v48.Name(Pirate); got != "" {
		t.Fatalf("v48 Name(Pirate) = %q, want \"\" (Pirate absent from v48 semantics)", got)
	}
}

// TestSet_AvailableIdentities_SortedByWireId checks the accessor returns a
// non-empty, ascending-by-wire-id slice, and that it excludes an identity
// known to be present-but-unavailable at that version (v61 Pirate).
func TestSet_AvailableIdentities_SortedByWireId(t *testing.T) {
	v61 := newSet_gms_61_1()
	ids := v61.AvailableIdentities()
	if len(ids) == 0 {
		t.Fatal("v61 AvailableIdentities() should be non-empty (Explorer classes are always available)")
	}
	for _, id := range ids {
		if id == Pirate {
			t.Fatal("v61 AvailableIdentities() must not include Pirate (present but unreleased)")
		}
	}
	var prev Id
	for i, id := range ids {
		w, ok := v61.Wire(id)
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
// advancement_test.go, asserting the Identity-typed port agrees with the
// Id-typed original on the same identity/id pair.

func TestIsAIdentity_GmFamily(t *testing.T) {
	if !IsAIdentity(SuperGm, Gm) {
		t.Fatal("SuperGm is-a Gm family")
	}
	if IsAIdentity(Pirate, Gm) {
		t.Fatal("Pirate is not-a Gm family")
	}
}

func TestIsBeginnerIdentity(t *testing.T) {
	if !IsBeginnerIdentity(Evan) {
		t.Fatal("Evan is a beginner-band identity")
	}
	if IsBeginnerIdentity(Pirate) {
		t.Fatal("Pirate is not a beginner-band identity")
	}
}

func TestGetTypeIdentity_And_IsCygnusIdentity(t *testing.T) {
	if !IsCygnusIdentity(DawnWarriorStage1) {
		t.Fatal("DawnWarriorStage1 is a Cygnus identity")
	}
	if GetTypeIdentity(DawnWarriorStage1) != TypeCygnus {
		t.Fatalf("GetTypeIdentity(DawnWarriorStage1) = %v, want TypeCygnus", GetTypeIdentity(DawnWarriorStage1))
	}
	if IsCygnusIdentity(Pirate) {
		t.Fatal("Pirate is an Explorer identity, not Cygnus")
	}
	if GetTypeIdentity(Pirate) != TypeExplorer {
		t.Fatalf("GetTypeIdentity(Pirate) = %v, want TypeExplorer", GetTypeIdentity(Pirate))
	}
}

func TestGetSkillBookIdentity(t *testing.T) {
	if got := GetSkillBookIdentity(EvanStage2); got != 1 {
		t.Fatalf("GetSkillBookIdentity(EvanStage2) = %d, want 1", got)
	}
	if got := GetSkillBookIdentity(EvanStage10); got != 9 {
		t.Fatalf("GetSkillBookIdentity(EvanStage10) = %d, want 9", got)
	}
	if got := GetSkillBookIdentity(Pirate); got != 0 {
		t.Fatalf("GetSkillBookIdentity(Pirate) = %d, want 0 (not an Evan stage)", got)
	}
}

func TestAdvancementIdentity(t *testing.T) {
	if got := AdvancementIdentity(Beginner); got != 0 {
		t.Fatalf("AdvancementIdentity(Beginner) = %d, want 0", got)
	}
	if got := AdvancementIdentity(Pirate); got != 1 {
		t.Fatalf("AdvancementIdentity(Pirate) = %d, want 1 (branch root)", got)
	}
	if got := AdvancementIdentity(Buccaneer); got != 4 {
		t.Fatalf("AdvancementIdentity(Buccaneer) = %d, want 4 (Buccaneer=512, 2+512%%10=4)", got)
	}
	if got := AdvancementIdentity(EvanStage5); got != -1 {
		t.Fatalf("AdvancementIdentity(EvanStage5) = %d, want -1 (Evan stages don't map onto the 4-tier scheme)", got)
	}
}

func TestIsFourthJobIdentity(t *testing.T) {
	if !IsFourthJobIdentity(Buccaneer) {
		t.Fatal("Buccaneer is a 4th-job identity")
	}
	if IsFourthJobIdentity(Marauder) {
		t.Fatal("Marauder (3rd job) is not a 4th-job identity")
	}
	// Evan's 4th-job band is curated (EvanStage6..10), not derivable from
	// jobId%10==2 the way the branch jobs are -- this is the case that
	// proves IsFourthJobIdentity is really consulting the same curated
	// Jobs table as IsFourthJob, not a re-derived formula.
	if !IsFourthJobIdentity(EvanStage6) {
		t.Fatal("EvanStage6 is a 4th-job identity (curated Jobs table)")
	}
	if IsFourthJobIdentity(EvanStage5) {
		t.Fatal("EvanStage5 is not a 4th-job identity")
	}
}

func TestFromSkillIdentity(t *testing.T) {
	// BrawlerCorkscrewBlow = 5101004; 5101004/10000 = 510 = Brawler (the
	// job identity, not the Pirate branch root) -- matches
	// IdFromSkillId's floor(skillId/10000) convention exactly.
	if got := FromSkillIdentity(skill.BrawlerCorkscrewBlow); got != Brawler {
		t.Fatalf("FromSkillIdentity(BrawlerCorkscrewBlow=5101004) = %v, want Brawler (510)", got)
	}
}
