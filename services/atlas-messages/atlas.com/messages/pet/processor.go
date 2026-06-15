package pet

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for pet lookups against the pets service.
type Processor interface {
	GetPets(characterId uint32) ([]Model, error)
	GetSpawnedPetIds(characterId uint32) ([]uint32, error)
}

// ProcessorImpl implements the Processor interface.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new pet processor.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetPets returns all pets owned by a character.
func (p *ProcessorImpl) GetPets(characterId uint32) ([]Model, error) {
	pets, err := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get pets for character [%d].", characterId)
		return nil, err
	}
	return pets, nil
}

// GetSpawnedPetIds returns the ids of the character's spawned pets (slot >= 0).
func (p *ProcessorImpl) GetSpawnedPetIds(characterId uint32) ([]uint32, error) {
	pets, err := p.GetPets(characterId)
	if err != nil {
		return nil, err
	}
	ids := make([]uint32, 0, len(pets))
	for _, pet := range pets {
		if pet.IsSpawned() {
			ids = append(ids, pet.Id())
		}
	}
	return ids, nil
}
