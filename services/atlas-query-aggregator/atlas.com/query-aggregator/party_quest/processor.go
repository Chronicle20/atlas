package party_quest

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for party quest processing
type Processor interface {
	GetInstanceByCharacter(characterId uint32) model.Provider[Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new party quest processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetInstanceByCharacter returns the active PQ instance for a character, or a zero-value model if none exists
func (p *ProcessorImpl) GetInstanceByCharacter(characterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		m, err := requests.Provider[RestModel, Model](p.l, p.ctx)(requestInstanceByCharacterId(characterId), Extract)()
		if err != nil {
			p.l.WithError(err).Debugf("Failed to get party quest instance for character %d, treating as no active PQ", characterId)
			return Model{}, nil
		}
		return m, nil
	}
}
