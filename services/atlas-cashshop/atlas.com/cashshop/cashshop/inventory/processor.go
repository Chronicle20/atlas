package inventory

import (
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message"
	inventoryMsg "atlas-cashshop/kafka/message/cashshop/inventory"
	"atlas-cashshop/kafka/producer"
	inventory2 "atlas-cashshop/kafka/producer/cashshop/inventory"
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	ccp compartment.Processor
}

// NewProcessor creates a new Processor instance
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		ccp: compartment.NewProcessor(l, ctx, db),
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
		ccp: p.ccp,
	}
}

// ByAccountIdProvider returns a provider for retrieving an inventory by account ID
func (p *ProcessorImpl) ByAccountIdProvider(accountId uint32) model.Provider[Model] {
	return func() (Model, error) {
		// Get all compartments for the account
		compartments, err := p.ccp.GetByAccountId(accountId)
		if err != nil {
			return Model{}, err
		}

		// Create the inventory builder
		builder := NewBuilder(accountId)

		// For each compartment, get its assets and add them to the compartment
		for _, c := range compartments {
			builder.SetCompartment(c)
		}

		// Build and return the inventory
		return builder.Build(), nil
	}
}

// GetByAccountId retrieves an inventory by account ID
func (p *ProcessorImpl) GetByAccountId(accountId uint32) (Model, error) {
	return p.ByAccountIdProvider(accountId)()
}

// createDefaultCompartments creates the default compartments for an account,
// buffering their status events into mb so they enqueue atomically with the
// rest of the caller's transaction.
func (p *ProcessorImpl) createDefaultCompartments(mb *message.Buffer, accountId uint32) (Model, error) {
	p.l.Debugf("Creating default compartments for account [%d] with capacity [%d].", accountId, compartment.DefaultCapacity)

	// Create a compartment processor bound to the same db/tx as p
	compartmentProcessor := compartment.NewProcessor(p.l, p.ctx, p.db)

	// Create Explorer compartment
	explorerCompartment, err := compartmentProcessor.Create(mb)(accountId)(compartment.TypeExplorer)(compartment.DefaultCapacity)
	if err != nil {
		p.l.WithError(err).Errorf("Could not create Explorer compartment for account [%d].", accountId)
		return Model{}, err
	}

	// Create Cygnus compartment
	cygnusCompartment, err := compartmentProcessor.Create(mb)(accountId)(compartment.TypeCygnus)(compartment.DefaultCapacity)
	if err != nil {
		p.l.WithError(err).Errorf("Could not create Cygnus compartment for account [%d].", accountId)
		return Model{}, err
	}

	// Create Legend compartment
	legendCompartment, err := compartmentProcessor.Create(mb)(accountId)(compartment.TypeLegend)(compartment.DefaultCapacity)
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
		p.l.Debugf("Initializing cash shop inventory for account [%d] with capacity [%d].", accountId, compartment.DefaultCapacity)

		// Create default compartments
		inventory, err := p.createDefaultCompartments(mb, accountId)
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
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			result, err = p.WithTransaction(tx).Create(buf)(accountId)
			return err
		})
	})
	return result, txErr
}

// Delete deletes an inventory and all its compartments and assets
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(accountId uint32) error {
	return func(accountId uint32) error {
		p.l.Debugf("Account [%d] was deleted. Cleaning up cash shop inventory...", accountId)

		// Create a compartment processor bound to the same db/tx as p
		compartmentProcessor := compartment.NewProcessor(p.l, p.ctx, p.db)

		// Delete all compartments for the account, buffering their events into mb
		err := compartmentProcessor.DeleteAllByAccountId(mb)(accountId)
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
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).Delete(buf)(accountId)
		})
	})
}
