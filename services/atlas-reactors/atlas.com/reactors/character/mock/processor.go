package mock

import (
	"atlas-reactors/character"
)

type ProcessorMock struct {
	PositionFunc func(characterId uint32) (int16, int16, error)
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Position(characterId uint32) (int16, int16, error) {
	if m.PositionFunc != nil {
		return m.PositionFunc(characterId)
	}
	return 0, 0, nil
}
