package model

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsMobAffectingBuff_PriestDoom(t *testing.T) {
	if !isMobAffectingBuff(skill.PriestDoomId) {
		t.Fatalf("isMobAffectingBuff(PriestDoomId) = false, want true")
	}
}
