package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v83 ida=0x66be61
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v87 ida=0x6a6cb3
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v95 ida=0x6521e0
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=jms_v185 ida=0x6e955a
func TestMonsterMovementRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			input := NewMonsterMovement(5001, false, true, false, 0, 0, 0, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, model.Movement{})
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterMovementRoundTripWithSkill(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			input := NewMonsterMovement(5001, true, true, true, 1, 100, 5, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, model.Movement{})
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
