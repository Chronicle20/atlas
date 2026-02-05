package reactor

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	InMapProvider(mapId _map.Id) model.Provider[[]Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) InMapProvider(mapId _map.Id) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestReactors(mapId), Extract, model.Filters[Model]())
}
