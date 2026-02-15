package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"atlas-channel/party"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
	model2 "github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func UseSkill(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, characterId uint32, info model.SkillUsageInfo, e effect.Model) error {
	return func(ctx context.Context) func(f field.Model, characterId uint32, info model.SkillUsageInfo, e effect.Model) error {
		return func(f field.Model, characterId uint32, info model.SkillUsageInfo, e effect.Model) error {
			if e.HPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeHP(f, characterId, -int16(e.HPConsume()))
			}
			if e.MPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeMP(f, characterId, -int16(e.MPConsume()))
			}
			if e.Cooldown() > 0 {
				_ = skill.NewProcessor(l, ctx).ApplyCooldown(f, skill2.Id(info.SkillId()), e.Cooldown())(characterId)
			}
			if e.Duration() > 0 && len(e.StatUps()) > 0 {
				applyBuffFunc := buff.NewProcessor(l, ctx).Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps())
				_ = applyBuffFunc(characterId)
				_ = applyToParty(l)(ctx)(f, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)
			}

			// Handle mob-affecting buffs (crash, dispel, etc.)
			applyToMobs(l, ctx, f, characterId, info, e)

			return nil
		}
	}
}

func applyToMobs(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, info model.SkillUsageInfo, e effect.Model) {
	mobIds := info.AffectedMobIds()
	if len(mobIds) == 0 {
		return
	}

	mp := monster.NewProcessor(l, ctx)
	sid := skill2.Id(info.SkillId())

	// Crash and Priest Dispel cancel monster self-buffs
	if isCrashOrDispel(sid) {
		for _, mobId := range mobIds {
			_ = mp.CancelStatus(f, mobId, nil)
		}
	}

	// Apply monster status effects from skill (e.g., crash debuff)
	if len(e.MonsterStatus()) > 0 {
		ms := make(map[string]int32)
		for k, v := range e.MonsterStatus() {
			ms[k] = int32(v)
		}
		for _, mobId := range mobIds {
			_ = mp.ApplyStatus(f, mobId, characterId, uint32(info.SkillId()), uint32(info.SkillLevel()), ms, uint32(e.Duration()))
		}
	}
}

func isCrashOrDispel(skillId skill2.Id) bool {
	return skill2.Is(skillId,
		skill2.CrusaderArmorCrashId,
		skill2.WhiteKnightMagicCrashId,
		skill2.DragonKnightPowerCrashId,
		skill2.PriestDispelId,
	)
}

func applyToParty(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, characterId uint32, memberBitmap byte) func(idOperator model2.Operator[uint32]) error {
	return func(ctx context.Context) func(f field.Model, characterId uint32, memberBitmap byte) func(idOperator model2.Operator[uint32]) error {
		return func(f field.Model, characterId uint32, memberBitmap byte) func(idOperator model2.Operator[uint32]) error {
			return func(idOperator model2.Operator[uint32]) error {
				if memberBitmap > 0 && memberBitmap < 128 {
					p, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
					if err == nil {
						for _, m := range p.Members() {
							// TODO restrict to those in range, based on bitmap
							if m.Id() != characterId && m.ChannelId() == f.ChannelId() && m.MapId() == f.MapId() {
								_ = idOperator(m.Id())
							}
						}
					}
				}
				return nil
			}
		}
	}
}
