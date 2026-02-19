package inventory

import (
	"atlas-inventory/compartment"
	database "github.com/Chronicle20/atlas-database"
	"atlas-inventory/kafka/message"
	inventory2 "atlas-inventory/kafka/message/inventory"
	"atlas-inventory/kafka/producer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithTransaction(db *gorm.DB) Processor
	GetByCharacterId(characterId uint32) (Model, error)
	ByCharacterIdProvider(characterId uint32) model.Provider[Model]
	CreateAndEmit(transactionId uuid.UUID, characterId uint32) (Model, error)
	Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) (Model, error)
	DeleteAndEmit(transactionId uuid.UUID, characterId uint32) error
	Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error
}

type ProcessorImpl struct {
	l                    logrus.FieldLogger
	ctx                  context.Context
	db                   *gorm.DB
	compartmentProcessor *compartment.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:                    l,
		ctx:                  ctx,
		db:                   db,
		compartmentProcessor: compartment.NewProcessor(l, ctx, db),
	}
	return p
}

func (p *ProcessorImpl) WithTransaction(db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:                    p.l,
		ctx:                  p.ctx,
		db:                   db,
		compartmentProcessor: p.compartmentProcessor,
	}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	b, err := model.Fold(p.compartmentProcessor.ByCharacterIdProvider(characterId), BuilderSupplier(characterId), FoldCompartment)()
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.FixedProvider(b.Build())
}

func (p *ProcessorImpl) CreateAndEmit(transactionId uuid.UUID, characterId uint32) (Model, error) {
	var m Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		m, err = p.Create(buf)(transactionId, characterId)
		return err
	})
	return m, err
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32) (Model, error) {
		p.l.Debugf("Attempting to create inventory for character [%d].", characterId)
		var i Model
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			// Check if inventory already exists for character.
			var err error
			i, err = p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if i.Equipable().Capacity() != 0 {
				return errors.New("already exists")
			}

			// Generate inventory model by creating new compartments.
			b := NewBuilder(characterId)
			for _, it := range inventory.Types {
				var c compartment.Model
				c, err = p.compartmentProcessor.WithTransaction(tx).Create(mb)(transactionId, characterId, it, 24)
				if err != nil {
					return err
				}
				b.SetCompartment(c)
			}
			i = b.Build()
			return mb.Put(inventory2.EnvEventTopicStatus, CreatedEventStatusProvider(characterId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to create inventory for character [%d].", characterId)
			return Model{}, txErr
		}
		p.l.Infof("Created inventory for character [%d].", characterId)
		return i, nil
	}
}

func (p *ProcessorImpl) DeleteAndEmit(transactionId uuid.UUID, characterId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.Delete(buf)(transactionId, characterId)
	})
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32) error {
		p.l.Debugf("Attempting to delete inventory for character [%d].", characterId)
		var i Model
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			var err error
			i, err = p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil {
				return err
			}
			err = model.ForEachSlice(model.FixedProvider(i.Compartments()), func(c compartment.Model) error {
				return p.compartmentProcessor.WithTransaction(tx).DeleteByModel(mb)(transactionId, c)
			}, model.ParallelExecute())
			if err != nil {
				return err
			}
			return mb.Put(inventory2.EnvEventTopicStatus, DeletedEventStatusProvider(characterId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to delete inventory for character [%d].", characterId)
			return txErr
		}
		p.l.Infof("Deleted inventory for character [%d].", characterId)
		return nil
	}
}
