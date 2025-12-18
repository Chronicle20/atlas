package _map

import (
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	CharacterIdsInMapProvider(worldId world.Id, channelId channel.Id, mapId _map2.Id) model.Provider[[]uint32]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *ProcessorImpl) CharacterIdsInMapProvider(worldId world.Id, channelId channel.Id, mapId _map2.Id) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMap(worldId, channelId, mapId), Extract, model.Filters[uint32]())
}
