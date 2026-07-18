package mock

import (
	"atlas-mini-games/data/character"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetByIdFunc      func(characterId uint32) (character.Model, error)
	ByIdProviderFunc func(characterId uint32) model.Provider[character.Model]
	HpFunc           func(characterId uint32) (uint16, error)
	NameFunc         func(characterId uint32) (string, error)
}

func (m *ProcessorMock) GetById(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(characterId)
	}
	return character.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(characterId uint32) model.Provider[character.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(characterId)
	}
	return model.FixedProvider(character.Model{})
}

func (m *ProcessorMock) Hp(characterId uint32) (uint16, error) {
	if m.HpFunc != nil {
		return m.HpFunc(characterId)
	}
	return 0, nil
}

func (m *ProcessorMock) Name(characterId uint32) (string, error) {
	if m.NameFunc != nil {
		return m.NameFunc(characterId)
	}
	return "", nil
}

var _ character.Processor = (*ProcessorMock)(nil)
