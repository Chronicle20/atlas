package handler

import (
	"math"
	"testing"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestSacrificeHpCost(t *testing.T) {
	cases := []struct {
		name      string
		firstLine uint32
		x         int16
		currentHp uint16
		want      uint16
	}{
		{"normal computation", 1000, 30, 5000, 300},
		{"truncating division", 99, 30, 5000, 29},
		{"x zero", 1000, 0, 5000, 0},
		{"x negative", 1000, -5, 5000, 0},
		{"miss (first line zero)", 0, 30, 5000, 0},
		{"clamp to hp minus one", 100000, 100, 500, 499},
		{"exact-kill boundary clamps", 1000, 100, 1000, 999},
		{"hp one is a no-op", 1000, 30, 1, 0},
		{"hp zero is a no-op", 1000, 30, 0, 0},
		{"narrowing guard caps at MaxInt16", 100000, 100, 65535, math.MaxInt16},
		{"max uint32 line does not wrap", math.MaxUint32, 100, 30000, 29999},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sacrificeHpCost(tc.firstLine, tc.x, tc.currentHp); got != tc.want {
				t.Fatalf("sacrificeHpCost(%d, %d, %d) = %d; want %d", tc.firstLine, tc.x, tc.currentHp, got, tc.want)
			}
		})
	}
}

func TestSacrificeFirstDamageLine(t *testing.T) {
	entry := func(monsterId uint32, damages []uint32) packetmodel.DamageInfo {
		return *packetmodel.NewDamageInfo(byte(len(damages))).
			SetMonsterId(monsterId).
			SetDamages(damages)
	}

	cases := []struct {
		name string
		ai   packetmodel.AttackInfo
		want uint32
	}{
		{
			"no damage entries",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee),
			0,
		},
		{
			"first entry has no lines",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee).
				AddDamageInfo(entry(100100, nil)),
			0,
		},
		{
			"multi-line first entry returns line zero only",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee).
				AddDamageInfo(entry(100100, []uint32{4000, 9999, 12345})),
			4000,
		},
		{
			"multi-target attack ignores second entry (FR-2 pin)",
			*packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee).
				AddDamageInfo(entry(100100, []uint32{4000})).
				AddDamageInfo(entry(100101, []uint32{99999})),
			4000,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sacrificeFirstDamageLine(tc.ai); got != tc.want {
				t.Fatalf("sacrificeFirstDamageLine() = %d; want %d", got, tc.want)
			}
		})
	}
}
