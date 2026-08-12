package handler

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestSkillUseEffectCarriesMagnetDirection pins that the `left` argument reaches
// the encoder's monsterMagnetLeft field. effect_body.go derives isMonsterMagnet
// from the skill id, so the byte is present only for a magnet cast — a
// non-magnet skill encodes nothing extra regardless of what `left` says.
func TestSkillUseEffectCarriesMagnetDirection(t *testing.T) {
	l := logrus.New()
	ctx := pt.CreateContext("GMS", 83, 1)
	opts := map[string]interface{}{
		"operations": map[string]interface{}{
			string(charpkt.CharacterEffectSkillUse): float64(1),
		},
	}

	magnetLeft := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.HeroMonsterMagnetId), 120, 30, false, false, true)(l, ctx)(opts)
	magnetRight := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.HeroMonsterMagnetId), 120, 30, false, false, false)(l, ctx)(opts)

	if len(magnetLeft) != len(magnetRight) {
		t.Fatalf("magnet bodies differ in length (%d vs %d); the direction is one byte, not a length change",
			len(magnetLeft), len(magnetRight))
	}
	if string(magnetLeft) == string(magnetRight) {
		t.Fatal("left=true and left=false encoded identically; the direction byte is not reaching the encoder")
	}

	nonMagnetLeft := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.FighterRageId), 120, 20, false, false, true)(l, ctx)(opts)
	nonMagnetRight := charpkt.CharacterSkillUseEffectBody(
		uint32(skill.FighterRageId), 120, 20, false, false, false)(l, ctx)(opts)
	if string(nonMagnetLeft) != string(nonMagnetRight) {
		t.Fatal("a non-magnet skill encoded differently for left=true/false; the gate must be skill-id derived")
	}
}
