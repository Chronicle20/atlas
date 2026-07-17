package mock

import (
	"atlas-mini-games/data/chalkboard"
)

type ProcessorMock struct {
	HasOpenFunc func(characterId uint32) (bool, error)
}

func (m *ProcessorMock) HasOpen(characterId uint32) (bool, error) {
	if m.HasOpenFunc != nil {
		return m.HasOpenFunc(characterId)
	}
	return false, nil
}

var _ chalkboard.Processor = (*ProcessorMock)(nil)
