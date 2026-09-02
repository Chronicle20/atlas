package job

import "testing"

// TestJobsTableShape guards the hand-rewritten Jobs literal (task-185 collapsed
// the 82 package-level Job value vars into inline map values). A dropped job or
// a dropped fourthJob marker is silent data loss: atlas-configurations' two
// preset validators use the key set as their "does this job exist" check, and
// atlas-character reads IsFourthJob off it.
func TestJobsTableShape(t *testing.T) {
	if got := len(Jobs); got != 83 {
		t.Fatalf("len(Jobs) = %d; want 83", got)
	}
	fourth := 0
	for id, j := range Jobs {
		if j.Id() != id {
			t.Errorf("Jobs[%d].Id() = %d; every entry must be self-keyed", id, j.Id())
		}
		if j.IsFourthJob() {
			fourth++
		}
	}
	if fourth != 23 {
		t.Fatalf("fourthJob markers = %d; want 23", fourth)
	}
}

// TestFourthJobMembership pins the exact ids, so a marker cannot migrate from
// one job to another while keeping the count correct. Populate `want` from the
// awk command in the plan's Task 6 preamble — constants.go is the authority.
func TestFourthJobMembership(t *testing.T) {
	want := fourthJobIdsUnderTest()
	if len(want) != 23 {
		t.Fatalf("want list has %d entries; the fourthJob count is 23", len(want))
	}
	for _, id := range want {
		j, ok := Jobs[id]
		if !ok {
			t.Errorf("Jobs is missing id %d", id)
			continue
		}
		if !j.IsFourthJob() {
			t.Errorf("Jobs[%d].IsFourthJob() = false; want true", id)
		}
	}
	for id, j := range Jobs {
		if !j.IsFourthJob() {
			continue
		}
		found := false
		for _, w := range want {
			if w == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Jobs[%d] is marked fourthJob but is not in the expected set", id)
		}
	}
}

// TestFromSkillIdStillResolves covers the two accessors external services use:
// atlas-character/character/processor.go:1045 calls .Id(), and
// atlas-character/skill/model.go:34 calls .IsFourthJob() (FR-5.4 — this is why
// Job stays a struct rather than collapsing to map[Id]bool).
func TestFromSkillIdStillResolves(t *testing.T) {
	j, ok := FromSkillId(1121000)
	if !ok {
		t.Fatal("FromSkillId(1121000) not ok")
	}
	if j.Id() != HeroId {
		t.Fatalf("FromSkillId(1121000).Id() = %d; want %d", j.Id(), HeroId)
	}
	if !j.IsFourthJob() {
		t.Fatal("Hero must be a fourth job")
	}
}

// fourthJobIdsUnderTest is the transcription of every job whose literal carried
// `fourthJob: true` before the task-185 collapse. Derived mechanically from
// constants.go via:
//
//	awk '/^var [A-Z][A-Za-z0-9]* = Job\{/{name=$2} /fourthJob: true/{print name}' \
//	  libs/atlas-constants/job/constants.go | sort
//
// not from memory.
func fourthJobIdsUnderTest() []Id {
	return []Id{
		AranStage4Id,
		BishopId,
		BlazeWizardStage4Id,
		BowmasterId,
		BuccaneerId,
		CorsairId,
		DarkKnightId,
		DawnWarriorStage4Id,
		EvanStage10Id,
		EvanStage6Id,
		EvanStage7Id,
		EvanStage8Id,
		EvanStage9Id,
		FirePoisonArchMagicianId,
		HeroId,
		IceLightningArchMagicianId,
		MarksmanId,
		NightLordId,
		NightWalkerStage4Id,
		PaladinId,
		ShadowerId,
		ThunderBreakerStage4Id,
		WindArcherStage4Id,
	}
}
