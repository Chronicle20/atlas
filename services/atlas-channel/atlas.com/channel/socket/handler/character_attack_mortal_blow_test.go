package handler

import (
	"math"
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestMortalBlowEligible(t *testing.T) {
	cases := []struct {
		name  string
		hp    uint32
		maxHp uint32
		x     int16
		want  bool
	}{
		// maxHp=999, x=20 -> threshold 999*20/100 = 199 (truncating division,
		// Cosmic parity: (getStats().getHp() * getX()) / 100).
		{"hp exactly at threshold", 199, 999, 20, true},
		{"hp one above threshold", 200, 999, 20, false},
		{"hp well below threshold", 1, 999, 20, true},
		{"x zero never eligible", 1, 999, 0, false},
		{"x negative never eligible", 1, 999, -5, false},
		{"maxHp zero never eligible", 0, 0, 20, false},
		// uint64 widening: maxHp near MaxUint32 must not overflow.
		// threshold = MaxUint32*50/100 = 2147483647 (floor).
		{"no overflow at MaxUint32", 2147483647, math.MaxUint32, 50, true},
		{"no overflow above threshold", 2147483648, math.MaxUint32, 50, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mortalBlowEligible(tc.hp, tc.maxHp, tc.x); got != tc.want {
				t.Fatalf("mortalBlowEligible(%d, %d, %d) = %v; want %v", tc.hp, tc.maxHp, tc.x, got, tc.want)
			}
		})
	}
}

func TestMortalBlowKillRoll(t *testing.T) {
	cases := []struct {
		name string
		roll int
		y    int16
		want bool
	}{
		{"roll equal to y procs", 5, 5, true},
		{"roll one above y misses", 6, 5, false},
		{"roll below y procs", 1, 5, true},
		{"y zero never procs", 1, 0, false},
		{"y negative never procs", 1, -1, false},
		{"y 100 always procs", 100, 100, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mortalBlowKillRoll(tc.roll, tc.y); got != tc.want {
				t.Fatalf("mortalBlowKillRoll(%d, %d) = %v; want %v", tc.roll, tc.y, got, tc.want)
			}
		})
	}
}

func TestIsMortalBlowAttack(t *testing.T) {
	cases := []struct {
		name    string
		at      packetmodel.AttackType
		skillId uint32
		want    bool
	}{
		{"ranged + Ranger Mortal Blow", packetmodel.AttackTypeRanged, uint32(skill3.RangerMortalBlowId), true},
		{"ranged + Sniper Mortal Blow", packetmodel.AttackTypeRanged, uint32(skill3.SniperMortalBlowId), true},
		{"ranged + other skill", packetmodel.AttackTypeRanged, uint32(skill3.RangerStrafeId), false},
		{"ranged + no skill", packetmodel.AttackTypeRanged, 0, false},
		{"melee + Mortal Blow id", packetmodel.AttackTypeMelee, uint32(skill3.RangerMortalBlowId), false},
		{"magic + Mortal Blow id", packetmodel.AttackTypeMagic, uint32(skill3.RangerMortalBlowId), false},
		{"energy + Mortal Blow id", packetmodel.AttackTypeEnergy, uint32(skill3.SniperMortalBlowId), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMortalBlowAttack(tc.at, tc.skillId); got != tc.want {
				t.Fatalf("isMortalBlowAttack(%v, %d) = %v; want %v", tc.at, tc.skillId, got, tc.want)
			}
		})
	}
}
