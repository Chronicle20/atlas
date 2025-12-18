package equipment

import (
	"atlas-drops/kafka/message"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for equipment processing operations
type Processor interface {
	// Create creates a new equipment item
	Create(mb *message.Buffer) func(itemId uint32) model.Provider[Model]
	// CreateAndEmit creates a new equipment item and emits a Kafka message
	CreateAndEmit(itemId uint32) model.Provider[Model]

	// Delete deletes an equipment item
	Delete(mb *message.Buffer) func(equipmentId uint32) error
	// DeleteAndEmit deletes an equipment item and emits a Kafka message
	DeleteAndEmit(equipmentId uint32) error

	// GetById gets an equipment item by ID
	GetById(equipmentId uint32) (Model, error)
	// ByIdProvider provides an equipment item by ID
	ByIdProvider(equipmentId uint32) model.Provider[Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new equipment processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
	}
}

// Create creates a new equipment item
func (p *ProcessorImpl) Create(mb *message.Buffer) func(itemId uint32) model.Provider[Model] {
	return func(itemId uint32) model.Provider[Model] {
		ro, err := requestCreate(itemId)(p.l, p.ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Generating equipment item %d, they were not awarded this item. Check request in ESO service.", itemId)
			return model.ErrorProvider[Model](err)
		}
		// No Kafka message to emit for equipment creation
		return model.Map(Extract)(model.FixedProvider(ro))
	}
}

// CreateAndEmit creates a new equipment item and emits a Kafka message
func (p *ProcessorImpl) CreateAndEmit(itemId uint32) model.Provider[Model] {
	// Since there's no Kafka message to emit for equipment creation, we can just call Create
	return p.Create(message.NewBuffer())(itemId)
}

// Delete deletes an equipment item
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(equipmentId uint32) error {
	return func(equipmentId uint32) error {
		err := deleteById(equipmentId)(p.l, p.ctx)
		// No Kafka message to emit for equipment deletion
		return err
	}
}

// DeleteAndEmit deletes an equipment item and emits a Kafka message
func (p *ProcessorImpl) DeleteAndEmit(equipmentId uint32) error {
	// Since there's no Kafka message to emit for equipment deletion, we can just call Delete
	return p.Delete(message.NewBuffer())(equipmentId)
}

// GetById gets an equipment item by ID
func (p *ProcessorImpl) GetById(equipmentId uint32) (Model, error) {
	return p.ByIdProvider(equipmentId)()
}

// ByIdProvider provides an equipment item by ID
func (p *ProcessorImpl) ByIdProvider(equipmentId uint32) model.Provider[Model] {
	req := requestById(equipmentId)
	return func() (Model, error) {
		rm, err := req(p.l, p.ctx)
		if err != nil {
			return Model{}, err
		}
		return Extract(rm)
	}
}

// Create creates a new equipment item
func Create(l logrus.FieldLogger) func(ctx context.Context) func(itemId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(itemId uint32) model.Provider[Model] {
		p := NewProcessor(l, ctx)
		return func(itemId uint32) model.Provider[Model] {
			return p.CreateAndEmit(itemId)
		}
	}
}

// Delete deletes an equipment item
func Delete(l logrus.FieldLogger) func(ctx context.Context) func(equipmentId uint32) error {
	return func(ctx context.Context) func(equipmentId uint32) error {
		p := NewProcessor(l, ctx)
		return func(equipmentId uint32) error {
			return p.DeleteAndEmit(equipmentId)
		}
	}
}
