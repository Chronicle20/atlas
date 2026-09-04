package mock

import (
	"atlas-maker/character"
)

type ProcessorMock struct {
	GetByIdFunc func(characterId uint32) (character.Model, error)
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(characterId)
	}
	return character.Model{}, nil
}
