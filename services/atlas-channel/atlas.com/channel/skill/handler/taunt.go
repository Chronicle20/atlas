package handler

import (
	"atlas-channel/character"
	"atlas-channel/skill/effect"
	"atlas-channel/socket/model"
	"context"
	"github.com/sirupsen/logrus"
)

func UseTaunt(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
			if effect.HPConsume() > 0 {
				_ = character.ChangeHP(l)(ctx)(worldId, channelId, characterId, -int16(effect.HPConsume()))
			}
			if effect.MPConsume() > 0 {
				_ = character.ChangeMP(l)(ctx)(worldId, channelId, characterId, -int16(effect.MPConsume()))
			}

			return nil
		}
	}
}
