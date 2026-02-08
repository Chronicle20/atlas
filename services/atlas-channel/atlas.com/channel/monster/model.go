package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	field              field.Model
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

func (m Model) Field() field.Model {
	return m.field
}

func (m Model) WorldId() world.Id {
	return m.Field().WorldId()
}

func (m Model) ChannelId() channel.Id {
	return m.Field().ChannelId()
}

func (m Model) MapId() _map.Id {
	return m.Field().MapId()
}

func (m Model) Instance() uuid.UUID {
	return m.Field().Instance()
}

func (m Model) Mp() uint32 {
	return m.mp
}

func (m Model) Hp() uint32 {
	return m.hp
}

func (m Model) MaxHp() uint32 {
	return m.maxHp
}
