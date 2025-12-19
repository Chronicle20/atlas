package asset

import (
	"atlas-cashshop/cashshop/item"
	"atlas-cashshop/database"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor provides functions to manipulate assets
type Processor interface {
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	GetById(id uuid.UUID) (Model, error)
	ByCompartmentIdProvider(compartmentId uuid.UUID) model.Provider[[]Model]
	GetByCompartmentId(compartmentId uuid.UUID) ([]Model, error)
	Create(mb *message.Buffer) func(compartmentId uuid.UUID) func(itemId uint32) (Model, error)
	CreateAndEmit(compartmentId uuid.UUID, itemId uint32) (Model, error)
	Release(mb *message.Buffer) func(cashItemId uint32) error
	ReleaseAndEmit(cashItemId uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	p    producer.Provider
	itmP item.Processor
}

// NewProcessor creates a new Processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		p:    producer.ProviderImpl(l)(ctx),
		itmP: item.NewProcessor(l, ctx, db),
	}
	return p
}

// ByIdProvider retrieves an asset by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	ap := model.Map(Make)(getByIdProvider(p.t.Id())(id)(p.db))
	return model.Map(model.Decorate(model.Decorators(p.DecorateItem)))(ap)
}

// GetById retrieves an asset by ID
func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return p.ByIdProvider(id)()
}

// ByCompartmentIdProvider retrieves all assets for a compartment
func (p *ProcessorImpl) ByCompartmentIdProvider(compartmentId uuid.UUID) model.Provider[[]Model] {
	ap := model.SliceMap(Make)(getByCompartmentIdProvider(p.t.Id())(compartmentId)(p.db))(model.ParallelMap())
	return model.SliceMap(model.Decorate(model.Decorators(p.DecorateItem)))(ap)(model.ParallelMap())
}

// GetByCompartmentId retrieves all assets for a compartment
func (p *ProcessorImpl) GetByCompartmentId(compartmentId uuid.UUID) ([]Model, error) {
	return p.ByCompartmentIdProvider(compartmentId)()
}

func (p *ProcessorImpl) DecorateItem(m Model) Model {
	im, err := p.itmP.GetById(m.Item().Id())
	if err != nil {
		return m
	}
	return Clone(m).SetItem(im).Build()
}

// Create creates a new asset
func (p *ProcessorImpl) Create(mb *message.Buffer) func(compartmentId uuid.UUID) func(itemId uint32) (Model, error) {
	return func(compartmentId uuid.UUID) func(itemId uint32) (Model, error) {
		return func(itemId uint32) (Model, error) {
			var result Model

			txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
				// Create the asset entity in the database
				entity, err := create(tx)(p.t.Id())(compartmentId)(itemId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to create asset with compartment ID [%s] and item ID [%d].", compartmentId, itemId)
					return err
				}

				// Create the asset model
				m, err := Make(entity)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to create asset model from entity.")
					return err
				}
				result = p.DecorateItem(m)
				return nil
			})

			if txErr != nil {
				return Model{}, txErr
			}

			return result, nil
		}
	}
}

// CreateAndEmit creates a new asset and emits Kafka messages
func (p *ProcessorImpl) CreateAndEmit(compartmentId uuid.UUID, itemId uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](p.p)(model.Flip(p.Create)(compartmentId))(itemId)
}

func (p *ProcessorImpl) Release(mb *message.Buffer) func(cashItemId uint32) error {
	return func(cashItemId uint32) error {
		p.l.Debugf("Deleting asset with item Id [%d].", cashItemId)
		p.db.Debug()
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			// Delete the asset entity
			err := deleteByItemId(tx, p.t.Id(), cashItemId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to delete asset entity item [%d].", cashItemId)
				return err
			}
			return nil
		})

		if txErr != nil {
			return txErr
		}

		return nil
	}
}

func (p *ProcessorImpl) ReleaseAndEmit(cashItemId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Release(buf)(cashItemId)
	})
}
