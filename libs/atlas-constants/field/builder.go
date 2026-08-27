package field

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Builder struct {
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
}

func NewBuilder(worldId world.Id, channelId channel.Id, mapId _map.Id) *Builder {
	return &Builder{
		worldId:   worldId,
		channelId: channelId,
		mapId:     mapId,
		instance:  uuid.Nil,
	}
}

func (m *Builder) SetWorldId(worldId world.Id) *Builder {
	m.worldId = worldId
	return m
}

func (m *Builder) SetChannelId(channelId channel.Id) *Builder {
	m.channelId = channelId
	return m
}

func (m *Builder) SetMapId(mapId _map.Id) *Builder {
	m.mapId = mapId
	return m
}

func (m *Builder) SetInstance(instance uuid.UUID) *Builder {
	m.instance = instance
	return m
}

func (m *Builder) Build() Model {
	return Model{
		worldId:   m.worldId,
		channelId: m.channelId,
		mapId:     m.mapId,
		instance:  m.instance,
	}
}
