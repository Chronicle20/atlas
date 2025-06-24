package saga

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for saga processing
type Processor interface {
	GetAll() ([]Saga[any], error)
	AllProvider() model.Provider[[]Saga[any]]
	GetById(transactionId uuid.UUID) (Saga[any], error)
	ByIdProvider(transactionId uuid.UUID) model.Provider[Saga[any]]
	Put(saga Saga[any]) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new saga processor
func NewProcessor(logger logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l:   logger,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

// GetAll returns all sagas for the current tenant
func (p *ProcessorImpl) GetAll() ([]Saga[any], error) {
	return p.AllProvider()()
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Saga[any]] {
	return func() ([]Saga[any], error) {
		return GetCache().GetAll(p.t.Id()), nil
	}
}

// GetById returns a saga by its transaction ID for the current tenant
func (p *ProcessorImpl) GetById(transactionId uuid.UUID) (Saga[any], error) {
	return p.ByIdProvider(transactionId)()
}

func (p *ProcessorImpl) ByIdProvider(transactionId uuid.UUID) model.Provider[Saga[any]] {
	return func() (Saga[any], error) {
		m, ok := GetCache().GetById(p.t.Id(), transactionId)
		if !ok {
			return Saga[any]{}, errors.New("saga not found")
		}
		return m, nil
	}
}

// Put adds or updates a saga in the cache for the current tenant
func (p *ProcessorImpl) Put(saga Saga[any]) error {
	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionID.String(),
		"saga_type":      saga.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Inserting saga into cache")

	GetCache().Put(p.t.Id(), saga)

	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionID.String(),
		"saga_type":      saga.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Saga inserted into cache")

	return nil
}
