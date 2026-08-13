package character

import (
	skilldata "atlas-effective-stats/external/data/skill"
	"atlas-effective-stats/stat"
	"errors"
	"testing"
)

func energyTestEffect(pad int16) func(uint32, byte) (*skilldata.EffectModel, error) {
	return func(uint32, byte) (*skilldata.EffectModel, error) {
		return &skilldata.EffectModel{WeaponAttack: pad}, nil
	}
}

func TestEnergyChargeBonus(t *testing.T) {
	// Charged: pad 15 at level 20 for 5110001, per WZ Skill.wz/511.img.xml.
	t.Run("charged grants pad as weapon attack", func(t *testing.T) {
		bs := energyChargeBonus("buff:5110001", 5110001, 20, 15000, energyTestEffect(15))
		if len(bs) != 1 {
			t.Fatalf("expected 1 bonus, got %d", len(bs))
		}
		if bs[0].StatType() != stat.TypeWeaponAttack || bs[0].Amount() != 15 {
			t.Fatalf("got %s=%d want weapon_attack=15", bs[0].StatType(), bs[0].Amount())
		}
	})

	// FR-5.3: the bar reading is NOT a stat value. A partial bar grants
	// nothing at all — never 4998 weapon attack.
	t.Run("partial bar grants nothing", func(t *testing.T) {
		if bs := energyChargeBonus("buff:5110001", 5110001, 20, 4998, energyTestEffect(15)); len(bs) != 0 {
			t.Fatalf("expected no bonus below the charged sentinel, got %+v", bs)
		}
	})

	// pad is 0 at levels 1-3; a zero bonus is omitted rather than emitted.
	t.Run("charged with pad 0 grants nothing", func(t *testing.T) {
		if bs := energyChargeBonus("buff:5110001", 5110001, 1, 15000, energyTestEffect(0)); len(bs) != 0 {
			t.Fatalf("expected no bonus for pad 0, got %+v", bs)
		}
	})

	t.Run("effect lookup failure grants nothing", func(t *testing.T) {
		fail := func(uint32, byte) (*skilldata.EffectModel, error) { return nil, errors.New("data down") }
		if bs := energyChargeBonus("buff:5110001", 5110001, 20, 15000, fail); len(bs) != 0 {
			t.Fatalf("expected no bonus on lookup failure, got %+v", bs)
		}
	})

	t.Run("nil effect grants nothing", func(t *testing.T) {
		nilEffect := func(uint32, byte) (*skilldata.EffectModel, error) { return nil, nil }
		if bs := energyChargeBonus("buff:5110001", 5110001, 99, 15000, nilEffect); len(bs) != 0 {
			t.Fatalf("expected no bonus for a missing effect row, got %+v", bs)
		}
	})
}
