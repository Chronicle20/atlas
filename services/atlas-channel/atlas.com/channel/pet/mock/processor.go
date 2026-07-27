package mock

import (
	"atlas-channel/pet"
	"atlas-channel/pet/exclude"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByIdProviderFunc    func(petId uint32) model.Provider[pet.Model]
	GetByIdFunc         func(petId uint32) (pet.Model, error)
	ByOwnerProviderFunc func(ownerId uint32) model.Provider[[]pet.Model]
	GetByOwnerFunc      func(ownerId uint32) ([]pet.Model, error)
	SpawnFunc           func(characterId uint32, petId uint32, lead bool) error
	DespawnFunc         func(characterId uint32, petId uint32) error
	AttemptCommandFunc  func(petId uint32, commandId byte, byName bool, characterId uint32) error
	SetExcludeItemsFunc func(characterId uint32, petId uint32, items []exclude.Model) error
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

func (m *ProcessorMock) ByOwnerProvider(ownerId uint32) model.Provider[[]pet.Model] {
	if m.ByOwnerProviderFunc != nil {
		return m.ByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider([]pet.Model{})
}

func (m *ProcessorMock) GetByOwner(ownerId uint32) ([]pet.Model, error) {
	if m.GetByOwnerFunc != nil {
		return m.GetByOwnerFunc(ownerId)
	}
	return nil, nil
}

func (m *ProcessorMock) Spawn(characterId uint32, petId uint32, lead bool) error {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(characterId, petId, lead)
	}
	return nil
}

func (m *ProcessorMock) Despawn(characterId uint32, petId uint32) error {
	if m.DespawnFunc != nil {
		return m.DespawnFunc(characterId, petId)
	}
	return nil
}

func (m *ProcessorMock) AttemptCommand(petId uint32, commandId byte, byName bool, characterId uint32) error {
	if m.AttemptCommandFunc != nil {
		return m.AttemptCommandFunc(petId, commandId, byName, characterId)
	}
	return nil
}

func (m *ProcessorMock) SetExcludeItems(characterId uint32, petId uint32, items []exclude.Model) error {
	if m.SetExcludeItemsFunc != nil {
		return m.SetExcludeItemsFunc(characterId, petId, items)
	}
	return nil
}
