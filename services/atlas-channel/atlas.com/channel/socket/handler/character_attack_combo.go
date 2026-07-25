package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	skill2 "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
	buff2 "atlas-channel/kafka/message/buff"
	"context"
	"math/rand"

	"github.com/sirupsen/logrus"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// comboLine is the Combo Attack skill line a character owns: the buff source
// (Combo Attack, adventurer or Cygnus variant) and its Advanced Combo
// upgrade. advLevel is 0 when Advanced Combo isn't learned.
type comboLine struct {
	comboId    skill3.Id
	comboLevel byte
	advId      skill3.Id
	advLevel   byte
}

// comboSkillIds resolves the character's combo line from owned skills.
// ok == false when neither Combo Attack variant is owned at level > 0.
func comboSkillIds(skills []skill.Model) (comboLine, bool) {
	find := func(id skill3.Id) byte {
		for _, s := range skills {
			if s.Id() == id {
				return s.Level()
			}
		}
		return 0
	}
	if lvl := find(skill3.CrusaderComboAttackId); lvl > 0 {
		return comboLine{
			comboId:    skill3.CrusaderComboAttackId,
			comboLevel: lvl,
			advId:      skill3.HeroAdvancedComboAttackId,
			advLevel:   find(skill3.HeroAdvancedComboAttackId),
		}, true
	}
	if lvl := find(skill3.DawnWarriorStage3ComboAttackId); lvl > 0 {
		return comboLine{
			comboId:    skill3.DawnWarriorStage3ComboAttackId,
			comboLevel: lvl,
			advId:      skill3.DawnWarriorStage3AdvancedComboId,
			advLevel:   find(skill3.DawnWarriorStage3AdvancedComboId),
		}, true
	}
	return comboLine{}, false
}

// isComboFinisher reports whether the skill consumes combo orbs.
func isComboFinisher(id skill3.Id) bool {
	switch id {
	case skill3.CrusaderPanicSwordId, skill3.CrusaderPanicAxeId,
		skill3.CrusaderComaSwordId, skill3.CrusaderComaAxeId,
		skill3.DawnWarriorStage3PanicId, skill3.DawnWarriorStage3ComaId:
		return true
	}
	return false
}

// comboGainAmount is the number of orbs one qualifying attack gains: 1, or 2
// when Advanced Combo is learned and its double-orb roll succeeds. Mirrors
// mpEaterShouldProc's prop handling (prop >= 1 always procs).
func comboGainAmount(advLearned bool, prop float64, roll float64) int32 {
	if !advLearned || prop <= 0 {
		return 1
	}
	if prop >= 1.0 || roll < prop {
		return 2
	}
	return 1
}

// comboOrbDeps groups the side-effecting lookups comboOrbTryUpdate needs so
// tests can drive every branch without a real processor or Kafka producer.
type comboOrbDeps struct {
	getEffect  func(skillId uint32, level byte) (effect.Model, error)
	emitUpdate func(sourceId int32, operation string, amount int32, capValue int32) error
	roll       func() float64
}

// comboOrbProductionDeps wires comboOrbDeps to the real effect lookup and the
// buff UPDATE_STAT_VALUE emitter for one attack.
func comboOrbProductionDeps(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) comboOrbDeps {
	bp := buff.NewProcessor(l, ctx)
	return comboOrbDeps{
		getEffect: skill2.NewProcessor(l, ctx).GetEffect,
		emitUpdate: func(sourceId int32, operation string, amount int32, capValue int32) error {
			return bp.UpdateStatValue(f, characterId, sourceId, string(constants.TemporaryStatTypeCombo), operation, amount, capValue)
		},
		roll: rand.Float64,
	}
}

// comboOrbTryUpdate applies Combo Attack orb bookkeeping for one melee
// attack. Finishers consume unconditionally (SET 1 — no hit or orb-count
// requirement, and the attack is never rejected); other attacks gain
// (INCREMENT clamped to the governing effect's x + 1) when at least one
// monster was hit and the skill is not Shout. Whether the COMBO buff is
// actually active is delegated to atlas-buffs, where a missing buff is a
// no-op. All failures are logged and swallowed — the attack pipeline never
// fails on orb bookkeeping.
func comboOrbTryUpdate(l logrus.FieldLogger, c character.Model, ai packetmodel.AttackInfo, deps comboOrbDeps) {
	line, ok := comboSkillIds(c.Skills())
	if !ok {
		return
	}

	sid := skill3.Id(ai.SkillId())
	if isComboFinisher(sid) {
		if err := deps.emitUpdate(int32(line.comboId), buff2.StatOperationSet, 1, 0); err != nil {
			l.WithError(err).Errorf("Combo orbs: consume emit failed for character [%d] finisher [%d].", c.Id(), sid)
		}
		return
	}

	if sid == skill3.CrusaderShoutId || len(ai.DamageInfo()) == 0 {
		return
	}

	effectId, effectLevel := uint32(line.comboId), line.comboLevel
	if line.advLevel > 0 {
		effectId, effectLevel = uint32(line.advId), line.advLevel
	}
	se, err := deps.getEffect(effectId, effectLevel)
	if err != nil {
		l.WithError(err).Errorf("Combo orbs: effect lookup failed for skill [%d] level [%d].", effectId, effectLevel)
		return
	}

	amount := comboGainAmount(line.advLevel > 0, se.Prop(), deps.roll())
	capValue := int32(se.X()) + 1
	if err := deps.emitUpdate(int32(line.comboId), buff2.StatOperationIncrement, amount, capValue); err != nil {
		l.WithError(err).Errorf("Combo orbs: gain emit failed for character [%d].", c.Id())
	}
}
