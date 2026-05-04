package location

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	mapId       _map.Id
	instance    uuid.UUID
}

func (m Model) CharacterId() uint32   { return m.characterId }
func (m Model) WorldId() world.Id     { return m.worldId }
func (m Model) ChannelId() channel.Id { return m.channelId }
func (m Model) MapId() _map.Id        { return m.mapId }
func (m Model) Instance() uuid.UUID   { return m.instance }

func (m Model) Field() field.Model {
	return field.NewBuilder(m.worldId, m.channelId, m.mapId).SetInstance(m.instance).Build()
}

type Builder struct{ m Model }

func NewBuilder(characterId uint32) *Builder {
	return &Builder{m: Model{characterId: characterId}}
}

func (b *Builder) SetWorldId(v world.Id) *Builder     { b.m.worldId = v; return b }
func (b *Builder) SetChannelId(v channel.Id) *Builder { b.m.channelId = v; return b }
func (b *Builder) SetMapId(v _map.Id) *Builder        { b.m.mapId = v; return b }
func (b *Builder) SetInstance(v uuid.UUID) *Builder   { b.m.instance = v; return b }
func (b *Builder) SetField(f field.Model) *Builder {
	b.m.worldId = f.WorldId()
	b.m.channelId = f.ChannelId()
	b.m.mapId = f.MapId()
	b.m.instance = f.Instance()
	return b
}
func (b *Builder) Build() Model { return b.m }
