package tenant

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	AllProvider() model.Provider[[]tenant.Model]
	GetAll() ([]tenant.Model, error)
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

func (p *ProcessorImpl) AllProvider() model.Provider[[]tenant.Model] {
	return requests.SliceProvider[RestModel, tenant.Model](p.l, p.ctx)(requestAll(), Extract, model.Filters[tenant.Model]())
}

func (p *ProcessorImpl) GetAll() ([]tenant.Model, error) {
	return p.AllProvider()()
}
