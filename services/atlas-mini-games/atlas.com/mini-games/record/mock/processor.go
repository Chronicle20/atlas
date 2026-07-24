package mock

import (
	"atlas-mini-games/record"
)

type ProcessorMock struct {
	GetByCharacterFunc func(characterId uint32) ([]record.Model, error)
}

func (m *ProcessorMock) GetByCharacter(characterId uint32) ([]record.Model, error) {
	if m.GetByCharacterFunc != nil {
		return m.GetByCharacterFunc(characterId)
	}
	return nil, nil
}

var _ record.Processor = (*ProcessorMock)(nil)
