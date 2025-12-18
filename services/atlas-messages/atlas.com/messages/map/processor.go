package _map

import (
	"atlas-messages/data/map"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"strconv"
)

type Processor interface {
	Exists(mapId uint32) bool
	CharacterIdsInMapStringProvider(worldId byte, channelId byte, mapStr string) model.Provider[[]uint32]
	CharacterIdsInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	dp  _map.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		dp:  _map.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) Exists(mapId uint32) bool {
	_, err := p.dp.GetById(mapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find requested map [%d].", mapId)
		return false
	}
	return true
}

func (p *ProcessorImpl) CharacterIdsInMapStringProvider(worldId byte, channelId byte, mapStr string) model.Provider[[]uint32] {
	mapId, err := strconv.ParseUint(mapStr, 10, 32)
	if err != nil {
		return model.ErrorProvider[[]uint32](err)
	}
	return p.CharacterIdsInMapProvider(worldId, channelId, uint32(mapId))
}

func (p *ProcessorImpl) CharacterIdsInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMap(worldId, channelId, mapId), Extract, model.Filters[uint32]())
}
