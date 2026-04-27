package monster

import (
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// nextSkillDecision is the picker's decision for the next skill (if any) the
// monster should fire on the controller's next tick. Held in-memory only;
// not persisted to Redis. Zero value is the sentinel "no skill, no scheduled
// re-pick" decision.
type nextSkillDecision struct {
	skillId                byte
	skillLevel             byte
	decidedAtMs            int64
	nextEligibleRepickAtMs int64
}

type DamageSummary struct {
	CharacterId   uint32
	Monster       Model
	VisibleDamage uint32
	ActualDamage  int64
	Killed        bool
	WasFirstHit   bool
}

type Model struct {
	uniqueId             uint32
	worldId              world.Id
	channelId            channel.Id
	mapId                _map.Id
	instance             uuid.UUID
	maxHp                uint32
	hp                   uint32
	maxMp                uint32
	mp                   uint32
	monsterId            uint32
	controlCharacterId   uint32
	controllerHasAggro   bool
	x                    int16
	y                    int16
	fh                   int16
	stance               byte
	team                 int8
	damageEntries        []entry
	statusEffects        []StatusEffect
	nextSkillDecision    nextSkillDecision
	lastDamageTakenMs    int64
}

type entry struct {
	CharacterId uint32
	Damage      uint32
	LastHitMs   int64
}

func NewMonster(f field.Model, uniqueId uint32, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) Model {
	return Model{
		uniqueId:           uniqueId,
		worldId:            f.WorldId(),
		channelId:          f.ChannelId(),
		mapId:              f.MapId(),
		instance:           f.Instance(),
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
		statusEffects:      make([]StatusEffect, 0),
	}
}

func (m Model) UniqueId() uint32 {
	return m.uniqueId
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

func (m Model) Field() field.Model {
	return field.NewBuilder(m.worldId, m.channelId, m.mapId).SetInstance(m.instance).Build()
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

func (m Model) ControllerHasAggro() bool {
	return m.controllerHasAggro
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

// DamageSummary returns the per-character damage entries. Entries are now
// pre-aggregated by characterId at write time (Task 1+4), so this is a
// straight passthrough of m.damageEntries.
func (m Model) DamageSummary() []entry {
	return m.damageEntries
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

func (m Model) StatusEffects() []StatusEffect {
	return m.statusEffects
}

func (m Model) NextSkillDecision() nextSkillDecision {
	return m.nextSkillDecision
}

func (m Model) HasStatusEffect(statusType string) bool {
	for _, se := range m.statusEffects {
		if se.HasStatus(statusType) {
			return true
		}
	}
	return false
}

func (m Model) ApplyStatus(effect StatusEffect) Model {
	return Clone(m).
		AddStatusEffect(effect).
		Build()
}

func (m Model) CancelStatus(effectId uuid.UUID) Model {
	return Clone(m).
		RemoveStatusEffect(effectId).
		Build()
}

func (m Model) CancelStatusByType(statusType string) Model {
	return Clone(m).
		RemoveStatusEffectByType(statusType).
		Build()
}

func (m Model) CancelAllStatuses() Model {
	return Clone(m).
		ClearStatusEffects().
		Build()
}

func (m Model) DeductMp(amount uint32) Model {
	deducted := amount
	if deducted > m.mp {
		deducted = m.mp
	}
	return Clone(m).
		SetMp(m.mp - deducted).
		Build()
}

func (m Model) Heal(amount uint32) Model {
	newHp := m.hp + amount
	if newHp > m.maxHp {
		newHp = m.maxHp
	}
	return Clone(m).
		SetHp(newHp).
		Build()
}

func (m Model) HpPercentage() uint32 {
	if m.maxHp == 0 {
		return 0
	}
	return (m.hp * 100) / m.maxHp
}

func (m Model) LastDamageTakenMs() int64 {
	return m.lastDamageTakenMs
}
