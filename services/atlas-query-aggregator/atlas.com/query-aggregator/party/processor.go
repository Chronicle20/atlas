package party

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for party processing
type Processor interface {
	GetPartyByCharacter(characterId uint32) model.Provider[Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new party processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetPartyByCharacter returns the party for a character, or a zero-value model if the character has no party
func (p *ProcessorImpl) GetPartyByCharacter(characterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		models, err := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(characterId), Extract, model.Filters[Model]())()
		if err != nil {
			p.l.WithError(err).Debugf("Failed to get party for character %d, treating as no party", characterId)
			return Model{}, nil
		}
		if len(models) == 0 {
			return Model{}, nil
		}
		return models[0], nil
	}
}
