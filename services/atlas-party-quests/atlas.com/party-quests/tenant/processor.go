package tenant

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

var _ Processor = (*ProcessorImpl)(nil)

// AllProvider fetches every tenant. atlas-tenants' GET /tenants is now
// paginated (task-117); this drives the startup per-tenant PQ timer ticker
// and graceful-shutdown sweep in main.go, a genuine semantic-all consumer,
// so it drains every page rather than fetching just the first.
func (p *ProcessorImpl) AllProvider() model.Provider[[]tenant.Model] {
	return requests.DrainProvider[RestModel, tenant.Model](p.l, p.ctx)(allTenantsUrl(), 250, Extract, model.Filters[tenant.Model]())
}

func (p *ProcessorImpl) GetAll() ([]tenant.Model, error) {
	return p.AllProvider()()
}
