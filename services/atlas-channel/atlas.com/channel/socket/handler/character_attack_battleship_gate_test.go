package handler

import (
	"atlas-channel/battleship"
	"testing"

	"github.com/google/uuid"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func gateTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

func TestBattleshipAttackPermitted(t *testing.T) {
	tm := gateTenant(t)
	other := gateTenant(t)
	t.Cleanup(func() {
		battleship.GetRideMirror().EvictTenant(tm.Id())
		battleship.GetRideMirror().EvictTenant(other.Id())
	})
	battleship.GetRideMirror().Put(tm, 100, battleship.RideState{SkillLevel: 7})

	tests := []struct {
		name        string
		t           tenant.Model
		characterId uint32
		skillId     skill3.Id
		expected    bool
	}{
		{"cannon while riding", tm, 100, skill3.CorsairBattleshipCannonId, true},
		{"torpedo while riding", tm, 100, skill3.CorsairBattleshipTorpedoId, true},
		{"cannon on foot rejected (FR-6.1)", tm, 200, skill3.CorsairBattleshipCannonId, false},
		{"torpedo on foot rejected", tm, 200, skill3.CorsairBattleshipTorpedoId, false},
		{"tenant isolation", other, 100, skill3.CorsairBattleshipCannonId, false},
		{"unrelated skill always passes", tm, 200, skill3.CorsairRapidFireId, true},
		{"battleship mount skill itself is not gated", tm, 200, skill3.CorsairBattleshipId, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := battleshipAttackPermitted(tc.t, tc.characterId, tc.skillId); got != tc.expected {
				t.Errorf("battleshipAttackPermitted(%d, %d) = %v, want %v", tc.characterId, tc.skillId, got, tc.expected)
			}
		})
	}
}
