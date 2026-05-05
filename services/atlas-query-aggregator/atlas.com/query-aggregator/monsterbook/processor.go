package monsterbook

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Collection is the immutable domain representation of a character's monster
// book collection summary, mirroring the shape exposed by atlas-monster-book.
type Collection struct {
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	totalUniqueCards uint16
	coverCardId      uint32
	expBonusPercent  uint16
}

// BookLevel returns the character's monster book level.
func (c Collection) BookLevel() uint16 { return c.bookLevel }

// NormalCount returns the unique normal cards collected.
func (c Collection) NormalCount() uint16 { return c.normalCount }

// SpecialCount returns the unique special cards collected.
func (c Collection) SpecialCount() uint16 { return c.specialCount }

// TotalUniqueCards returns the total unique cards collected, the value used
// by the monsterBookCount validation condition.
func (c Collection) TotalUniqueCards() uint16 { return c.totalUniqueCards }

// CoverCardId returns the active cover card id.
func (c Collection) CoverCardId() uint32 { return c.coverCardId }

// ExpBonusPercent returns the experience bonus tier earned from the book
// level, expressed as a percentage.
func (c Collection) ExpBonusPercent() uint16 { return c.expBonusPercent }

// Processor exposes monster book reads for the query-aggregator's validation
// evaluator. Implementations call atlas-monster-book over REST.
type Processor interface {
	// ByCharacterIdProvider returns a provider that lazily fetches the
	// monster book collection for a character from atlas-monster-book.
	ByCharacterIdProvider(characterId character.Id) model.Provider[Collection]
	// GetByCharacterId fetches and returns the monster book collection for
	// the given character.
	GetByCharacterId(characterId character.Id) (Collection, error)
	// GetTotalUniqueCards returns just the totalUniqueCards count for a
	// character, the value used by the monsterBookCount condition.
	GetTotalUniqueCards(characterId character.Id) (uint16, error)
}

// ProcessorImpl is the REST-backed Processor implementation.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor builds a Processor bound to the request logger and context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
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

// GetTotalUniqueCards returns just the totalUniqueCards count for a character.
// Errors are surfaced so callers can distinguish "no cards" (0) from "lookup
// failed" (error).
func (p *ProcessorImpl) GetTotalUniqueCards(characterId character.Id) (uint16, error) {
	c, err := p.GetByCharacterId(characterId)
	if err != nil {
		return 0, err
	}
	return c.TotalUniqueCards(), nil
}
