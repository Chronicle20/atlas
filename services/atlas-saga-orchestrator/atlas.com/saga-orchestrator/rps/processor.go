package rps

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	StartGame(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) StartGame(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (RestModel, error) {
	return StartGame(p.l, p.ctx)(characterId, worldId, channelId, npcId)
}
