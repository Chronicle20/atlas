package mock

import (
	"atlas-maps/map/character"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Processor struct {
	GetCharactersInMapFunc    func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
	GetMapsWithCharactersFunc func() []character.MapKey
	EnterFunc                 func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
	ExitFunc                  func(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32)
}

func (m *Processor) GetCharactersInMap(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error) {
	if m.GetCharactersInMapFunc != nil {
		return m.GetCharactersInMapFunc(transactionId, worldId, channelId, mapId)
	}
	return nil, nil
}

func (m *Processor) GetMapsWithCharacters() []character.MapKey {
	if m.GetMapsWithCharactersFunc != nil {
		return m.GetMapsWithCharactersFunc()
	}
	return nil
}

func (m *Processor) Enter(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	if m.EnterFunc != nil {
		m.EnterFunc(transactionId, worldId, channelId, mapId, characterId)
	}
}

func (m *Processor) Exit(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) {
	if m.ExitFunc != nil {
		m.ExitFunc(transactionId, worldId, channelId, mapId, characterId)
	}
}
