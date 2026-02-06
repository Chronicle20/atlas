package tenant

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
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

// AllProvider returns a provider for all tenants
func (p *ProcessorImpl) AllProvider() model.Provider[[]tenant.Model] {
	return requests.SliceProvider[RestModel, tenant.Model](p.l, p.ctx)(requestAll(), Extract, model.Filters[tenant.Model]())
}

// GetAll returns all tenants
func (p *ProcessorImpl) GetAll() ([]tenant.Model, error) {
	return p.AllProvider()()
}
