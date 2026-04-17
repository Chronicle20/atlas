package handler

import (
	"atlas-channel/data/skill/effect"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

type Handler func(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, effect effect.Model) error
