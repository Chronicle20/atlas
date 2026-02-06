package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor provides operations for querying pet data
type Processor interface {
	GetPets(characterId uint32) model.Provider[[]Model]
	GetPetIdBySlot(characterId uint32, slot int8) model.Provider[uint32]
}

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

// GetPets retrieves all pets for a character
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

// GetPetIdBySlot retrieves the pet ID for a character's pet at the specified slot
// Returns an error if no pet is found at the specified slot
func (p *processor) GetPetIdBySlot(characterId uint32, slot int8) model.Provider[uint32] {
	return func() (uint32, error) {
		pets, err := p.GetPets(characterId)()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get pets for character %d when looking for slot %d", characterId, slot)
			return 0, err
		}

		// Find the pet at the specified slot
		for _, pet := range pets {
			if pet.Slot() == slot {
				return pet.Id(), nil
			}
		}

		// No pet found at the specified slot
		return 0, fmt.Errorf("no pet found at slot %d for character %d", slot, characterId)
	}
}
