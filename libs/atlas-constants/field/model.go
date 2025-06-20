package field

import (
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
}

func (m Model) Id() Id {
	return Id(fmt.Sprintf(IdFormat, m.worldId, m.channelId, m.mapId, m.instance))
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

func (m Model) Instance() uuid.UUID {
	return m.instance
}

func FromId(id Id) (Model, bool) {
	var worldId world.Id
	var channelId channel.Id
	var mapId _map.Id
	var instanceStr string

	// Parse the first three fields and the UUID string
	count, err := fmt.Sscanf(string(id), "%d:%d:%d:%s", &worldId, &channelId, &mapId, &instanceStr)
	if err != nil {
		return Model{}, false
	}
	if count != 4 {
		return Model{}, false
	}

	// Parse the UUID string
	instance, err := uuid.Parse(instanceStr)
	if err != nil {
		return Model{}, false
	}

	return Model{
		worldId:   worldId,
		channelId: channelId,
		mapId:     mapId,
		instance:  instance,
	}, true
}

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
