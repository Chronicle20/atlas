package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	"atlas-channel/skill/handler"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// energyTestSet returns the version-aware constant set for one GMS major.
func energyTestSet(major uint16) skill3.Set {
	return constants.For("GMS", major, 1).Skill
}

// energyTestSkill / energyTestCharacter / energyTestAttack mirror the
// constructors character_attack_combo_test.go already uses (:113-134), so the
// two attack-helper test files stay consistent.
func energyTestSkill(t *testing.T, id skill3.Id, level byte) skill.Model {
	t.Helper()
	m, err := skill.Extract(skill.RestModel{Id: uint32(id), Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	return m
}

func energyTestCharacter(t *testing.T, skills ...skill.Model) character.Model {
	t.Helper()
	return character.NewModelBuilder().SetId(1000).SetSkills(skills).MustBuild()
}

func energyTestAttack(at packetmodel.AttackType, skillId uint32, hits int) packetmodel.AttackInfo {
	ai := packetmodel.NewAttackInfo(at)
	ai.SetSkillId(skillId)
	for i := 0; i < hits; i++ {
		ai.AddDamageInfo(*packetmodel.NewDamageInfo(1))
	}
	return *ai
}

func TestEnergyChargeLine(t *testing.T) {
	v83 := energyTestSet(83)

	t.Run("adventurer line owned", func(t *testing.T) {
		line, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.MarauderEnergyChargeId, 20)})
		if !ok || line.skillId != skill3.MarauderEnergyChargeId || line.level != 20 {
			t.Fatalf("got (%+v,%v)", line, ok)
		}
	})
	t.Run("cygnus line owned", func(t *testing.T) {
		line, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.ThunderBreakerStage2EnergyChargeId, 10)})
		if !ok || line.skillId != skill3.ThunderBreakerStage2EnergyChargeId || line.level != 10 {
			t.Fatalf("got (%+v,%v)", line, ok)
		}
	})
	t.Run("both owned prefers adventurer", func(t *testing.T) {
		line, ok := energyChargeLine(v83, []skill.Model{
			energyTestSkill(t, skill3.ThunderBreakerStage2EnergyChargeId, 10),
			energyTestSkill(t, skill3.MarauderEnergyChargeId, 20),
		})
		if !ok || line.skillId != skill3.MarauderEnergyChargeId {
			t.Fatalf("got (%+v,%v)", line, ok)
		}
	})
	t.Run("level 0 is not owned", func(t *testing.T) {
		if _, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.MarauderEnergyChargeId, 0)}); ok {
			t.Fatal("level 0 must not resolve a line")
		}
	})
	t.Run("neither owned", func(t *testing.T) {
		if _, ok := energyChargeLine(v83, []skill.Model{energyTestSkill(t, skill3.CrusaderComboAttackId, 30)}); ok {
			t.Fatal("a non-pirate must not resolve a line")
		}
	})
	// AC-10: gms_v61 has the adventurer line only. The Cygnus identity has no
	// wire binding there, so the branch must be a no-op, never a bogus id.
	t.Run("gms_v61 cygnus identity is unavailable", func(t *testing.T) {
		v61 := energyTestSet(61)
		if _, ok := v61.Wire(skill3.ThunderBreakerStage2EnergyCharge); ok {
			t.Fatal("precondition: gms_v61 must not bind the Cygnus Energy Charge identity")
		}
		if _, ok := energyChargeLine(v61, []skill.Model{energyTestSkill(t, skill3.ThunderBreakerStage2EnergyChargeId, 10)}); ok {
			t.Fatal("gms_v61 must not resolve a Cygnus energy line")
		}
		line, ok := energyChargeLine(v61, []skill.Model{energyTestSkill(t, skill3.MarauderEnergyChargeId, 20)})
		if !ok || line.skillId != skill3.MarauderEnergyChargeId {
			t.Fatalf("gms_v61 adventurer line: got (%+v,%v)", line, ok)
		}
	})
}

func TestEnergyChargeGainAmount(t *testing.T) {
	for _, tc := range []struct {
		mobs int
		want int32
	}{{0, 0}, {-1, 0}, {1, 102}, {6, 612}} {
		if got := energyChargeGainAmount(tc.mobs); got != tc.want {
			t.Fatalf("mobs=%d: got %d want %d", tc.mobs, got, tc.want)
		}
	}
}

func TestEnergyChargeQualifies(t *testing.T) {
	v83 := energyTestSet(83)
	sharkWave, _ := v83.Resolve(skill3.ThunderBreakerStage3SharkWaveId)
	spark, _ := v83.Resolve(skill3.ThunderBreakerStage3SparkId)

	for _, tc := range []struct {
		name string
		at   packetmodel.AttackType
		id   skill3.Identity
		ok   bool
		want bool
	}{
		{"melee always", packetmodel.AttackTypeMelee, 0, false, true},
		{"energy/touch always", packetmodel.AttackTypeEnergy, 0, false, true},
		{"ranged shark wave", packetmodel.AttackTypeRanged, sharkWave, true, true},
		{"ranged other skill", packetmodel.AttackTypeRanged, spark, true, false},
		{"ranged unresolved", packetmodel.AttackTypeRanged, 0, false, false},
		{"magic never", packetmodel.AttackTypeMagic, 0, false, false},
	} {
		if got := energyChargeQualifies(tc.at, tc.id, tc.ok); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsEnergyBlast(t *testing.T) {
	v83 := energyTestSet(83)
	marauder, _ := v83.Resolve(skill3.MarauderEnergyBlastId)
	tb, _ := v83.Resolve(skill3.ThunderBreakerStage2EnergyBlastId)
	shockwave, _ := v83.Resolve(skill3.MarauderShockwaveId)

	if !isEnergyBlast(marauder, true) {
		t.Fatal("marauder energy blast must be recognised")
	}
	if !isEnergyBlast(tb, true) {
		t.Fatal("thunder breaker energy blast must be recognised")
	}
	if isEnergyBlast(shockwave, true) {
		t.Fatal("shockwave is MP-costed and must not be gated")
	}
	if isEnergyBlast(marauder, false) {
		t.Fatal("an unresolved id must never be treated as energy blast")
	}
}

func TestEnergyChargeTryUpdate(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	v83 := energyTestSet(83)

	marauder := energyTestCharacter(t, energyTestSkill(t, skill3.MarauderEnergyChargeId, 20))

	t.Run("emits one upsert of 102 x mobs", func(t *testing.T) {
		var gotSource, gotAmount, gotCap int32
		var gotLevel byte
		calls := 0
		deps := energyChargeDeps{emitUpsert: func(sourceId int32, level byte, amount int32, capValue int32) error {
			calls++
			gotSource, gotLevel, gotAmount, gotCap = sourceId, level, amount, capValue
			return nil
		}}

		energyChargeTryUpdate(l, v83, marauder, energyTestAttack(packetmodel.AttackTypeMelee, 0, 3), deps)

		if calls != 1 {
			t.Fatalf("expected exactly 1 emit, got %d", calls)
		}
		if gotSource != int32(skill3.MarauderEnergyChargeId) || gotLevel != 20 || gotAmount != 306 || gotCap != 10000 {
			t.Fatalf("got source=%d level=%d amount=%d cap=%d", gotSource, gotLevel, gotAmount, gotCap)
		}
	})

	t.Run("no line means no emit", func(t *testing.T) {
		nonPirate := energyTestCharacter(t, energyTestSkill(t, skill3.CrusaderComboAttackId, 30))
		called := false
		deps := energyChargeDeps{emitUpsert: func(int32, byte, int32, int32) error { called = true; return nil }}

		energyChargeTryUpdate(l, v83, nonPirate, energyTestAttack(packetmodel.AttackTypeMelee, 0, 3), deps)

		if called {
			t.Fatal("a character without the skill must produce no emit")
		}
	})

	t.Run("zero monsters means no emit", func(t *testing.T) {
		called := false
		deps := energyChargeDeps{emitUpsert: func(int32, byte, int32, int32) error { called = true; return nil }}

		energyChargeTryUpdate(l, v83, marauder, energyTestAttack(packetmodel.AttackTypeMelee, 0, 0), deps)

		if called {
			t.Fatal("an attack that hit nothing must produce no emit")
		}
	})

	// FR-7.2: the Energy Charge aura's own touch damage still grants energy.
	t.Run("energy/touch attack by the charge skill itself still gains", func(t *testing.T) {
		var gotAmount int32
		deps := energyChargeDeps{emitUpsert: func(_ int32, _ byte, amount int32, _ int32) error {
			gotAmount = amount
			return nil
		}}

		energyChargeTryUpdate(l, v83, marauder,
			energyTestAttack(packetmodel.AttackTypeEnergy, uint32(skill3.MarauderEnergyChargeId), 1), deps)

		if gotAmount != 102 {
			t.Fatalf("touch damage must still grant energy: got %d want 102", gotAmount)
		}
	})

	// AC-13: a failing dep must never escape as an error.
	t.Run("emit failure is swallowed", func(t *testing.T) {
		deps := energyChargeDeps{emitUpsert: func(int32, byte, int32, int32) error { return errors.New("kafka down") }}
		energyChargeTryUpdate(l, v83, marauder, energyTestAttack(packetmodel.AttackTypeMelee, 0, 3), deps)
	})
}

// AC-12 / FR-7.1: Energy Charge must never be registered as an attack-cast
// handler. Atlas's attack path applies no skill statups of its own — the only
// per-skill hook it consults is the LookupAttackCast registry — so registering
// Energy Charge there would reintroduce exactly the bug Cosmic patched: the
// aura's own touch damage perpetually refreshing the charged window
// (AbstractDealDamageHandler.java:183-184).
func TestEnergyChargeIsNotAnAttackCastHandler(t *testing.T) {
	for _, id := range []skill3.Identity{skill3.MarauderEnergyCharge, skill3.ThunderBreakerStage2EnergyCharge} {
		if _, ok := handler.LookupAttackCast(id); ok {
			t.Fatalf("skill [%d] must not be registered as an attack-cast handler; its own touch damage would refresh the charged window", id)
		}
	}
}
