package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/party"
	"atlas-channel/skill/effect"
	"atlas-channel/socket/model"
	"context"
	"github.com/sirupsen/logrus"
)

func UseSkillHaste(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
			if effect.HPConsume() > 0 {
				_ = character.ChangeHP(l)(ctx)(worldId, channelId, characterId, -int16(effect.HPConsume()))
			}
			if effect.MPConsume() > 0 {
				_ = character.ChangeMP(l)(ctx)(worldId, channelId, characterId, -int16(effect.MPConsume()))
			}

			_ = buff.Apply(l)(ctx)(worldId, channelId, characterId, info.SkillId(), effect.Duration(), effect.StatUps())

			if info.AffectedPartyMemberBitmap() > 0 && info.AffectedPartyMemberBitmap() < 128 {
				p, err := party.GetByMemberId(l)(ctx)(characterId)
				if err == nil {
					for _, m := range p.Members() {
						if m.Id() != characterId && m.ChannelId() == channelId && m.MapId() == mapId {
							// TODO restrict to those in range, based on bitmap
							_ = buff.Apply(l)(ctx)(worldId, channelId, m.Id(), info.SkillId(), effect.Duration(), effect.StatUps())
						}
					}
				}
			}

			return nil
		}
	}
}
