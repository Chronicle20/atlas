package mock

import (
	"atlas-channel/macro"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByCharacterIdProviderFunc func(characterId uint32) model.Provider[[]macro.Model]
	GetByCharacterIdFunc      func(characterId uint32) ([]macro.Model, error)
	UpdateFunc                func(characterId uint32, macros []macro.Model) error
}

var _ macro.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[[]macro.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return model.FixedProvider([]macro.Model{})
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]macro.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return nil, nil
}

func (m *ProcessorMock) Update(characterId uint32, macros []macro.Model) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(characterId, macros)
	}
	return nil
}
