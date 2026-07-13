package mock

import (
	"atlas-chairs/validation"
)

type ProcessorMock struct {
	HasItemFunc func(characterId uint32, itemId uint32) (bool, error)
}

var _ validation.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) HasItem(characterId uint32, itemId uint32) (bool, error) {
	if m.HasItemFunc != nil {
		return m.HasItemFunc(characterId, itemId)
	}
	return false, nil
}
