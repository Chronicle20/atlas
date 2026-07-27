package mock

import (
	"atlas-channel/party_quest"
)

type ProcessorMock struct {
	GetTimerByCharacterIdFunc func(characterId uint32) (party_quest.TimerModel, error)
}

var _ party_quest.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetTimerByCharacterId(characterId uint32) (party_quest.TimerModel, error) {
	if m.GetTimerByCharacterIdFunc != nil {
		return m.GetTimerByCharacterIdFunc(characterId)
	}
	return party_quest.TimerModel{}, nil
}
