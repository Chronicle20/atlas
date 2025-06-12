package inventory

import (
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/item"
	"atlas-cashshop/kafka/message"
	inventoryMsg "atlas-cashshop/kafka/message/cashshop/inventory"
	"atlas-cashshop/kafka/producer"
	inventory2 "atlas-cashshop/kafka/producer/cashshop/inventory"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor interface defines the operations for cash shop inventory
type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	ByAccountIdProvider(accountId uint32) model.Provider[Model]
	GetByAccountId(accountId uint32) (Model, error)
	Create(mb *message.Buffer) func(accountId uint32) (Model, error)
	CreateAndEmit(accountId uint32) (Model, error)
	Delete(mb *message.Buffer) func(accountId uint32) error
	DeleteAndEmit(accountId uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
}

// NewProcessor creates a new Processor instance
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
	return p
}

// WithTransaction returns a new Processor with the given transaction
func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
		p:   p.p,
	}
}

// ByAccountIdProvider returns a provider for retrieving an inventory by account ID
func (p *ProcessorImpl) ByAccountIdProvider(accountId uint32) model.Provider[Model] {
	return ByAccountIdProvider(p.t.Id())(accountId)(func(itemId uint32) model.Provider[item.Model] {
		return func() (item.Model, error) {
			return item.Model{}, nil
		}
	})(p.db)
}

// GetByAccountId retrieves an inventory by account ID
func (p *ProcessorImpl) GetByAccountId(accountId uint32) (Model, error) {
	return p.ByAccountIdProvider(accountId)()
}

// createDefaultCompartments creates the default compartments for an account
func (p *ProcessorImpl) createDefaultCompartments(accountId uint32) (Model, error) {
	p.l.Debugf("Creating default compartments for account [%d] with capacity [%d].", accountId, compartment.DefaultCapacity)

	// Create a compartment processor
	compartmentProcessor := compartment.NewProcessor(p.l, p.ctx, p.db)

	// Create Explorer compartment
	explorerCompartment, err := compartmentProcessor.CreateAndEmit(accountId, compartment.TypeExplorer, compartment.DefaultCapacity)
	if err != nil {
		p.l.WithError(err).Errorf("Could not create Explorer compartment for account [%d].", accountId)
		return Model{}, err
	}

	// Create Cygnus compartment
	cygnusCompartment, err := compartmentProcessor.CreateAndEmit(accountId, compartment.TypeCygnus, compartment.DefaultCapacity)
	if err != nil {
		p.l.WithError(err).Errorf("Could not create Cygnus compartment for account [%d].", accountId)
		return Model{}, err
	}

	// Create Legend compartment
	legendCompartment, err := compartmentProcessor.CreateAndEmit(accountId, compartment.TypeLegend, compartment.DefaultCapacity)
	if err != nil {
		p.l.WithError(err).Errorf("Could not create Legend compartment for account [%d].", accountId)
		return Model{}, err
	}

	// Build the inventory model
	builder := NewBuilder(accountId)
	builder.SetCompartment(explorerCompartment)
	builder.SetCompartment(cygnusCompartment)
	builder.SetCompartment(legendCompartment)

	return builder.Build(), nil
}

// Create creates a new inventory with default compartments
func (p *ProcessorImpl) Create(mb *message.Buffer) func(accountId uint32) (Model, error) {
	return func(accountId uint32) (Model, error) {
		p.l.Debugf("Initializing cash shop inventory for account [%d] with capacity [%d].", accountId)

		// Create default compartments
		inventory, err := p.createDefaultCompartments(accountId)
		if err != nil {
			return Model{}, err
		}

		// Add message to buffer
		_ = mb.Put(inventoryMsg.EnvEventTopicStatus, inventory2.CreateStatusEventProvider(accountId))

		return inventory, nil
	}
}

// CreateAndEmit creates a new inventory and emits an event
func (p *ProcessorImpl) CreateAndEmit(accountId uint32) (Model, error) {
	mb := message.NewBuffer()
	m, err := p.Create(mb)(accountId)
	if err != nil {
		return Model{}, err
	}

	for t, ms := range mb.GetAll() {
		if err = p.p(t)(model.FixedProvider(ms)); err != nil {
			return Model{}, err
		}
	}

	return m, nil
}

// Delete deletes an inventory and all its compartments and assets
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(accountId uint32) error {
	return func(accountId uint32) error {
		p.l.Debugf("Account [%d] was deleted. Cleaning up cash shop inventory...", accountId)

		// Create a compartment processor
		compartmentProcessor := compartment.NewProcessor(p.l, p.ctx, p.db)

		// Delete all compartments for the account
		err := compartmentProcessor.DeleteAllByAccountIdAndEmit(accountId)
		if err != nil {
			return err
		}

		// Add message to buffer
		_ = mb.Put(inventoryMsg.EnvEventTopicStatus, inventory2.DeleteStatusEventProvider(accountId))
		return nil
	}
}

// DeleteAndEmit deletes an inventory and emits an event
func (p *ProcessorImpl) DeleteAndEmit(accountId uint32) error {
	mb := message.NewBuffer()
	err := p.Delete(mb)(accountId)
	if err != nil {
		return err
	}

	for t, ms := range mb.GetAll() {
		if err = p.p(t)(model.FixedProvider(ms)); err != nil {
			return err
		}
	}

	return nil
}
