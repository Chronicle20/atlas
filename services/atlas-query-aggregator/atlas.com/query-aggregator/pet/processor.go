package pet

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor defines the interface for pet processing
type Processor interface {
	GetPets(characterId uint32) model.Provider[[]Model]
	GetSpawnedPetCount(characterId uint32) model.Provider[int]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new pet processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetPets returns all pets for a character. The upstream atlas-pets list is
// now paginated (task-117); this drains every page rather than fetching
// just the first, since GetSpawnedPetCount below needs the complete set to
// count every spawned pet.
func (p *ProcessorImpl) GetPets(characterId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		petsProvider := requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byCharacterIdUrl(characterId), 250, Extract, []model.Filter[Model]{})
		pets, err := petsProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get pets for character %d", characterId)
			return []Model{}, err
		}
		return pets, nil
	}
}

// GetSpawnedPetCount returns the count of spawned pets (slot >= 0) for a character
func (p *ProcessorImpl) GetSpawnedPetCount(characterId uint32) model.Provider[int] {
	return func() (int, error) {
		pets, err := p.GetPets(characterId)()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get spawned pet count for character %d", characterId)
			return 0, err
		}

		count := 0
		for _, pet := range pets {
			if pet.IsSpawned() {
				count++
			}
		}

		return count, nil
	}
}
