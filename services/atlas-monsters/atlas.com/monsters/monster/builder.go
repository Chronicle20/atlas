package monster

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Clone creates a ModelBuilder initialized from an existing Model.
// This centralizes field copying for immutable model mutations.
func Clone(m Model) *ModelBuilder {
	effects := make([]StatusEffect, len(m.statusEffects))
	copy(effects, m.statusEffects)
	return &ModelBuilder{
		uniqueId:           m.uniqueId,
		worldId:            m.worldId,
		channelId:          m.channelId,
		mapId:              m.mapId,
		instance:           m.instance,
		maxHp:              m.maxHp,
		hp:                 m.hp,
		maxMp:              m.maxMp,
		mp:                 m.mp,
		monsterId:          m.monsterId,
		controlCharacterId: m.controlCharacterId,
		controllerHasAggro: m.controllerHasAggro,
		x:                  m.x,
		y:                  m.y,
		fh:                 m.fh,
		stance:             m.stance,
		team:               m.team,
		damageEntries:      m.damageEntries,
		statusEffects:      effects,
		nextSkillDecision:  m.nextSkillDecision,
		lastDamageTakenMs:  m.lastDamageTakenMs,
	}
}

// ModelBuilder provides a fluent interface for creating Model instances.
type ModelBuilder struct {
	uniqueId           uint32
	worldId            world.Id
	channelId          channel.Id
	mapId              _map.Id
	instance           uuid.UUID
	maxHp              uint32
	hp                 uint32
	maxMp              uint32
	mp                 uint32
	monsterId          uint32
	controlCharacterId uint32
	controllerHasAggro bool
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
	damageEntries      []entry
	statusEffects      []StatusEffect
	nextSkillDecision  nextSkillDecision
	lastDamageTakenMs  int64
}

// SetX sets the X coordinate.
func (b *ModelBuilder) SetX(x int16) *ModelBuilder {
	b.x = x
	return b
}

// SetY sets the Y coordinate.
func (b *ModelBuilder) SetY(y int16) *ModelBuilder {
	b.y = y
	return b
}

// SetStance sets the stance/animation state.
func (b *ModelBuilder) SetStance(stance byte) *ModelBuilder {
	b.stance = stance
	return b
}

// SetHp sets the current hit points.
func (b *ModelBuilder) SetHp(hp uint32) *ModelBuilder {
	b.hp = hp
	return b
}

// SetControlCharacterId sets the controlling character ID.
func (b *ModelBuilder) SetControlCharacterId(id uint32) *ModelBuilder {
	b.controlCharacterId = id
	return b
}

// SetControllerHasAggro sets whether the controlling character has aggro.
func (b *ModelBuilder) SetControllerHasAggro(v bool) *ModelBuilder {
	b.controllerHasAggro = v
	return b
}

// SetMp sets the current mana points.
func (b *ModelBuilder) SetMp(mp uint32) *ModelBuilder {
	b.mp = mp
	return b
}

// SetNextSkillDecision sets the picker's chosen next skill (or sentinel zero
// for "no skill"). Picker-only API; not used by gameplay code.
func (b *ModelBuilder) SetNextSkillDecision(d nextSkillDecision) *ModelBuilder {
	b.nextSkillDecision = d
	return b
}

// SetLastDamageTakenMs sets the most-recent damage timestamp. Used by the
// recovery task's HP-regen idle gate.
func (b *ModelBuilder) SetLastDamageTakenMs(v int64) *ModelBuilder {
	b.lastDamageTakenMs = v
	return b
}

// AddDamageEntry appends a damage entry to the damage tracking list.
func (b *ModelBuilder) AddDamageEntry(characterId uint32, damage uint32) *ModelBuilder {
	b.damageEntries = append(b.damageEntries, entry{
		CharacterId: characterId,
		Damage:      damage,
	})
	return b
}

// AddStatusEffect adds a status effect, replacing any existing effect with overlapping status types.
// Exception: VENOM stacks up to 3 times. When the cap is reached, the
// VENOM-bearing effect with the earliest ExpiresAt is evicted (not the
// first-inserted), per design D3 / PRD FR-4.4.2.
func (b *ModelBuilder) AddStatusEffect(effect StatusEffect) *ModelBuilder {
	for statusType := range effect.Statuses() {
		if statusType == "VENOM" {
			venomCount := 0
			evictIdx := -1
			for i, se := range b.statusEffects {
				if !se.HasStatus("VENOM") {
					continue
				}
				venomCount++
				if evictIdx < 0 || se.ExpiresAt().Before(b.statusEffects[evictIdx].ExpiresAt()) {
					evictIdx = i
				}
			}
			if venomCount >= 3 && evictIdx >= 0 {
				b.statusEffects = append(b.statusEffects[:evictIdx], b.statusEffects[evictIdx+1:]...)
			}
		} else {
			b.RemoveStatusEffectByType(statusType)
		}
	}
	b.statusEffects = append(b.statusEffects, effect)
	return b
}

// RemoveStatusEffect removes a status effect by its ID.
func (b *ModelBuilder) RemoveStatusEffect(effectId uuid.UUID) *ModelBuilder {
	for i, se := range b.statusEffects {
		if se.EffectId() == effectId {
			b.statusEffects = append(b.statusEffects[:i], b.statusEffects[i+1:]...)
			return b
		}
	}
	return b
}

// RemoveStatusEffectByType removes all status effects that contain the given status type.
func (b *ModelBuilder) RemoveStatusEffectByType(statusType string) *ModelBuilder {
	filtered := make([]StatusEffect, 0, len(b.statusEffects))
	for _, se := range b.statusEffects {
		if !se.HasStatus(statusType) {
			filtered = append(filtered, se)
		}
	}
	b.statusEffects = filtered
	return b
}

// ClearStatusEffects removes all status effects.
func (b *ModelBuilder) ClearStatusEffects() *ModelBuilder {
	b.statusEffects = make([]StatusEffect, 0)
	return b
}

// Build creates an immutable Model from the builder state.
func (b *ModelBuilder) Build() Model {
	return Model{
		uniqueId:           b.uniqueId,
		worldId:            b.worldId,
		channelId:          b.channelId,
		mapId:              b.mapId,
		instance:           b.instance,
		maxHp:              b.maxHp,
		hp:                 b.hp,
		maxMp:              b.maxMp,
		mp:                 b.mp,
		monsterId:          b.monsterId,
		controlCharacterId: b.controlCharacterId,
		controllerHasAggro: b.controllerHasAggro,
		x:                  b.x,
		y:                  b.y,
		fh:                 b.fh,
		stance:             b.stance,
		team:               b.team,
		damageEntries:      b.damageEntries,
		statusEffects:      b.statusEffects,
		nextSkillDecision:  b.nextSkillDecision,
		lastDamageTakenMs:  b.lastDamageTakenMs,
	}
}
