package mock

import (
	"atlas-mini-games/data/inventory"
)

type ProcessorMock struct {
	HasItemFunc func(characterId uint32, itemId uint32) (bool, error)
}

func (m *ProcessorMock) HasItem(characterId uint32, itemId uint32) (bool, error) {
	if m.HasItemFunc != nil {
		return m.HasItemFunc(characterId, itemId)
	}
	return false, nil
}

var _ inventory.Processor = (*ProcessorMock)(nil)
