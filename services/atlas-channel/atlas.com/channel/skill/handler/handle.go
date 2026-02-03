package handler

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Handler func(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error
