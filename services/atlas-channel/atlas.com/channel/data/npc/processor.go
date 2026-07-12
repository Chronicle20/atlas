package npc

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ForEachInMap(mapId _map.Id, f model.Operator[Model]) error
	InMapModelProvider(mapId _map.Id) model.Provider[[]Model]
	InMapByObjectIdModelProvider(mapId _map.Id, objectId uint32) model.Provider[[]Model]
	GetInMapByObjectId(mapId _map.Id, objectId uint32) (Model, error)
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ForEachInMap(mapId _map.Id, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(mapId), f, model.ParallelExecute())
}

func (p *ProcessorImpl) InMapModelProvider(mapId _map.Id) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestNPCsInMap(mapId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) InMapByObjectIdModelProvider(mapId _map.Id, objectId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestNPCsInMapByObjectId(mapId, objectId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetInMapByObjectId(mapId _map.Id, objectId uint32) (Model, error) {
	mp := p.InMapByObjectIdModelProvider(mapId, objectId)
	return model.First[Model](mp, model.Filters[Model]())
}
