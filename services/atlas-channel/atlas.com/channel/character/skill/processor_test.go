package skill_test

import (
	"atlas-channel/character/skill"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func mkSkillModel(t *testing.T, id uint32, level byte) skill.Model {
	t.Helper()
	m, err := skill.Extract(skill.RestModel{Id: id, Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	return m
}

// TestGetLevelIdentity proves the Identity form of GetLevel (task-187)
// resolves each trained skill's wire id through set before comparing,
// matching GetLevel's raw-wire behavior at a version where the wire id and
// canonical Identity token coincide (NightWalkerStage2ClawMastery is a
// version-stable Thief-branch root).
func TestGetLevelIdentity(t *testing.T) {
	skills := []skill.Model{
		mkSkillModel(t, uint32(skill2.NightWalkerStage2ClawMasteryId), 7),
		mkSkillModel(t, uint32(skill2.GunslingerGunMasteryId), 3),
	}
	set := constants.For("GMS", 83, 1).Skill

	if got := skill.GetLevelIdentity(skills, set, skill2.NightWalkerStage2ClawMastery); got != 7 {
		t.Errorf("GetLevelIdentity(NightWalkerStage2ClawMastery) = %d, want 7", got)
	}
	if got := skill.GetLevelIdentity(skills, set, skill2.GunslingerGunMastery); got != 3 {
		t.Errorf("GetLevelIdentity(GunslingerGunMastery) = %d, want 3", got)
	}
	if got := skill.GetLevelIdentity(skills, set, skill2.AssassinClawMastery); got != 0 {
		t.Errorf("GetLevelIdentity(untrained) = %d, want 0", got)
	}
}
