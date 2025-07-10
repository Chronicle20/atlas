package marriage

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for marriage gift processing
type Processor interface {
	GetMarriageGifts(characterId uint32) model.Provider[Model]
	HasUnclaimedGifts(characterId uint32) model.Provider[bool]
	GetUnclaimedGiftCount(characterId uint32) model.Provider[int]
}

// processor implements the Processor interface
type processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new marriage processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &processor{
		l:   l,
		ctx: ctx,
	}
}

// GetMarriageGifts returns the marriage gift data for a character
func (p *processor) GetMarriageGifts(characterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		marriageProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
		marriage, err := marriageProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get marriage gifts for character %d", characterId)
			return NewModel(characterId, false), err
		}
		return marriage, nil
	}
}

// HasUnclaimedGifts returns whether the character has unclaimed marriage gifts
func (p *processor) HasUnclaimedGifts(characterId uint32) model.Provider[bool] {
	return func() (bool, error) {
		marriageProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
		marriage, err := marriageProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to check unclaimed gifts for character %d", characterId)
			return false, err
		}
		return marriage.HasUnclaimedGifts(), nil
	}
}

// GetUnclaimedGiftCount returns the number of unclaimed gifts for a character
func (p *processor) GetUnclaimedGiftCount(characterId uint32) model.Provider[int] {
	return func() (int, error) {
		marriageProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
		marriage, err := marriageProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get unclaimed gift count for character %d", characterId)
			return 0, err
		}
		return marriage.UnclaimedGiftCount(), nil
	}
}