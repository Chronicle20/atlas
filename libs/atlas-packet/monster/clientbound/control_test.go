package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestMonsterControlActiveInit(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterControl(ControlTypeActiveInit, 5001, 100100, m)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterControlReset(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterControl(ControlTypeReset, 5001, 100100, m)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterControlActiveRequest(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterControl(ControlTypeActiveRequest, 5001, 100100, m)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
