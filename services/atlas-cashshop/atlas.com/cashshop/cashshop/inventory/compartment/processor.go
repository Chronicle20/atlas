package compartment

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/compartment"
	"atlas-cashshop/kafka/producer"
	compartmentProducer "atlas-cashshop/kafka/producer/cashshop/inventory/compartment"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const DefaultCapacity = uint32(55)

// Processor interface defines the operations for cash shop inventory compartments
type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	GetByAccountIdAndType(accountId uint32, type_ CompartmentType) (Model, error)
	ByAccountIdAndTypeProvider(accountId uint32, type_ CompartmentType) model.Provider[Model]
	AllByAccountIdProvider(accountId uint32) model.Provider[[]Model]
	GetByAccountId(accountId uint32) ([]Model, error)
	Create(mb *message.Buffer) func(accountId uint32) func(type_ CompartmentType) func(capacity uint32) (Model, error)
	CreateAndEmit(accountId uint32, type_ CompartmentType, capacity uint32) (Model, error)
	UpdateCapacity(mb *message.Buffer) func(id uuid.UUID) func(capacity uint32) (Model, error)
	UpdateCapacityAndEmit(id uuid.UUID, capacity uint32) (Model, error)
	Delete(mb *message.Buffer) func(id uuid.UUID) error
	DeleteAndEmit(id uuid.UUID) error
	DeleteAllByAccountId(mb *message.Buffer) func(accountId uint32) error
	DeleteAllByAccountIdAndEmit(accountId uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
	cap asset.Processor
}

// NewProcessor creates a new Processor instance
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		cap: asset.NewProcessor(l, ctx, db),
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
		cap: p.cap,
	}
}

func (p *ProcessorImpl) DecorateAssets(m Model) Model {
	// Get all assets for this compartment
	assets, err := p.cap.GetByCompartmentId(m.Id())
	if err != nil {
		return m
	}
	return Clone(m).SetAssets(assets).Build()
}

// ByIdProvider returns a provider for retrieving a compartment by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	cp := model.Map[Entity, Model](Make)(getByIdProvider(p.t.Id())(id)(p.db))
	return model.Map(model.Decorate(model.Decorators(p.DecorateAssets)))(cp)
}

// ByAccountIdAndTypeProvider returns a provider for retrieving a compartment by account ID and type
func (p *ProcessorImpl) ByAccountIdAndTypeProvider(accountId uint32, type_ CompartmentType) model.Provider[Model] {
	cp := model.Map[Entity, Model](Make)(getByAccountIdAndTypeProvider(p.t.Id())(accountId)(type_)(p.db))
	return model.Map(model.Decorate(model.Decorators(p.DecorateAssets)))(cp)
}

func (p *ProcessorImpl) GetByAccountIdAndType(accountId uint32, type_ CompartmentType) (Model, error) {
	return p.ByAccountIdAndTypeProvider(accountId, type_)()
}

// AllByAccountIdProvider returns a provider for retrieving all compartments for an account
func (p *ProcessorImpl) AllByAccountIdProvider(accountId uint32) model.Provider[[]Model] {
	cp := model.SliceMap[Entity, Model](Make)(getAllByAccountIdProvider(p.t.Id())(accountId)(p.db))(model.ParallelMap())
	return model.SliceMap(model.Decorate(model.Decorators(p.DecorateAssets)))(cp)(model.ParallelMap())
}

func (p *ProcessorImpl) GetByAccountId(accountId uint32) ([]Model, error) {
	return p.AllByAccountIdProvider(accountId)()
}

// Create creates a new compartment
func (p *ProcessorImpl) Create(mb *message.Buffer) func(accountId uint32) func(type_ CompartmentType) func(capacity uint32) (Model, error) {
	return func(accountId uint32) func(type_ CompartmentType) func(capacity uint32) (Model, error) {
		return func(type_ CompartmentType) func(capacity uint32) (Model, error) {
			return func(capacity uint32) (Model, error) {
				p.l.Debugf("Creating compartment for account [%d] with type [%s] and capacity [%d].", accountId, type_, capacity)

				// Create the compartment
				model, err := createEntity(p.db, p.t, accountId, type_, capacity)
				if err != nil {
					p.l.WithError(err).Errorf("Could not create compartment for account [%d].", accountId)
					return Model{}, err
				}

				// Add message to buffer
				_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.CreateStatusEventProvider(model.Id().String(), string(type_), capacity))

				return model, nil
			}
		}
	}
}

// CreateAndEmit creates a new compartment and emits an event
func (p *ProcessorImpl) CreateAndEmit(accountId uint32, type_ CompartmentType, capacity uint32) (Model, error) {
	mb := message.NewBuffer()
	m, err := p.Create(mb)(accountId)(type_)(capacity)
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

// UpdateCapacity updates the capacity of a compartment
func (p *ProcessorImpl) UpdateCapacity(mb *message.Buffer) func(id uuid.UUID) func(capacity uint32) (Model, error) {
	return func(id uuid.UUID) func(capacity uint32) (Model, error) {
		return func(capacity uint32) (Model, error) {
			p.l.Debugf("Updating capacity of compartment [%s] to [%d].", id, capacity)

			// Update the compartment
			model, err := updateCapacity(p.db, p.t.Id(), id, capacity)
			if err != nil {
				p.l.WithError(err).Errorf("Could not update capacity of compartment [%s].", id)
				return Model{}, err
			}

			// Add message to buffer
			_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.UpdateStatusEventProvider(id.String(), string(model.Type()), capacity))

			return model, nil
		}
	}
}

// UpdateCapacityAndEmit updates the capacity of a compartment and emits an event
func (p *ProcessorImpl) UpdateCapacityAndEmit(id uuid.UUID, capacity uint32) (Model, error) {
	mb := message.NewBuffer()
	m, err := p.UpdateCapacity(mb)(id)(capacity)
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

// Delete deletes a compartment
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		p.l.Debugf("Deleting compartment [%s].", id)

		// Get the compartment to get the account ID
		model, err := p.ByIdProvider(id)()
		if err != nil {
			p.l.WithError(err).Errorf("Could not find compartment [%s] to delete.", id)
			return err
		}

		// Delete the compartment
		err = deleteEntity(p.db, p.t.Id(), id)
		if err != nil {
			p.l.WithError(err).Errorf("Could not delete compartment [%s].", id)
			return err
		}

		// Add message to buffer
		_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.DeleteStatusEventProvider(id.String(), string(model.Type())))

		return nil
	}
}

// DeleteAndEmit deletes a compartment and emits an event
func (p *ProcessorImpl) DeleteAndEmit(id uuid.UUID) error {
	mb := message.NewBuffer()
	err := p.Delete(mb)(id)
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

// DeleteAllByAccountId deletes all compartments for an account
func (p *ProcessorImpl) DeleteAllByAccountId(mb *message.Buffer) func(accountId uint32) error {
	return func(accountId uint32) error {
		p.l.Debugf("Deleting all compartments for account [%d].", accountId)

		// Delete all compartments
		err := deleteAllByAccountId(p.db, p.t.Id(), accountId)
		if err != nil {
			p.l.WithError(err).Errorf("Could not delete compartments for account [%d].", accountId)
			return err
		}

		// Add message to buffer
		// For DeleteAllByAccountId, we don't have compartment IDs, so we'll emit an inventory deletion event instead
		_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.DeleteStatusEventProvider("all", "all"))

		return nil
	}
}

// DeleteAllByAccountIdAndEmit deletes all compartments for an account and emits an event
func (p *ProcessorImpl) DeleteAllByAccountIdAndEmit(accountId uint32) error {
	mb := message.NewBuffer()
	err := p.DeleteAllByAccountId(mb)(accountId)
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
