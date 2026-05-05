package collection

import (
	"context"
	"errors"

	"atlas-monster-book/card"
	"atlas-monster-book/kafka/message"
	"atlas-monster-book/kafka/message/monsterbook"
	"atlas-monster-book/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var entityModelMapper = model.Map(Make)

// Sentinel errors classify SetCoverAndEmit failures so the REST handler can
// map them to 422 (validation) vs 500 (DB).
var (
	ErrCardIdOutOfRange = errors.New("cardId is not a monster-book card item")
	ErrCoverNotOwned    = errors.New("cover requires owned card")
)

// Topic + envelope shape mirror what atlas-channel already consumes at
// services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go
// (handleStatusEventExperienceChanged → announceExperienceGain, lines 238-270).
// The deserializer is character.StatusEvent[character.ExperienceChangedStatusEventBody];
// only the fields the consumer actually reads (CharacterId, Type,
// Body.Distributions[].ExperienceType, Body.Distributions[].Amount) are populated.
const (
	envExperienceTopic                    = "EVENT_TOPIC_CHARACTER_STATUS"
	experienceStatusEventTypeChanged      = "EXPERIENCE_CHANGED"
	experienceDistributionTypeMonsterBook = "MONSTER_BOOK"
)

type experienceDistribution struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

type experienceChangedBody struct {
	Distributions []experienceDistribution `json:"distributions"`
}

type experienceStatusEvent struct {
	CharacterId uint32                `json:"characterId"`
	Type        string                `json:"type"`
	Body        experienceChangedBody `json:"body"`
}

type Processor interface {
	GetByCharacterId(characterId character.Id) (Model, error)
	SetCoverAndEmit(eventId uuid.UUID, characterId character.Id, cardId item.Id) error
	RecomputeAndEmit(mb *message.Buffer) func(characterId character.Id) error
	WithTransaction(tx *gorm.DB) Processor
	DeleteByCharacterId(characterId character.Id) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	cp  card.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		cp:  card.NewProcessor(l, ctx, db),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
		cp:  p.cp.WithTransaction(tx),
	}
}

func (p *ProcessorImpl) GetByCharacterId(characterId character.Id) (Model, error) {
	m, err := entityModelMapper(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db.WithContext(p.ctx)))()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewModelBuilder().
				SetTenantId(p.t.Id()).
				SetCharacterId(characterId).
				SetBookLevel(1).
				Build()
		}
		return Model{}, err
	}
	return m, nil
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId character.Id) error {
	return deleteByCharacter(p.db.WithContext(p.ctx), p.t.Id(), characterId)
}

// computeBookLevel applies the Cosmic monster-book book-level formula.
// Starting from level=0, expToNext=1, increment level and add level*10 to
// expToNext while the running threshold is still <= total. Return the level
// that first failed the condition.
func computeBookLevel(totalUniqueCards uint16) uint16 {
	var level uint16 = 0
	var expToNext uint32 = 1
	for {
		level++
		expToNext += uint32(level) * 10
		if uint32(totalUniqueCards) < expToNext {
			return level
		}
	}
}

// computeExpBonusPercent returns the EXP bonus percentage granted by the
// monster book at the given book level. In Cosmic v83, the bonus equals the
// book level itself (level 7 → +7% party EXP).
func computeExpBonusPercent(bookLevel uint16) uint16 {
	return bookLevel
}

func (p *ProcessorImpl) RecomputeAndEmit(mb *message.Buffer) func(characterId character.Id) error {
	return func(characterId character.Id) error {
		normals, err := p.cp.GetByCharacterIdAndIsSpecial(characterId, false)
		if err != nil {
			return err
		}
		specials, err := p.cp.GetByCharacterIdAndIsSpecial(characterId, true)
		if err != nil {
			return err
		}

		normalCount := uint16(len(normals))
		specialCount := uint16(len(specials))
		total := normalCount + specialCount
		bookLevel := computeBookLevel(total)
		expBonus := computeExpBonusPercent(bookLevel)

		// Defaults are zero-valued for new characters; ignore err.
		prior, _ := p.GetByCharacterId(characterId)

		changed := prior.NormalCount() != normalCount ||
			prior.SpecialCount() != specialCount ||
			prior.BookLevel() != bookLevel ||
			prior.ExpBonusPercent() != expBonus

		if _, err := upsertStats(p.db.WithContext(p.ctx), p.t.Id(), characterId, statsUpdate{
			NormalCount:     normalCount,
			SpecialCount:    specialCount,
			BookLevel:       bookLevel,
			ExpBonusPercent: expBonus,
		}); err != nil {
			return err
		}

		if !changed {
			return nil
		}

		// STATS_CHANGED on monster-book status topic.
		statsEv := monsterbook.StatusEvent[monsterbook.StatsChangedBody]{
			TenantId:    p.t.Id(),
			CharacterId: uint32(characterId),
			EventId:     uuid.New(),
			Type:        monsterbook.StatusEventTypeStatsChanged,
			Body: monsterbook.StatsChangedBody{
				BookLevel:        bookLevel,
				NormalCount:      normalCount,
				SpecialCount:     specialCount,
				TotalUniqueCards: total,
				ExpBonusPercent:  expBonus,
			},
		}
		key := kafkaProducer.CreateKey(int(characterId))
		if err := mb.Put(monsterbook.EnvEventTopicStatus, kafkaProducer.SingleMessageProvider(key, &statsEv)); err != nil {
			return err
		}

		// EXPERIENCE_DISTRIBUTION envelope on the topic atlas-channel already
		// consumes. Subset of character.StatusEvent[ExperienceChangedStatusEventBody]
		// — the consumer only reads CharacterId, Type, and Body.Distributions[].
		expEv := experienceStatusEvent{
			CharacterId: uint32(characterId),
			Type:        experienceStatusEventTypeChanged,
			Body: experienceChangedBody{
				Distributions: []experienceDistribution{{
					ExperienceType: experienceDistributionTypeMonsterBook,
					Amount:         uint32(expBonus),
				}},
			},
		}
		if err := mb.Put(envExperienceTopic, kafkaProducer.SingleMessageProvider(key, &expEv)); err != nil {
			return err
		}
		return nil
	}
}

func (p *ProcessorImpl) SetCoverAndEmit(eventId uuid.UUID, characterId character.Id, cardId item.Id) error {
	// Validate ownership. cardId == 0 is allowed and clears the cover.
	if cardId != 0 {
		if !card.IsCardId(cardId) {
			return ErrCardIdOutOfRange
		}
		owned, err := p.cp.GetByCharacterIdAndCardId(characterId, cardId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCoverNotOwned
			}
			return err
		}
		if owned.Level() < 1 {
			return ErrCoverNotOwned
		}
	}

	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		changed, err := setCover(p.db.WithContext(p.ctx), p.t.Id(), characterId, cardId, eventId)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		ev := monsterbook.StatusEvent[monsterbook.CoverChangedBody]{
			TenantId:    p.t.Id(),
			CharacterId: uint32(characterId),
			EventId:     eventId,
			Type:        monsterbook.StatusEventTypeCoverChanged,
			Body: monsterbook.CoverChangedBody{
				CoverCardId: uint32(cardId),
			},
		}
		key := kafkaProducer.CreateKey(int(characterId))
		return mb.Put(monsterbook.EnvEventTopicStatus, kafkaProducer.SingleMessageProvider(key, &ev))
	})
}
