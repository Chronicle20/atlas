package character

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestAttackMeleeEncode(t *testing.T) {
	ai := *model.NewAttackInfo(model.AttackTypeMelee)
	input := NewAttackMelee(12345, 50, 10, 15, 0, false, false, ai)
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}

func TestAttackRangedEncode(t *testing.T) {
	ai := *model.NewAttackInfo(model.AttackTypeRanged)
	input := NewAttackRanged(12345, 50, 10, 15, 2070000, false, false, ai)
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}

func TestAttackMagicEncode(t *testing.T) {
	ai := *model.NewAttackInfo(model.AttackTypeMagic)
	input := NewAttackMagic(12345, 50, 10, 15, 0, false, ai)
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}

func TestAttackEnergyEncode(t *testing.T) {
	ai := *model.NewAttackInfo(model.AttackTypeEnergy)
	input := NewAttackEnergy(12345, 50, 10, 15, 0, false, ai)
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}
