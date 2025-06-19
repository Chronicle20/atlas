package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/model"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func UseEnergyCharge(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
		return func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {

			_ = buff.NewProcessor(l, ctx).Apply(m, characterId, int32(info.SkillId()), effect.Duration(), effect.StatUps())(characterId)

			return nil
		}
	}
}
