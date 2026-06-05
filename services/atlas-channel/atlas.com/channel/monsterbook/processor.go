package monsterbook

import (
	mbmsg "atlas-channel/kafka/message/monsterbook"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// Collection is the immutable domain representation of a character's monster
// book collection summary.
type Collection struct {
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	totalUniqueCards uint16
	coverCardId      item.Id
	coverMonsterId   monster.Id
	expBonusPercent  uint16
}

func (c Collection) BookLevel() uint16          { return c.bookLevel }
func (c Collection) NormalCount() uint16        { return c.normalCount }
func (c Collection) SpecialCount() uint16       { return c.specialCount }
func (c Collection) TotalUniqueCards() uint16   { return c.totalUniqueCards }
func (c Collection) CoverCardId() item.Id       { return c.coverCardId }
func (c Collection) CoverMonsterId() monster.Id { return c.coverMonsterId }
func (c Collection) ExpBonusPercent() uint16    { return c.expBonusPercent }

// Card is the immutable domain representation of a single owned monster-book card.
type Card struct {
	cardId    item.Id
	level     uint8
	isSpecial bool
}

func (c Card) CardId() item.Id { return c.cardId }
func (c Card) Level() uint8    { return c.level }
func (c Card) IsSpecial() bool { return c.isSpecial }

// Processor exposes monster book emissions and reads from atlas-channel.
type Processor interface {
	RequestSetCover(characterId character.Id, coverCardId item.Id) error
	ByCharacterIdProvider(characterId character.Id) model.Provider[Collection]
	GetByCharacterId(characterId character.Id) (Collection, error)
	GetCardsByCharacterId(characterId character.Id) ([]Card, error)
	CardsByCharacterIdProvider(characterId character.Id) model.Provider[[]Card]
}

// ProcessorImpl emits SET_COVER commands to the monster book service and
// fetches collection summaries via REST.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor builds a Processor bound to the request context's tenant.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

// RequestSetCover emits a SET_COVER command keyed on the character.
func (p *ProcessorImpl) RequestSetCover(characterId character.Id, coverCardId item.Id) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mbmsg.EnvCommandTopic)(SetCoverCommandProvider(p.t.Id(), characterId, coverCardId))
}

// ByCharacterIdProvider returns a provider that fetches the character's
// monster book collection from atlas-monster-book.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId character.Id) model.Provider[Collection] {
	return requests.Provider[CollectionRestModel, Collection](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
}

// GetByCharacterId fetches and returns the monster book collection for the
// given character.
func (p *ProcessorImpl) GetByCharacterId(characterId character.Id) (Collection, error) {
	return p.ByCharacterIdProvider(characterId)()
}

// CardsByCharacterIdProvider returns a provider that fetches the character's
// owned monster-book cards from atlas-monster-book.
func (p *ProcessorImpl) CardsByCharacterIdProvider(characterId character.Id) model.Provider[[]Card] {
	return requests.SliceProvider[CardRestModel, Card](p.l, p.ctx)(requestCardsByCharacterId(characterId), ExtractCard, model.Filters[Card]())
}

// GetCardsByCharacterId fetches and returns the owned card list for the character.
func (p *ProcessorImpl) GetCardsByCharacterId(characterId character.Id) ([]Card, error) {
	return p.CardsByCharacterIdProvider(characterId)()
}
