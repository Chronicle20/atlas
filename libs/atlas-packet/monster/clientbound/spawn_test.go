package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v83 ida=0x67945a
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v87 ida=0x6b4fa6
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v95 ida=0x6589e0
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=jms_v185 ida=0x6f885c
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v84 ida=0x68fff0
func TestMonsterSpawnControlled(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, true, 100100, m)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterSpawnUncontrolled(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, false, 100100, m)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
