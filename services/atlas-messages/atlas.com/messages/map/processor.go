package _map

import (
	"atlas-messages/data/map"
	"context"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Exists(mapId _map2.Id) bool
	CharacterIdsInMapStringProvider(ch channel.Model, mapStr string) model.Provider[[]uint32]
	CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32]
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

func (p *ProcessorImpl) Exists(mapId _map2.Id) bool {
	_, err := p.dp.GetById(mapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find requested map [%d].", mapId)
		return false
	}
	return true
}

func (p *ProcessorImpl) CharacterIdsInMapStringProvider(ch channel.Model, mapStr string) model.Provider[[]uint32] {
	mapId, err := strconv.ParseUint(mapStr, 10, 32)
	if err != nil {
		return model.ErrorProvider[[]uint32](err)
	}
	f := field.NewBuilder(ch.WorldId(), ch.Id(), _map2.Id(mapId)).Build()
	return p.CharacterIdsInFieldProvider(f)
}

func (p *ProcessorImpl) CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMap(f), Extract, model.Filters[uint32]())
}
