package mock

import (
	"atlas-summons/effectivestats"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ProcessorMock struct {
	GetByCharacterFunc func(worldId world.Id, channelId channel.Id, characterId uint32) (effectivestats.Model, error)
}

var _ effectivestats.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetByCharacter(worldId world.Id, channelId channel.Id, characterId uint32) (effectivestats.Model, error) {
	if m.GetByCharacterFunc != nil {
		return m.GetByCharacterFunc(worldId, channelId, characterId)
	}
	return effectivestats.Model{}, nil
}
