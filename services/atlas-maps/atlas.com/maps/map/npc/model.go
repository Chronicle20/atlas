package npc

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is a scripted NPC placed on a field by the spawn_npc saga action
// (task-290 G2), mirroring Cosmic's AbstractPlayerInteraction.spawnNpc. It is
// session/instance-scoped: not persisted, and not automatically re-created
// across a restart.
type Model struct {
	uniqueId  uint32
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
	npcId     uint32
	x         int16
	y         int16
	fh        int16
}

func NewModel(f field.Model, uniqueId uint32, npcId uint32, x int16, y int16, fh int16) Model {
	return Model{
		uniqueId:  uniqueId,
		worldId:   f.WorldId(),
		channelId: f.ChannelId(),
		mapId:     f.MapId(),
		instance:  f.Instance(),
		npcId:     npcId,
		x:         x,
		y:         y,
		fh:        fh,
	}
}

func (m Model) UniqueId() uint32      { return m.uniqueId }
func (m Model) WorldId() world.Id     { return m.worldId }
func (m Model) ChannelId() channel.Id { return m.channelId }
func (m Model) MapId() _map.Id        { return m.mapId }
func (m Model) Instance() uuid.UUID   { return m.instance }
func (m Model) NpcId() uint32         { return m.npcId }
func (m Model) X() int16              { return m.x }
func (m Model) Y() int16              { return m.y }
func (m Model) Fh() int16             { return m.fh }

func (m Model) Field() field.Model {
	return field.NewBuilder(m.worldId, m.channelId, m.mapId).SetInstance(m.instance).Build()
}
