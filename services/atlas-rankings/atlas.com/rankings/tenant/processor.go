package tenant

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor defines the interface for tenant operations
type Processor interface {
	// AllProvider returns a provider for all tenants
	AllProvider() model.Provider[[]tenant.Model]

	// GetAll returns all tenants
	GetAll() ([]tenant.Model, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new processor implementation
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// AllProvider fetches every tenant. atlas-tenants' GET /tenants is now
// paginated (task-117); this drives the startup per-tenant route-config
// load and ticker loop in main.go, a genuine semantic-all consumer, so it
// drains every page rather than fetching just the first.
func (p *ProcessorImpl) AllProvider() model.Provider[[]tenant.Model] {
	return requests.DrainProvider[RestModel, tenant.Model](p.l, p.ctx)(allTenantsUrl(), 250, Extract, model.Filters[tenant.Model]())
}

// GetAll returns all tenants
func (p *ProcessorImpl) GetAll() ([]tenant.Model, error) {
	return p.AllProvider()()
}
