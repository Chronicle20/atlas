package monster

import (
	"math"
)

type DamageSummary struct {
	CharacterId   uint32
	Monster       Model
	VisibleDamage uint32
	ActualDamage  int64
	Killed        bool
}

type Model struct {
	uniqueId           uint32
	worldId            byte
	channelId          byte
	mapId              uint32
	maxHp              uint32
	hp                 uint32
	maxMp              uint32
	mp                 uint32
	monsterId          uint32
	controlCharacterId uint32
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
	damageEntries      []entry
}

type entry struct {
	CharacterId uint32
	Damage      uint32
}

func NewMonster(worldId byte, channelId byte, mapId uint32, uniqueId uint32, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) Model {
	return Model{
		uniqueId:           uniqueId,
		worldId:            worldId,
		channelId:          channelId,
		mapId:              mapId,
		maxHp:              hp,
		hp:                 hp,
		maxMp:              mp,
		mp:                 mp,
		monsterId:          monsterId,
		controlCharacterId: 0,
		x:                  x,
		y:                  y,
		fh:                 fh,
		stance:             stance,
		team:               team,
		damageEntries:      make([]entry, 0),
	}
}

func (m Model) UniqueId() uint32 {
	return m.uniqueId
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}

func (m Model) MapId() uint32 {
	return m.mapId
}

func (m Model) Hp() uint32 {
	return m.hp
}

func (m Model) MonsterId() uint32 {
	return m.monsterId
}

func (m Model) ControlCharacterId() uint32 {
	return m.controlCharacterId
}

func (m Model) Fh() int16 {
	return m.fh
}

func (m Model) Team() int8 {
	return m.team
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

func (m Model) DamageEntries() []entry {
	return m.damageEntries
}

func (m Model) DamageSummary() []entry {
	var damageSummary = make(map[uint32]uint32)
	for _, x := range m.damageEntries {
		if _, ok := damageSummary[x.CharacterId]; ok {
			damageSummary[x.CharacterId] += x.Damage
		} else {
			damageSummary[x.CharacterId] = x.Damage
		}
	}

	var results []entry
	for id, dmg := range damageSummary {
		results = append(results, entry{
			CharacterId: id,
			Damage:      dmg,
		})
	}
	return results
}

func (m Model) Move(x int16, y int16, stance byte) Model {
	return Clone(m).
		SetX(x).
		SetY(y).
		SetStance(stance).
		Build()
}

func (m Model) Control(characterId uint32) Model {
	return Clone(m).
		SetControlCharacterId(characterId).
		Build()
}

func (m Model) ClearControl() Model {
	return Clone(m).
		SetControlCharacterId(0).
		Build()
}

func (m Model) Damage(characterId uint32, damage uint32) Model {
	actualDamage := m.Hp() - uint32(math.Max(float64(m.Hp())-float64(damage), 0))

	return Clone(m).
		SetHp(m.Hp() - actualDamage).
		AddDamageEntry(characterId, actualDamage).
		Build()
}

func (m Model) Alive() bool {
	return m.Hp() > 0
}

func (m Model) DamageLeader() uint32 {
	index := -1
	for i, x := range m.damageEntries {
		if index == -1 {
			index = i
		} else if m.damageEntries[index].Damage < x.Damage {
			index = i
		}
	}
	return m.damageEntries[index].CharacterId
}

func (m Model) MaxHp() uint32 {
	return m.maxHp
}

func (m Model) MaxMp() uint32 {
	return m.maxMp
}

func (m Model) Mp() uint32 {
	return m.mp
}
