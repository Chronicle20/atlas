package mock

import (
	"atlas-maker/skill"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByCharacterIdProviderFunc func(characterId uint32) model.Provider[[]skill.Model]
	GetByCharacterIdFunc      func(characterId uint32) ([]skill.Model, error)
}

var _ skill.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[[]skill.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return model.FixedProvider[[]skill.Model](nil)
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]skill.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return nil, nil
}
