package mock

import (
	"atlas-channel/mount"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ByCharacterIdProviderFunc func(characterId uint32) model.Provider[mount.Model]
	GetByCharacterIdFunc      func(characterId uint32) (mount.Model, error)
}

var _ mount.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[mount.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return model.FixedProvider(mount.Model{})
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) (mount.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return mount.Model{}, nil
}
