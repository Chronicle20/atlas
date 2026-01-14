package mock

import (
	"atlas-guilds/character"
)

type ProcessorMock struct {
	GetByIdFunc func(characterId uint32) (character.Model, error)
}

func (m *ProcessorMock) GetById(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(characterId)
	}
	return character.Model{}, nil
}
