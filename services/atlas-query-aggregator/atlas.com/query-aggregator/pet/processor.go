package pet

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for pet processing
type Processor interface {
	GetPets(characterId uint32) model.Provider[[]Model]
	GetSpawnedPetCount(characterId uint32) model.Provider[int]
}

// processor implements the Processor interface
type processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new pet processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &processor{
		l:   l,
		ctx: ctx,
	}
}

// GetPets returns all pets for a character
func (p *processor) GetPets(characterId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		petsProvider := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, []model.Filter[Model]{})
		pets, err := petsProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get pets for character %d", characterId)
			return []Model{}, err
		}
		return pets, nil
	}
}

// GetSpawnedPetCount returns the count of spawned pets (slot >= 0) for a character
func (p *processor) GetSpawnedPetCount(characterId uint32) model.Provider[int] {
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
