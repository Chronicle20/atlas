package handler

import (
	"atlas-channel/skill/effect"
	"atlas-channel/socket/model"
	"context"
	"github.com/sirupsen/logrus"
)

type Handler func(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error
