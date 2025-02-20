package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/skill/effect"
	"atlas-channel/socket/model"
	"context"
	"github.com/sirupsen/logrus"
)

func UseShadowMeso(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {

			_ = buff.Apply(l)(ctx)(worldId, channelId, characterId, info.SkillId(), effect.Duration(), effect.StatUps())(characterId)

			return nil
		}
	}
}
