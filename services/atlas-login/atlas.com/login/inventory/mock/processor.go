package mock

import (
	"atlas-login/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByCharacterIdProviderFunc func(characterId uint32) model.Provider[inventory.Model]
	GetByCharacterIdFunc      func(characterId uint32) (inventory.Model, error)
}

var _ inventory.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[inventory.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return model.FixedProvider(inventory.Model{})
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) (inventory.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return inventory.Model{}, nil
}
