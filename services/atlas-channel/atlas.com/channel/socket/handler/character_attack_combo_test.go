package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"
	buff2 "atlas-channel/kafka/message/buff"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// comboTestBuff builds an active (or expired) COMBO buff carrying the given
// stat value (orb count + 1) under the given source id, for the Enrage
// orb-cap gate tests.
func comboTestBuff(sourceId int32, comboValue int32, expired bool) buff.Model {
	exp := time.Now().Add(time.Hour)
	if expired {
		exp = time.Now().Add(-time.Hour)
	}
	return buff.NewBuff(sourceId, 20, 150000, []stat.Model{stat.NewStat("COMBO", comboValue)}, time.Now().Add(-time.Minute), exp, false)
}

func comboTestLine(t *testing.T, advanced bool) comboLine {
	t.Helper()
	skills := []skill.Model{comboTestSkill(t, skill3.CrusaderComboAttackId, 30)}
	if advanced {
		skills = append(skills, comboTestSkill(t, skill3.HeroAdvancedComboAttackId, 30))
	}
	line, ok := comboSkillIds(skills)
	if !ok {
		t.Fatal("comboSkillIds: expected a combo line")
	}
	return line
}

func comboTestEffectLookup(t *testing.T, x int16) func(uint32, byte) (effect.Model, error) {
	return func(uint32, byte) (effect.Model, error) { return comboTestEffect(t, x, 0), nil }
}

func TestComboCurrentOrbValue(t *testing.T) {
	line := comboTestLine(t, false)
	src := int32(skill3.CrusaderComboAttackId)

	t.Run("returns value from active combo buff", func(t *testing.T) {
		v, ok := comboCurrentOrbValue(line, []buff.Model{comboTestBuff(src, 7, false)})
		if !ok || v != 7 {
			t.Fatalf("got (%d,%v), want (7,true)", v, ok)
		}
	})
	t.Run("ignores expired combo buff", func(t *testing.T) {
		if _, ok := comboCurrentOrbValue(line, []buff.Model{comboTestBuff(src, 7, true)}); ok {
			t.Fatal("expected ok=false for expired buff")
		}
	})
	t.Run("ignores other-source buff", func(t *testing.T) {
		if _, ok := comboCurrentOrbValue(line, []buff.Model{comboTestBuff(int32(skill3.HeroEnrageId), 7, false)}); ok {
			t.Fatal("expected ok=false when no buff under the combo source")
		}
	})
	t.Run("false when no buffs", func(t *testing.T) {
		if _, ok := comboCurrentOrbValue(line, nil); ok {
			t.Fatal("expected ok=false with no buffs")
		}
	})
}

func TestComboAtOrbCap(t *testing.T) {
	// Advanced line: governing effect is Advanced Combo (x=10 -> cap 11 = 10 orbs).
	line := comboTestLine(t, true)
	src := int32(skill3.CrusaderComboAttackId)

	t.Run("at cap is eligible", func(t *testing.T) {
		if !comboAtOrbCap(line, []buff.Model{comboTestBuff(src, 11, false)}, comboTestEffectLookup(t, 10)) {
			t.Fatal("expected eligible at value == x+1")
		}
	})
	t.Run("below cap is ineligible", func(t *testing.T) {
		if comboAtOrbCap(line, []buff.Model{comboTestBuff(src, 10, false)}, comboTestEffectLookup(t, 10)) {
			t.Fatal("expected ineligible below cap")
		}
	})
	t.Run("no combo buff is ineligible", func(t *testing.T) {
		if comboAtOrbCap(line, nil, comboTestEffectLookup(t, 10)) {
			t.Fatal("expected ineligible with no active combo buff")
		}
	})
	t.Run("effect lookup error is ineligible", func(t *testing.T) {
		bad := func(uint32, byte) (effect.Model, error) { return effect.Model{}, errors.New("boom") }
		if comboAtOrbCap(line, []buff.Model{comboTestBuff(src, 11, false)}, bad) {
			t.Fatal("expected ineligible on effect lookup error")
		}
	})
	t.Run("non-advanced line caps lower", func(t *testing.T) {
		// Combo-only line: governing effect is Combo Attack (x=5 -> cap 6 = 5 orbs).
		plain := comboTestLine(t, false)
		if !comboAtOrbCap(plain, []buff.Model{comboTestBuff(src, 6, false)}, comboTestEffectLookup(t, 5)) {
			t.Fatal("expected eligible at value 6 with x=5")
		}
		if comboAtOrbCap(plain, []buff.Model{comboTestBuff(src, 6, false)}, comboTestEffectLookup(t, 10)) {
			t.Fatal("expected ineligible at value 6 when x=10 (cap 11)")
		}
	})
}

func comboTestSkill(t *testing.T, id skill3.Id, level byte) skill.Model {
	t.Helper()
	m, err := skill.Extract(skill.RestModel{Id: uint32(id), Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	return m
}

func comboTestCharacter(t *testing.T, skills ...skill.Model) character.Model {
	t.Helper()
	return character.NewBuilder().SetId(1).SetSkills(skills).MustBuild()
}

func comboTestAttack(skillId uint32, hits int) packetmodel.AttackInfo {
	ai := packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	ai.SetSkillId(skillId)
	for i := 0; i < hits; i++ {
		ai.AddDamageInfo(*packetmodel.NewDamageInfo(1))
	}
	return *ai
}

func comboTestEffect(t *testing.T, x int16, prop float64) effect.Model {
	t.Helper()
	se, err := effect.Extract(effect.RestModel{X: x, Prop: prop})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return se
}

type comboEmitRecord struct {
	sourceId  int32
	operation string
	amount    int32
	capValue  int32
}

// comboTestDeps returns deps that record every emit and serve a fixed
// governing effect. roll is fixed so double-orb branches are deterministic.
func comboTestDeps(t *testing.T, se effect.Model, roll float64, emitted *[]comboEmitRecord) comboOrbDeps {
	t.Helper()
	return comboOrbDeps{
		getEffect: func(skillId uint32, level byte) (effect.Model, error) {
			return se, nil
		},
		emitUpdate: func(sourceId int32, operation string, amount int32, capValue int32) error {
			*emitted = append(*emitted, comboEmitRecord{sourceId, operation, amount, capValue})
			return nil
		},
		roll: func() float64 { return roll },
	}
}

func TestComboSkillIds(t *testing.T) {
	t.Run("adventurer line", func(t *testing.T) {
		line, ok := comboSkillIds([]skill.Model{
			comboTestSkill(t, skill3.CrusaderComboAttackId, 20),
			comboTestSkill(t, skill3.HeroAdvancedComboAttackId, 10),
		})
		if !ok {
			t.Fatal("expected ok")
		}
		if line.comboId != skill3.CrusaderComboAttackId || line.comboLevel != 20 {
			t.Fatalf("combo = %d L%d", line.comboId, line.comboLevel)
		}
		if line.advId != skill3.HeroAdvancedComboAttackId || line.advLevel != 10 {
			t.Fatalf("adv = %d L%d", line.advId, line.advLevel)
		}
	})
	t.Run("cygnus line", func(t *testing.T) {
		line, ok := comboSkillIds([]skill.Model{
			comboTestSkill(t, skill3.DawnWarriorStage3ComboAttackId, 15),
		})
		if !ok {
			t.Fatal("expected ok")
		}
		if line.comboId != skill3.DawnWarriorStage3ComboAttackId || line.comboLevel != 15 {
			t.Fatalf("combo = %d L%d", line.comboId, line.comboLevel)
		}
		if line.advId != skill3.DawnWarriorStage3AdvancedComboId || line.advLevel != 0 {
			t.Fatalf("adv = %d L%d, want DawnWarrior adv at 0", line.advId, line.advLevel)
		}
	})
	t.Run("no combo line", func(t *testing.T) {
		if _, ok := comboSkillIds([]skill.Model{comboTestSkill(t, skill3.CrusaderShoutId, 20)}); ok {
			t.Fatal("expected ok=false without Combo Attack")
		}
	})
	t.Run("combo at level 0", func(t *testing.T) {
		if _, ok := comboSkillIds([]skill.Model{comboTestSkill(t, skill3.CrusaderComboAttackId, 0)}); ok {
			t.Fatal("expected ok=false at level 0")
		}
	})
}

func TestIsComboFinisher(t *testing.T) {
	finishers := []skill3.Id{
		skill3.CrusaderPanicSwordId, skill3.CrusaderPanicAxeId,
		skill3.CrusaderComaSwordId, skill3.CrusaderComaAxeId,
		skill3.DawnWarriorStage3PanicId, skill3.DawnWarriorStage3ComaId,
	}
	for _, id := range finishers {
		if !isComboFinisher(id) {
			t.Fatalf("expected %d to be a finisher", id)
		}
	}
	for _, id := range []skill3.Id{skill3.CrusaderComboAttackId, skill3.CrusaderShoutId, skill3.Id(0)} {
		if isComboFinisher(id) {
			t.Fatalf("expected %d NOT to be a finisher", id)
		}
	}
}

func TestComboGainAmount(t *testing.T) {
	cases := []struct {
		name       string
		advLearned bool
		prop       float64
		roll       float64
		want       int32
	}{
		{"no advanced combo", false, 0.60, 0.0, 1},
		{"roll under prop", true, 0.60, 0.59, 2},
		{"roll equal prop", true, 0.60, 0.60, 1},
		{"roll over prop", true, 0.60, 0.61, 1},
		{"prop >= 1 always doubles", true, 1.0, 0.99, 2},
		{"zero prop never doubles", true, 0.0, 0.0, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := comboGainAmount(tc.advLearned, tc.prop, tc.roll); got != tc.want {
				t.Fatalf("comboGainAmount(%v, %v, %v) = %d; want %d", tc.advLearned, tc.prop, tc.roll, got, tc.want)
			}
		})
	}
}

func TestComboOrbTryUpdate(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	crusader := comboTestCharacter(t, comboTestSkill(t, skill3.CrusaderComboAttackId, 20))
	hero := comboTestCharacter(t,
		comboTestSkill(t, skill3.CrusaderComboAttackId, 30),
		comboTestSkill(t, skill3.HeroAdvancedComboAttackId, 30),
	)

	t.Run("no combo line emits nothing", func(t *testing.T) {
		var emitted []comboEmitRecord
		c := comboTestCharacter(t, comboTestSkill(t, skill3.CrusaderShoutId, 20))
		comboOrbTryUpdate(l, c, comboTestAttack(0, 1), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("finisher emits SET 1 even with zero hits", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(uint32(skill3.CrusaderPanicSwordId), 0), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 1 {
			t.Fatalf("emitted %d, want 1", len(emitted))
		}
		e := emitted[0]
		if e.operation != buff2.StatOperationSet || e.amount != 1 || e.sourceId != int32(skill3.CrusaderComboAttackId) {
			t.Fatalf("got %+v, want SET 1 on Combo Attack sourceId", e)
		}
	})

	t.Run("cygnus finisher targets cygnus combo sourceId", func(t *testing.T) {
		var emitted []comboEmitRecord
		dw := comboTestCharacter(t, comboTestSkill(t, skill3.DawnWarriorStage3ComboAttackId, 15))
		comboOrbTryUpdate(l, dw, comboTestAttack(uint32(skill3.DawnWarriorStage3PanicId), 1), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 1 {
			t.Fatalf("emitted %d, want 1", len(emitted))
		}
		e := emitted[0]
		if e.operation != buff2.StatOperationSet || e.amount != 1 || e.sourceId != int32(skill3.DawnWarriorStage3ComboAttackId) {
			t.Fatalf("got %+v, want SET 1 on DawnWarrior Combo sourceId", e)
		}
	})

	t.Run("shout emits nothing", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(uint32(skill3.CrusaderShoutId), 3), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("zero hits emits nothing on non-finisher", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 0), comboTestDeps(t, comboTestEffect(t, 3, 0), 0.99, &emitted))
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("normal attack gains one orb with cap x+1", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 1), comboTestDeps(t, comboTestEffect(t, 5, 0), 0.99, &emitted))
		if len(emitted) != 1 {
			t.Fatalf("emitted %d, want 1", len(emitted))
		}
		e := emitted[0]
		if e.operation != buff2.StatOperationIncrement || e.amount != 1 || e.capValue != 6 || e.sourceId != int32(skill3.CrusaderComboAttackId) {
			t.Fatalf("got %+v, want INCREMENT 1 cap 6", e)
		}
	})

	t.Run("advanced combo double orb on successful roll", func(t *testing.T) {
		var emitted []comboEmitRecord
		comboOrbTryUpdate(l, hero, comboTestAttack(0, 1), comboTestDeps(t, comboTestEffect(t, 10, 0.60), 0.10, &emitted))
		if len(emitted) != 1 {
			t.Fatalf("emitted %d, want 1", len(emitted))
		}
		e := emitted[0]
		if e.operation != buff2.StatOperationIncrement || e.amount != 2 || e.capValue != 11 {
			t.Fatalf("got %+v, want INCREMENT 2 cap 11 (adv combo effect governs)", e)
		}
	})

	t.Run("effect lookup error emits nothing", func(t *testing.T) {
		var emitted []comboEmitRecord
		deps := comboTestDeps(t, comboTestEffect(t, 5, 0), 0.99, &emitted)
		deps.getEffect = func(skillId uint32, level byte) (effect.Model, error) {
			return effect.Model{}, errors.New("boom")
		}
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 1), deps)
		if len(emitted) != 0 {
			t.Fatalf("emitted %d, want 0", len(emitted))
		}
	})

	t.Run("emit error is swallowed", func(t *testing.T) {
		var emitted []comboEmitRecord
		deps := comboTestDeps(t, comboTestEffect(t, 5, 0), 0.99, &emitted)
		deps.emitUpdate = func(sourceId int32, operation string, amount int32, capValue int32) error {
			return errors.New("kafka down")
		}
		// must not panic
		comboOrbTryUpdate(l, crusader, comboTestAttack(0, 1), deps)
	})

	t.Run("finisher emit error is swallowed", func(t *testing.T) {
		var emitted []comboEmitRecord
		deps := comboTestDeps(t, comboTestEffect(t, 5, 0), 0.99, &emitted)
		deps.emitUpdate = func(sourceId int32, operation string, amount int32, capValue int32) error {
			return errors.New("kafka down")
		}
		// must not panic
		comboOrbTryUpdate(l, crusader, comboTestAttack(uint32(skill3.CrusaderPanicSwordId), 0), deps)
	})
}
