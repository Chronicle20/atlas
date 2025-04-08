package compartment

import (
	"atlas-inventory/asset"
	"atlas-inventory/kafka/message"
	"atlas-inventory/kafka/message/compartment"
	compartment2 "atlas-inventory/kafka/producer/compartment"
	model2 "atlas-inventory/model"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l                logrus.FieldLogger
	ctx              context.Context
	db               *gorm.DB
	assetProcessor   *asset.Processor
	GetById          func(id uuid.UUID) (Model, error)
	GetByCharacterId func(characterId uint32) ([]Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:              l,
		ctx:            ctx,
		db:             db,
		assetProcessor: asset.NewProcessor(l, ctx, db),
	}
	p.GetById = model2.CollapseProvider(p.ByIdProvider)
	p.GetByCharacterId = model2.CollapseProvider(p.ByCharacterIdProvider)
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:   p.l,
		ctx: p.ctx,
		db:  db,
	}
}

func (p *Processor) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	t := tenant.MustFromContext(p.ctx)
	cs, err := model.Map(Make)(getById(t.Id(), id)(p.db))()
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.Map(p.DecorateAsset)(model.FixedProvider(cs))
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(p.ctx)
	cs, err := model.SliceMap(Make)(getByCharacter(t.Id(), characterId)(p.db))(model.ParallelMap())()
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return model.SliceMap(p.DecorateAsset)(model.FixedProvider(cs))(model.ParallelMap())
}

func (p *Processor) DecorateAsset(m Model) (Model, error) {
	as, err := p.assetProcessor.ByCompartmentIdProvider(m.Id(), m.Type())()
	if err != nil {
		return Model{}, err
	}
	return Clone(m).SetAssets(as).Build(), nil
}

func (p *Processor) Create(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, capacity uint32) (Model, error) {
	return func(characterId uint32, inventoryType inventory.Type, capacity uint32) (Model, error) {
		t := tenant.MustFromContext(p.ctx)
		p.l.Debugf("Attempting to create compartment of type [%d] for character [%d] with capacity [%d].", inventoryType, characterId, capacity)
		var c Model
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			var err error
			c, err = create(tx, t.Id(), characterId, inventoryType, capacity)
			if err != nil {
				return err
			}
			return mb.Put(compartment.EnvEventTopicStatus, compartment2.CreatedEventStatusProvider(c.Id(), characterId, c.Type(), c.Capacity()))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		p.l.Debugf("Created compartment [%s] for character [%d] with capacity [%d].", c.Id().String(), characterId, capacity)
		return c, nil
	}
}

func (p *Processor) DeleteByModel(mb *message.Buffer) func(c Model) error {
	return func(c Model) error {
		p.l.Debugf("Attempting to delete compartment [%s].", c.Id().String())
		t := tenant.MustFromContext(p.ctx)
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			err := model.ForEachSlice(model.FixedProvider(c.Assets()), p.assetProcessor.WithTransaction(tx).Delete(mb))
			if err != nil {
				return err
			}
			err = deleteById(tx, t.Id(), c.Id())
			if err != nil {
				return err
			}
			return mb.Put(compartment.EnvEventTopicStatus, compartment2.DeletedEventStatusProvider(c.Id(), c.CharacterId()))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to delete compartment [%s].", c.Id().String())
			return txErr
		}
		p.l.Debugf("Deleted compartment [%s].", c.Id().String())
		return nil
	}
}
