package wishlist

import (
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/wishlist"
	wishlist2 "atlas-cashshop/kafka/producer/wishlist"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	Add(mb *message.Buffer) func(characterId uint32) func(serialNumber uint32) (Model, error)
	AddAndEmit(characterId uint32, serialNumber uint32) (Model, error)
	Delete(mb *message.Buffer) func(characterId uint32) func(itemId uuid.UUID) error
	DeleteAndEmit(characterId uint32, itemId uuid.UUID) error
	DeleteAll(mb *message.Buffer) func(characterId uint32) error
	DeleteAllAndEmit(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdEntityProvider(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) Add(mb *message.Buffer) func(characterId uint32) func(serialNumber uint32) (Model, error) {
	return func(characterId uint32) func(serialNumber uint32) (Model, error) {
		return func(serialNumber uint32) (Model, error) {
			p.l.Debugf("Character [%d] adding [%d] to their wishlist.", characterId, serialNumber)
			m, err := createEntity(p.db.WithContext(p.ctx), p.t, characterId, serialNumber)
			if err != nil {
				return Model{}, err
			}

			_ = mb.Put(wishlist.EnvEventTopicStatus, wishlist2.AddStatusEventProvider(characterId, serialNumber, m.Id()))
			return m, nil
		}
	}
}

func (p *ProcessorImpl) AddAndEmit(characterId uint32, serialNumber uint32) (Model, error) {
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var err error
		wp := NewProcessor(p.l, p.ctx, tx)
		result, err = message.EmitWithResult[Model, uint32](outbox.EmitProvider(p.l, p.ctx, tx))(model.Flip(wp.Add)(characterId))(serialNumber)
		return err
	})
	return result, txErr
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(characterId uint32) func(itemId uuid.UUID) error {
	return func(characterId uint32) func(itemId uuid.UUID) error {
		return func(itemId uuid.UUID) error {
			p.l.Debugf("Deleting wish list item [%s] for character [%d].", itemId, characterId)
			err := deleteEntity(p.db.WithContext(p.ctx), characterId, itemId)
			if err != nil {
				return err
			}

			_ = mb.Put(wishlist.EnvEventTopicStatus, wishlist2.DeleteStatusEventProvider(characterId, itemId))
			return nil
		}
	}
}

func (p *ProcessorImpl) DeleteAndEmit(characterId uint32, itemId uuid.UUID) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		wp := NewProcessor(p.l, p.ctx, tx)
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(model.Flip(model.Flip(wp.Delete)(characterId))(itemId))
	})
}

func (p *ProcessorImpl) DeleteAll(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		p.l.Debugf("Deleting wish list for character [%d].", characterId)
		err := deleteEntityForCharacter(p.db.WithContext(p.ctx), characterId)
		if err != nil {
			return err
		}

		_ = mb.Put(wishlist.EnvEventTopicStatus, wishlist2.DeleteAllStatusEventProvider(characterId))
		return nil
	}
}

func (p *ProcessorImpl) DeleteAllAndEmit(characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		wp := NewProcessor(p.l, p.ctx, tx)
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(model.Flip(wp.DeleteAll)(characterId))
	})
}
