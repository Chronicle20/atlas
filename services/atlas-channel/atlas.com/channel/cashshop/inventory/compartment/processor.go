package compartment

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByAccountIdProvider(accountId uint32) model.Provider[[]Model]
	GetByAccountId(accountId uint32) ([]Model, error)
	ByAccountIdAndTypeProvider(accountId uint32, compartmentType CompartmentType) model.Provider[Model]
	GetByAccountIdAndType(accountId uint32, compartmentType CompartmentType) (Model, error)
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

// ByAccountIdProvider returns a provider function that fetches all compartments for an account
func (p *ProcessorImpl) ByAccountIdProvider(accountId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByAccountId(accountId), Extract, model.Filters[Model]())
}

// GetByAccountId retrieves all compartments for an account
func (p *ProcessorImpl) GetByAccountId(accountId uint32) ([]Model, error) {
	return p.ByAccountIdProvider(accountId)()
}

// ByAccountIdAndTypeProvider returns a provider function that fetches a specific compartment by account ID and type
func (p *ProcessorImpl) ByAccountIdAndTypeProvider(accountId uint32, compartmentType CompartmentType) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByAccountIdAndType(accountId, compartmentType), Extract)
}

// GetByAccountIdAndType retrieves a specific compartment by account ID and type
func (p *ProcessorImpl) GetByAccountIdAndType(accountId uint32, compartmentType CompartmentType) (Model, error) {
	return p.ByAccountIdAndTypeProvider(accountId, compartmentType)()
}
