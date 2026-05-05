package card

import (
	"context"
	"errors"

	"atlas-monster-book/kafka/message"
	"atlas-monster-book/kafka/message/monsterbook"
	"atlas-monster-book/kafka/producer"

	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var entityModelMapper = model.Map(Make)
var entitySliceMapper = model.SliceMap(Make)

type Processor interface {
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetByCharacterIdAndCardId(characterId uint32, cardId uint32) (Model, error)
	GetByCharacterIdAndIsSpecial(characterId uint32, isSpecial bool) ([]Model, error)
	Add(mb *message.Buffer) func(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error)
	AddAndEmit(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error)
	WithTransaction(tx *gorm.DB) Processor
	DeleteByCharacterId(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: tenant.MustFromContext(ctx)}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: tx, t: p.t}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return entitySliceMapper(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db.WithContext(p.ctx)))()()
}

func (p *ProcessorImpl) GetByCharacterIdAndCardId(characterId uint32, cardId uint32) (Model, error) {
	return entityModelMapper(byCharacterIdAndCardIdEntityProvider(p.t.Id(), characterId, cardId)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByCharacterIdAndIsSpecial(characterId uint32, isSpecial bool) ([]Model, error) {
	return entitySliceMapper(bySpecialEntityProvider(p.t.Id(), characterId, isSpecial)(p.db.WithContext(p.ctx)))()()
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId uint32) error {
	return deleteByCharacter(p.db.WithContext(p.ctx), p.t.Id(), characterId)
}

func (p *ProcessorImpl) Add(mb *message.Buffer) func(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error) {
	return func(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error) {
		if !IsCardId(cardId) {
			return UpsertResult{}, errors.New("cardId out of range")
		}
		res, err := upsertCard(p.db.WithContext(p.ctx), p.t.Id(), characterId, cardId, eventId)
		if err != nil {
			return UpsertResult{}, err
		}
		if res.Duplicate {
			return res, nil
		}
		// Buffer the CARD_ADDED status event. STATS_CHANGED + EXP_DISTRIBUTION are
		// emitted by the collection processor when bookLevel actually changes.
		ev := monsterbook.StatusEvent[monsterbook.CardAddedBody]{
			TenantId:    p.t.Id(),
			CharacterId: characterId,
			EventId:     eventId,
			Type:        monsterbook.StatusEventTypeCardAdded,
			Body: monsterbook.CardAddedBody{
				CardId:   cardId,
				NewLevel: res.NewLevel,
				Full:     res.Full,
			},
		}
		key := kafkaProducer.CreateKey(int(characterId))
		if err := mb.Put(monsterbook.EnvEventTopicStatus, kafkaProducer.SingleMessageProvider(key, &ev)); err != nil {
			return UpsertResult{}, err
		}
		return res, nil
	}
}

func (p *ProcessorImpl) AddAndEmit(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error) {
	var out UpsertResult
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		out, err = p.Add(buf)(eventId, characterId, cardId)
		return err
	})
	return out, err
}
