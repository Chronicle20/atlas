package mock

import (
	"atlas-inventory/pet"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdProviderFunc func(petId uint32) model.Provider[pet.Model]
	GetByIdFunc      func(petId uint32) (pet.Model, error)
	CreateFunc       func(characterId uint32, templateId uint32) (pet.Model, error)
}

var _ pet.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByIdProvider(petId uint32) model.Provider[pet.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(petId)
	}
	return model.FixedProvider(pet.Model{})
}

func (m *ProcessorMock) GetById(petId uint32) (pet.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(petId)
	}
	return pet.Model{}, nil
}

func (m *ProcessorMock) Create(characterId uint32, templateId uint32) (pet.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(characterId, templateId)
	}
	return pet.Model{}, nil
}
