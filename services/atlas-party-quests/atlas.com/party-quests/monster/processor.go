package monster

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	DestroyInField(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) DestroyInField(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) error {
	return requestDestroyInField(worldId, channelId, mapId, instance)(p.l, p.ctx)
}
