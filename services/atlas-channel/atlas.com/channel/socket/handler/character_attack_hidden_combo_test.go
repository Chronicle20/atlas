package handler

import (
	"atlas-channel/character/skill"
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// The attack pipeline's ownership gate destroys the session for any attack
// whose skill id the character does not own. Aran's hidden combo variants are
// never in the skill book, so the gate has to resolve them through their
// parent -- otherwise a legitimate Aran is disconnected the moment the combo
// count escalates the swing (observed live: an attack with 21120009 at combo 7
// destroyed the session).
func TestResolveAttackSkillHiddenComboDefersToParent(t *testing.T) {
	skills := []skill.Model{
		comboTestSkill(t, skill3.AranStage4OverSwingId, 22),
		comboTestSkill(t, skill3.AranStage3FullSwingId, 17),
	}

	cases := []struct {
		name   string
		wireId skill3.Id
		level  byte
	}{
		{"over swing double swing", skill3.AranStage4OverswingDoubleSwingId, 22},
		{"over swing triple swing", skill3.AranStage4OverswingTripleSwingId, 22},
		{"full swing double swing", skill3.AranStage3FullSwingDoubleSwingId, 17},
		{"full swing triple swing", skill3.AranStage3FullSwingTripleSwingId, 17},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sk, ok := resolveAttackSkill(skills, tc.wireId)
			if !ok {
				t.Fatalf("resolveAttackSkill(%d): rejected an owned parent's hidden variant", tc.wireId)
			}
			if sk.Level() != tc.level {
				t.Fatalf("resolveAttackSkill(%d): level = %d, want the parent's %d", tc.wireId, sk.Level(), tc.level)
			}
		})
	}
}

// A hidden variant whose parent is not owned stays a rejection: the client can
// only produce the variant by escalating a swing it has, so an unowned parent
// is the same forged-attack signal the gate exists to catch.
func TestResolveAttackSkillHiddenComboRejectsUnownedParent(t *testing.T) {
	skills := []skill.Model{comboTestSkill(t, skill3.AranStage1ComboAbilityId, 20)}

	if _, ok := resolveAttackSkill(skills, skill3.AranStage4OverswingDoubleSwingId); ok {
		t.Fatal("resolveAttackSkill: accepted a hidden variant whose parent is not owned")
	}
}

func TestResolveAttackSkillOwnedSkill(t *testing.T) {
	skills := []skill.Model{comboTestSkill(t, skill3.AranStage4OverSwingId, 22)}

	sk, ok := resolveAttackSkill(skills, skill3.AranStage4OverSwingId)
	if !ok {
		t.Fatal("resolveAttackSkill: rejected an owned skill")
	}
	if sk.Id() != skill3.AranStage4OverSwingId || sk.Level() != 22 {
		t.Fatalf("resolveAttackSkill: got skill [%d] level [%d], want [%d] level [22]", sk.Id(), sk.Level(), skill3.AranStage4OverSwingId)
	}
}

func TestResolveAttackSkillUnownedSkill(t *testing.T) {
	skills := []skill.Model{comboTestSkill(t, skill3.AranStage4OverSwingId, 22)}

	if _, ok := resolveAttackSkill(skills, skill3.CrusaderComboAttackId); ok {
		t.Fatal("resolveAttackSkill: accepted a skill the character does not own")
	}
}
