package buff

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for buff data processing
type Processor interface {
	// HasActiveBuff checks if a character has an active buff with the specified source ID
	HasActiveBuff(characterId uint32, sourceId int32) model.Provider[bool]
	// GetBuffsByCharacter returns all active buffs for a character
	GetBuffsByCharacter(characterId uint32) model.Provider[[]Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new buff processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// HasActiveBuff checks if a character has an active buff with the specified source ID
// Returns false if the buff is not found or has expired
func (p *ProcessorImpl) HasActiveBuff(characterId uint32, sourceId int32) model.Provider[bool] {
	return func() (bool, error) {
		buffs, err := p.GetBuffsByCharacter(characterId)()
		if err != nil {
			p.l.WithError(err).Debugf("Failed to get buffs for character %d, assuming no buff", characterId)
			return false, nil
		}

		for _, buff := range buffs {
			if buff.SourceId() == sourceId && buff.IsActive() {
				return true, nil
			}
		}
		return false, nil
	}
}

// GetBuffsByCharacter returns all active buffs for a character
func (p *ProcessorImpl) GetBuffsByCharacter(characterId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		buffsProvider := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacter(characterId), Extract, model.Filters[Model]())
		buffs, err := buffsProvider()
		if err != nil {
			// If not found or error, return empty slice (character may have no buffs)
			p.l.WithError(err).Debugf("Failed to get buffs for character %d", characterId)
			return []Model{}, nil
		}
		return buffs, nil
	}
}
