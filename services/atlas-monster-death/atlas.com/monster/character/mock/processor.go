package mock

import (
	"atlas-monster-death/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

type ProcessorMock struct {
	GetByIdFunc         func(characterId uint32) (character.Model, error)
	AwardExperienceFunc func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(characterId)
	}
	return character.Model{}, nil
}

func (m *ProcessorMock) AwardExperience(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
	if m.AwardExperienceFunc != nil {
		return m.AwardExperienceFunc(ch, characterId, white, amount, party)
	}
	return nil
}
