package monsterbook

import (
	mbmsg "atlas-channel/kafka/message/monsterbook"
	"atlas-channel/kafka/producer"
	"context"

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
	coverCardId      uint32
	expBonusPercent  uint16
}

func (c Collection) BookLevel() uint16        { return c.bookLevel }
func (c Collection) NormalCount() uint16      { return c.normalCount }
func (c Collection) SpecialCount() uint16     { return c.specialCount }
func (c Collection) TotalUniqueCards() uint16 { return c.totalUniqueCards }
func (c Collection) CoverCardId() uint32      { return c.coverCardId }
func (c Collection) ExpBonusPercent() uint16  { return c.expBonusPercent }

// Processor exposes monster book emissions and reads from atlas-channel.
type Processor interface {
	RequestSetCover(characterId uint32, coverCardId uint32) error
	ByCharacterIdProvider(characterId uint32) model.Provider[Collection]
	GetByCharacterId(characterId uint32) (Collection, error)
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
func (p *ProcessorImpl) RequestSetCover(characterId uint32, coverCardId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(mbmsg.EnvCommandTopic)(SetCoverCommandProvider(p.t.Id(), characterId, coverCardId))
}

// ByCharacterIdProvider returns a provider that fetches the character's
// monster book collection from atlas-monster-book.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[Collection] {
	return requests.Provider[CollectionRestModel, Collection](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
}

// GetByCharacterId fetches and returns the monster book collection for the
// given character.
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Collection, error) {
	return p.ByCharacterIdProvider(characterId)()
}
