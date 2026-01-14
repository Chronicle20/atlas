package mock

import (
	"atlas-query-aggregator/pet"

	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the pet.Processor interface
type ProcessorImpl struct {
	GetPetsFunc            func(characterId uint32) model.Provider[[]pet.Model]
	GetSpawnedPetCountFunc func(characterId uint32) model.Provider[int]
}

// GetPets returns all pets for a character
func (m *ProcessorImpl) GetPets(characterId uint32) model.Provider[[]pet.Model] {
	if m.GetPetsFunc != nil {
		return m.GetPetsFunc(characterId)
	}
	return func() ([]pet.Model, error) {
		return []pet.Model{}, nil
	}
}

// GetSpawnedPetCount returns the count of spawned pets for a character
func (m *ProcessorImpl) GetSpawnedPetCount(characterId uint32) model.Provider[int] {
	if m.GetSpawnedPetCountFunc != nil {
		return m.GetSpawnedPetCountFunc(characterId)
	}
	return func() (int, error) {
		return 0, nil
	}
}
