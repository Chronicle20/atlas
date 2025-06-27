package portal

import (
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapByIdModelProvider(mapId _map.Id, id uint32) model.Provider[Model]
	GetInMapById(mapId _map.Id, id uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) InMapByIdModelProvider(mapId _map.Id, id uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestInMapById(mapId, id), Extract)
}

func (p *ProcessorImpl) GetInMapById(mapId _map.Id, id uint32) (Model, error) {
	return p.InMapByIdModelProvider(mapId, id)()
}
