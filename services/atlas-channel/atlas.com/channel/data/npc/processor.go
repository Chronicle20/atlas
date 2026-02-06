package npc

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ForEachInMap(mapId _map.Id, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(mapId), f, model.ParallelExecute())
}

func (p *Processor) InMapModelProvider(mapId _map.Id) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestNPCsInMap(mapId), Extract, model.Filters[Model]())
}

func (p *Processor) InMapByObjectIdModelProvider(mapId _map.Id, objectId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestNPCsInMapByObjectId(mapId, objectId), Extract, model.Filters[Model]())
}

func (p *Processor) GetInMapByObjectId(mapId _map.Id, objectId uint32) (Model, error) {
	mp := p.InMapByObjectIdModelProvider(mapId, objectId)
	return model.First[Model](mp, model.Filters[Model]())
}
