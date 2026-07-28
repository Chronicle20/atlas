package handler

import (
	"testing"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestShouldApplyCastCooldown(t *testing.T) {
	tests := []struct {
		name     string
		cooldown uint32
		skillId  skill2.Id
		expected bool
	}{
		{"battleship exempt despite cooltime (FR-2.3)", 90, skill2.CorsairBattleshipId, false},
		{"other skill with cooltime applies", 90, skill2.PriestDispelId, true},
		{"other skill without cooltime skips", 0, skill2.PriestDispelId, false},
		{"battleship without cooltime skips", 0, skill2.CorsairBattleshipId, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldApplyCastCooldown(tc.cooldown, tc.skillId); got != tc.expected {
				t.Errorf("shouldApplyCastCooldown(%d, %d) = %v, want %v", tc.cooldown, tc.skillId, got, tc.expected)
			}
		})
	}
}
