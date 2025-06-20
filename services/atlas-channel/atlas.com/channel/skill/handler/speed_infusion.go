package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/model"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func UseSpeedInfusion(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
		return func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
			if effect.HPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeHP(m, characterId, -int16(effect.HPConsume()))
			}
			if effect.MPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeMP(m, characterId, -int16(effect.MPConsume()))
			}

			applyBuffFunc := buff.NewProcessor(l, ctx).Apply(m, characterId, int32(info.SkillId()), effect.Duration(), effect.StatUps())

			_ = applyBuffFunc(characterId)
			_ = applyToParty(l)(ctx)(m, characterId, info.AffectedPartyMemberBitmap())(applyBuffFunc)

			return nil
		}
	}
}
