package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/skill/effect"
	"atlas-channel/socket/model"
	"context"
	"github.com/sirupsen/logrus"
)

func UseBeginnerRecovery(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
			if effect.HPConsume() > 0 {
				_ = character.ChangeHP(l)(ctx)(worldId, channelId, characterId, -int16(effect.HPConsume()))
			}
			if effect.MPConsume() > 0 {
				_ = character.ChangeMP(l)(ctx)(worldId, channelId, characterId, -int16(effect.MPConsume()))
			}

			if effect.Cooldown() > 0 {
				_ = skill.ApplyCooldown(l)(ctx)(worldId, channelId, info.SkillId(), effect.Cooldown())(characterId)
			}

			_ = buff.Apply(l)(ctx)(worldId, channelId, characterId, info.SkillId(), effect.Duration(), effect.StatUps())(characterId)
			return nil
		}
	}
}
