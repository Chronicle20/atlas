package mock

import (
	"atlas-monsters/character/position"
)

type ProcessorMock struct {
	GetPositionFunc func(characterId uint32) (int16, int16, error)
}

var _ position.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetPosition(characterId uint32) (int16, int16, error) {
	if m.GetPositionFunc != nil {
		return m.GetPositionFunc(characterId)
	}
	return 0, 0, nil
}
