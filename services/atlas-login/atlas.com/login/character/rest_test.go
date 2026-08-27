package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestTransformRoundTrip asserts Extract(Transform(m)) reproduces every
// field Extract actually populates. Extract's builder chain never calls
// SetSpawnPoint, SetPets, SetEquipment, SetInventory, SetRank, SetRankMove,
// SetJobRank, or SetJobRankMove, so those Model fields do not survive a
// round trip regardless of what Transform emits; see
// docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md.
func TestTransformRoundTrip(t *testing.T) {
	m := NewBuilder().
		SetId(1).
		SetAccountId(2).
		SetWorldId(world.Id(3)).
		SetName("Bob").
		SetLevel(4).
		SetExperience(5).
		SetGachaponExperience(6).
		SetStrength(7).
		SetDexterity(8).
		SetIntelligence(9).
		SetLuck(10).
		SetHp(11).
		SetMaxHp(12).
		SetMp(13).
		SetMaxMp(14).
		SetMeso(15).
		SetHpMpUsed(16).
		SetJobId(job.Id(17)).
		SetSkinColor(18).
		SetGender(19).
		SetFame(20).
		SetHair(21).
		SetFace(22).
		SetAp(23).
		SetSp("1,2,3").
		SetGm(24).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if got.id != m.id {
		t.Errorf("id mismatch. Expected %v, got %v", m.id, got.id)
	}
	if got.accountId != m.accountId {
		t.Errorf("accountId mismatch. Expected %v, got %v", m.accountId, got.accountId)
	}
	if got.worldId != m.worldId {
		t.Errorf("worldId mismatch. Expected %v, got %v", m.worldId, got.worldId)
	}
	if got.name != m.name {
		t.Errorf("name mismatch. Expected %v, got %v", m.name, got.name)
	}
	if got.level != m.level {
		t.Errorf("level mismatch. Expected %v, got %v", m.level, got.level)
	}
	if got.experience != m.experience {
		t.Errorf("experience mismatch. Expected %v, got %v", m.experience, got.experience)
	}
	if got.gachaponExperience != m.gachaponExperience {
		t.Errorf("gachaponExperience mismatch. Expected %v, got %v", m.gachaponExperience, got.gachaponExperience)
	}
	if got.strength != m.strength {
		t.Errorf("strength mismatch. Expected %v, got %v", m.strength, got.strength)
	}
	if got.dexterity != m.dexterity {
		t.Errorf("dexterity mismatch. Expected %v, got %v", m.dexterity, got.dexterity)
	}
	if got.intelligence != m.intelligence {
		t.Errorf("intelligence mismatch. Expected %v, got %v", m.intelligence, got.intelligence)
	}
	if got.luck != m.luck {
		t.Errorf("luck mismatch. Expected %v, got %v", m.luck, got.luck)
	}
	if got.hp != m.hp {
		t.Errorf("hp mismatch. Expected %v, got %v", m.hp, got.hp)
	}
	if got.maxHp != m.maxHp {
		t.Errorf("maxHp mismatch. Expected %v, got %v", m.maxHp, got.maxHp)
	}
	if got.mp != m.mp {
		t.Errorf("mp mismatch. Expected %v, got %v", m.mp, got.mp)
	}
	if got.maxMp != m.maxMp {
		t.Errorf("maxMp mismatch. Expected %v, got %v", m.maxMp, got.maxMp)
	}
	if got.meso != m.meso {
		t.Errorf("meso mismatch. Expected %v, got %v", m.meso, got.meso)
	}
	if got.hpMpUsed != m.hpMpUsed {
		t.Errorf("hpMpUsed mismatch. Expected %v, got %v", m.hpMpUsed, got.hpMpUsed)
	}
	if got.jobId != m.jobId {
		t.Errorf("jobId mismatch. Expected %v, got %v", m.jobId, got.jobId)
	}
	if got.skinColor != m.skinColor {
		t.Errorf("skinColor mismatch. Expected %v, got %v", m.skinColor, got.skinColor)
	}
	if got.gender != m.gender {
		t.Errorf("gender mismatch. Expected %v, got %v", m.gender, got.gender)
	}
	if got.fame != m.fame {
		t.Errorf("fame mismatch. Expected %v, got %v", m.fame, got.fame)
	}
	if got.hair != m.hair {
		t.Errorf("hair mismatch. Expected %v, got %v", m.hair, got.hair)
	}
	if got.face != m.face {
		t.Errorf("face mismatch. Expected %v, got %v", m.face, got.face)
	}
	if got.ap != m.ap {
		t.Errorf("ap mismatch. Expected %v, got %v", m.ap, got.ap)
	}
	if got.sp != m.sp {
		t.Errorf("sp mismatch. Expected %v, got %v", m.sp, got.sp)
	}
	if got.gm != m.gm {
		t.Errorf("gm mismatch. Expected %v, got %v", m.gm, got.gm)
	}
}
