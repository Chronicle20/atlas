package mock

import (
	"atlas-maker/quest"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByCharacterIdProviderFunc func(characterId uint32) model.Provider[[]quest.Model]
	GetByCharacterIdFunc      func(characterId uint32) ([]quest.Model, error)
}

var _ quest.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[[]quest.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return model.FixedProvider[[]quest.Model](nil)
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]quest.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return nil, nil
}
