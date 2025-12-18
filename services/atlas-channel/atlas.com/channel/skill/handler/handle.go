package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/model"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Handler func(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error
