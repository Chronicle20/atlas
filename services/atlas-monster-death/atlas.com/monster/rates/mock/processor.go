package mock

import (
	"atlas-monster-death/rates"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

type ProcessorMock struct {
	GetForCharacterFunc func(ch channel.Model, characterId uint32) rates.Model
}

var _ rates.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetForCharacter(ch channel.Model, characterId uint32) rates.Model {
	if m.GetForCharacterFunc != nil {
		return m.GetForCharacterFunc(ch, characterId)
	}
	return rates.Default()
}
