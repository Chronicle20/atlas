package portal

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapByNameModelProvider(mapId _map.Id, name string) model.Provider[[]Model]
	GetInMapByName(mapId _map.Id, name string) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *ProcessorImpl) InMapByNameModelProvider(mapId _map.Id, name string) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMapByName(mapId, name), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetInMapByName(mapId _map.Id, name string) (Model, error) {
	return model.First(p.InMapByNameModelProvider(mapId, name), model.Filters[Model]())
}
