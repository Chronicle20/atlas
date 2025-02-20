package _map

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

type Id uint32

const (
	EmptyMapId = Id(999999999)
)

type Model struct {
	worldId   world.Id
	channelId channel.Id
	mapId     Id
}

func NewModel(worldId world.Id) func(channelId channel.Id) func(mapId Id) Model {
	return func(channelId channel.Id) func(mapId Id) Model {
		return func(mapId Id) Model {
			return Model{
				worldId:   worldId,
				channelId: channelId,
				mapId:     mapId,
			}
		}
	}
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) MapId() Id {
	return m.mapId
}
