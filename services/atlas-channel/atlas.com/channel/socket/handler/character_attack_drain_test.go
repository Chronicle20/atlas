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

func TestDrainHealAmount(t *testing.T) {
	const big = uint32(2_000_000_000) // caps well above every raw heal below

	tests := []struct {
		name           string
		totalDamage    uint32
		x              int16
		monsterMaxHp   uint32
		effectiveMaxHp uint32
		want           int16
	}{
		// FR-3 spot values: percentage math with no cap engaged.
		{"drain L30 x=45: 1000 dmg heals 450", 1000, 45, big, big, 450},
		{"drain L1 x=16 floor: 333*16/100 = 53", 333, 16, big, big, 53},
		{"vampire L20 x=10: 5000 dmg heals 500", 5000, 10, big, big, 500},
		{"energy drain L20 x=20: 12345*20/100 = 2469", 12345, 20, big, big, 2469},
		// Caps.
		{"monster max HP caps the heal", 10000, 45, 100, big, 100},
		{"half effective max HP caps the heal", 10000, 45, big, 2000, 1000},
		{"half-cap floors odd effectiveMaxHp: 2001/2 = 1000", 10000, 45, big, 2001, 1000},
		{"tighter of the two caps wins", 10000, 45, 300, 2000, 300},
		// Zero guards.
		{"zero damage heals nothing", 0, 45, big, big, 0},
		{"x=0 heals nothing", 1000, 0, big, big, 0},
		{"negative x heals nothing", 1000, -5, big, big, 0},
		{"effectiveMaxHp=0 (stats fetch failed) heals nothing", 1000, 45, big, 0, 0},
		{"monsterMaxHp=0 heals nothing", 1000, 45, 0, big, 0},
		// Defensive int16 clamp with pathological inputs.
		{"int16 clamp on pathological damage", 4_000_000_000, 100, 4_294_967_295, 4_294_967_295, 32767},
		{"raw heal exactly at MaxInt16 passes through", 32767, 100, big, big, 32767},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := drainHealAmount(tc.totalDamage, tc.x, tc.monsterMaxHp, tc.effectiveMaxHp)
			if got != tc.want {
				t.Errorf("drainHealAmount(%d, %d, %d, %d) = %d, want %d",
					tc.totalDamage, tc.x, tc.monsterMaxHp, tc.effectiveMaxHp, got, tc.want)
			}
		})
	}
}
