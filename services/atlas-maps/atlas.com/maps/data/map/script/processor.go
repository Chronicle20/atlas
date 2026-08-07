package script

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetScripts(mapId _map.Id) (Model, error)
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetScripts(mapId _map.Id) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestMapScripts(mapId), Extract)()
}
