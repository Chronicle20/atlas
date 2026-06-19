package handler

import (
	skill2 "atlas-channel/character/skill"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// buildSkillModel is a test helper that constructs a skill.Model for the
// given skill id and level using the package's Extract/RestModel path. It
// avoids creating a *_testhelpers.go (per CLAUDE.md) and avoids direct struct
// literals (private fields). The builder-style helper lives here, inline.
func buildSkillModel(t *testing.T, skillId skill.Id, level byte) skill2.Model {
	t.Helper()
	m, err := skill2.Extract(skill2.RestModel{
		Id:    uint32(skillId),
		Level: level,
	})
	if err != nil {
		t.Fatalf("buildSkillModel: Extract error: %v", err)
	}
	return m
}

// TestShouldBroadcastKeydown is a table-driven unit test for the gate function
// that controls whether a prepare/cancel packet should be relayed.
//
// Testability path chosen: no handler-test harness. The handler uses concrete
// processors (character.NewProcessor) that require a live tenant context and
// REST back-end. Instead we extracted shouldBroadcastKeydown as a small pure
// package-level function and test that directly, following the same pattern as
// computeReflect in character_attack_common_test.go.
func TestShouldBroadcastKeydown(t *testing.T) {
	// BowmasterHurricane is a known keydown skill (IsKeyDownSkill = true).
	const hurricaneId = skill.BowmasterHurricaneId
	// CorsairRapidFire is another keydown skill.
	const rapidFireId = skill.CorsairRapidFireId
	// HeroComboAttack (1100003) is NOT a keydown skill.
	const comboAttackId = skill.Id(1100003)

	cases := []struct {
		name    string
		skills  []skill2.Model
		skillId uint32
		want    bool
	}{
		{
			name: "owned keydown skill level>0 → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, hurricaneId, 1),
			},
			skillId: uint32(hurricaneId),
			want:    true,
		},
		{
			name: "owned keydown skill level>0 (second keydown) → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, rapidFireId, 3),
			},
			skillId: uint32(rapidFireId),
			want:    true,
		},
		{
			name: "non-keydown skill at level>0 → no broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, comboAttackId, 5),
			},
			skillId: uint32(comboAttackId),
			want:    false,
		},
		{
			name:    "skill not in character book → no broadcast",
			skills:  []skill2.Model{},
			skillId: uint32(hurricaneId),
			want:    false,
		},
		{
			name: "keydown skill at level 0 → no broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, hurricaneId, 0),
			},
			skillId: uint32(hurricaneId),
			want:    false,
		},
		{
			name: "multiple skills in book, only keydown one matches → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, comboAttackId, 5),
				buildSkillModel(t, hurricaneId, 2),
			},
			skillId: uint32(hurricaneId),
			want:    true,
		},
		{
			name: "multiple skills in book, queried skill is non-keydown → no broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, hurricaneId, 2),
				buildSkillModel(t, comboAttackId, 5),
			},
			skillId: uint32(comboAttackId),
			want:    false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := shouldBroadcastKeydown(tc.skills, tc.skillId)
			if got != tc.want {
				t.Errorf("shouldBroadcastKeydown(skills, %d) = %v, want %v", tc.skillId, got, tc.want)
			}
		})
	}
}
