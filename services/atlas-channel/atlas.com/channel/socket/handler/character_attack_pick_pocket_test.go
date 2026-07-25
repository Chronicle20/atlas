package handler

import (
	"testing"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestPickPocketWhitelisted(t *testing.T) {
	cases := []struct {
		name    string
		skillId uint32
		want    bool
	}{
		{"basic attack", 0, true},
		{"Double Stab", uint32(skill3.RogueDoubleStabId), true},
		{"Savage Blow", uint32(skill3.BanditSavageBlowId), true},
		{"Assaulter", uint32(skill3.ChiefBanditAssaulterId), true},
		{"Band of Thieves", uint32(skill3.ChiefBanditBandOfThievesId), true},
		{"Assassinate", uint32(skill3.ShadowerAssassinateId), true},
		{"Taunt", uint32(skill3.ShadowerTauntId), true},
		{"Boomerang Step", uint32(skill3.ShadowerBoomerangStepId), true},
		{"Meso Explosion not whitelisted", uint32(skill3.ChiefBanditMesoExplosionId), false},
		{"Pick Pocket itself not whitelisted", uint32(skill3.ChiefBanditPickpocketId), false},
		{"ranged skill not whitelisted", uint32(skill3.BowmasterHurricaneId), false},
		{"magic skill not whitelisted", uint32(skill3.MagicianMagicClawId), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickPocketWhitelisted(tc.skillId); got != tc.want {
				t.Fatalf("pickPocketWhitelisted(%d) = %v; want %v", tc.skillId, got, tc.want)
			}
		})
	}
}

func TestPickPocketMesoAmount(t *testing.T) {
	cases := []struct {
		name    string
		damage  uint32
		maxmeso int32
		want    uint32
	}{
		{"zero damage yields floor of 1", 0, 60, 1},
		{"exact maxmeso at 20000 damage", 20000, 60, 60},
		{"half maxmeso at 10000 damage", 10000, 60, 30},
		{"huge damage clamps to maxmeso", 2000000, 60, 60},
		{"zero maxmeso yields nothing", 10000, 0, 0},
		{"negative maxmeso yields nothing", 10000, -5, 0},
		{"product below 1 raised to floor (0.999)", 333, 60, 1},
		{"non-integer product truncates (25.5 -> 25)", 8500, 60, 25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickPocketMesoAmount(tc.damage, tc.maxmeso); got != tc.want {
				t.Fatalf("pickPocketMesoAmount(%d, %d) = %d; want %d", tc.damage, tc.maxmeso, got, tc.want)
			}
		})
	}
}
