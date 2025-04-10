package inventory

import (
	"atlas-inventory/compartment"
	"atlas-inventory/kafka/message"
	inventory2 "atlas-inventory/kafka/message/inventory"
	"atlas-inventory/kafka/producer"
	inventory3 "atlas-inventory/kafka/producer/inventory"
	model2 "atlas-inventory/model"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l                    logrus.FieldLogger
	ctx                  context.Context
	db                   *gorm.DB
	compartmentProcessor *compartment.Processor
	GetByCharacterId     func(characterId uint32) (Model, error)
	CreateAndEmit        func(characterId uint32) (Model, error)
	DeleteAndEmit        func(characterId uint32) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:                    l,
		ctx:                  ctx,
		db:                   db,
		compartmentProcessor: compartment.NewProcessor(l, ctx, db),
	}
	p.GetByCharacterId = model2.CollapseProvider(p.ByCharacterIdProvider)
	p.CreateAndEmit = message.EmitWithResult[Model, uint32](producer.ProviderImpl(l)(ctx))(p.Create)
	p.DeleteAndEmit = model.Compose(message.Emit(producer.ProviderImpl(l)(ctx)), model.Flip(p.Delete))
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:                    p.l,
		ctx:                  p.ctx,
		db:                   db,
		compartmentProcessor: p.compartmentProcessor,
		GetByCharacterId:     p.GetByCharacterId,
		CreateAndEmit:        p.CreateAndEmit,
		DeleteAndEmit:        p.DeleteAndEmit,
	}
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	b, err := model.Fold(p.compartmentProcessor.ByCharacterIdProvider(characterId), BuilderSupplier(characterId), FoldCompartment)()
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.FixedProvider(b.Build())
}

func (p *Processor) Create(mb *message.Buffer) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		p.l.Debugf("Attempting to create inventory for character [%d].", characterId)
		var i Model
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
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
				c, err = p.compartmentProcessor.WithTransaction(tx).Create(mb)(characterId, it, 24)
				if err != nil {
					return err
				}
				b.SetCompartment(c)
			}
			return mb.Put(inventory2.EnvEventTopicStatus, inventory3.CreatedEventStatusProvider(characterId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to create inventory for character [%d].", characterId)
			return Model{}, txErr
		}
		p.l.Infof("Created inventory for character [%d].", characterId)
		return i, nil
	}
}

func (p *Processor) Delete(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		p.l.Debugf("Attempting to delete inventory for character [%d].", characterId)
		var i Model
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			var err error
			i, err = p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil {
				return err
			}
			err = model.ForEachSlice(model.FixedProvider(i.Compartments()), p.compartmentProcessor.WithTransaction(tx).DeleteByModel(mb), model.ParallelExecute())
			if err != nil {
				return err
			}
			return mb.Put(inventory2.EnvEventTopicStatus, inventory3.DeletedEventStatusProvider(characterId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to delete inventory for character [%d].", characterId)
			return txErr
		}
		p.l.Infof("Deleted inventory for character [%d].", characterId)
		return nil
	}
}
