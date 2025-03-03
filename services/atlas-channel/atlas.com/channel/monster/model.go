package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

type Model struct {
	worldId            world.Id
	channelId          channel.Id
	mapId              _map.Id
	uniqueId           uint32
	maxHp              uint32
	hp                 uint32
	mp                 uint32
	monsterId          uint32
	controlCharacterId uint32
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
}

func (m Model) UniqueId() uint32 {
	return m.uniqueId
}

func (m Model) Controlled() bool {
	return m.controlCharacterId != 0
}

func (m Model) MonsterId() uint32 {
	return m.monsterId
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Stance() byte {
	return m.stance
}

func (m Model) Fh() int16 {
	return m.fh
}

func (m Model) Team() int8 {
	return m.team
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

func (m Model) MP() uint32 {
	return m.mp
}

func (m Model) HP() uint32 {
	return m.hp
}

func (m Model) MaxHP() uint32 {
	return m.maxHp
}

type ModelBuilder struct {
	worldId            world.Id
	channelId          channel.Id
	mapId              _map.Id
	uniqueId           uint32
	maxHp              uint32
	hp                 uint32
	mp                 uint32
	monsterId          uint32
	controlCharacterId uint32
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
}

func NewModelBuilder(uniqueId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, monsterId uint32) *ModelBuilder {
	return &ModelBuilder{
		worldId:   worldId,
		channelId: channelId,
		mapId:     mapId,
		uniqueId:  uniqueId,
		monsterId: monsterId,
	}
}

func (b *ModelBuilder) SetMaxHP(maxHp uint32) *ModelBuilder {
	b.maxHp = maxHp
	return b
}

func (b *ModelBuilder) Build() Model {
	return Model{
		worldId:            b.worldId,
		channelId:          b.channelId,
		mapId:              b.mapId,
		uniqueId:           b.uniqueId,
		maxHp:              b.maxHp,
		hp:                 b.hp,
		mp:                 b.mp,
		monsterId:          b.monsterId,
		controlCharacterId: b.controlCharacterId,
		x:                  b.x,
		y:                  b.y,
		fh:                 b.fh,
		stance:             b.stance,
		team:               b.team,
	}
}

func (b *ModelBuilder) SetControlCharacterId(controlCharacterId uint32) *ModelBuilder {
	b.controlCharacterId = controlCharacterId
	return b
}

func (b *ModelBuilder) SetX(x int16) *ModelBuilder {
	b.x = x
	return b
}

func (b *ModelBuilder) SetY(y int16) *ModelBuilder {
	b.y = y
	return b
}

func (b *ModelBuilder) SetStance(stance byte) *ModelBuilder {
	b.stance = stance
	return b
}

func (b *ModelBuilder) SetFH(fh int16) *ModelBuilder {
	b.fh = fh
	return b
}

func (b *ModelBuilder) SetTeam(team int8) *ModelBuilder {
	b.team = team
	return b
}
