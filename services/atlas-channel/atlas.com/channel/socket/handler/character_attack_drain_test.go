package handler

import (
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsDrainSkill(t *testing.T) {
	tests := []struct {
		name string
		id   skill3.Id
		want bool
	}{
		{"assassin drain", skill3.AssassinDrainId, true},
		{"marauder energy drain", skill3.MarauderEnergyDrainId, true},
		{"thunder breaker energy drain", skill3.ThunderBreakerStage3EnergyDrainId, true},
		{"night walker vampire", skill3.NightWalkerStage2VampireId, true},
		{"aran combo drain is NOT attack-side drain", skill3.AranStage2ComboDrainId, false},
		{"zero id", skill3.Id(0), false},
		{"adjacent id", skill3.AssassinDrainId + 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDrainSkill(tc.id); got != tc.want {
				t.Errorf("isDrainSkill(%d) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
