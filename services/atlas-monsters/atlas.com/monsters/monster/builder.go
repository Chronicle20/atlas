package monster

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Clone creates a Builder initialized from an existing Model.
// This centralizes field copying for immutable model mutations.
func Clone(m Model) *Builder {
	effects := make([]StatusEffect, len(m.statusEffects))
	copy(effects, m.statusEffects)
	return &Builder{
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
		experienceEntries:  m.experienceEntries,
		statusEffects:      effects,
		nextSkillDecision:  m.nextSkillDecision,
		lastDamageTakenMs:  m.lastDamageTakenMs,
		aggroRefreshedMs:   m.aggroRefreshedMs,
		spawnSourceType:    m.spawnSourceType,
		spawnSourceId:      m.spawnSourceId,
	}
}

// Builder provides a fluent interface for creating Model instances.
type Builder struct {
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
	experienceEntries  []entry
	statusEffects      []StatusEffect
	nextSkillDecision  nextSkillDecision
	lastDamageTakenMs  int64
	aggroRefreshedMs   int64
	spawnSourceType    string
	spawnSourceId      string
}

// SetX sets the X coordinate.
func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	return b
}

// SetY sets the Y coordinate.
func (b *Builder) SetY(y int16) *Builder {
	b.y = y
	return b
}

// SetStance sets the stance/animation state.
func (b *Builder) SetStance(stance byte) *Builder {
	b.stance = stance
	return b
}

// SetFh sets the foothold the monster is anchored to.
func (b *Builder) SetFh(fh int16) *Builder {
	b.fh = fh
	return b
}

// SetHp sets the current hit points.
func (b *Builder) SetHp(hp uint32) *Builder {
	b.hp = hp
	return b
}

// SetControlCharacterId sets the controlling character ID.
func (b *Builder) SetControlCharacterId(id uint32) *Builder {
	b.controlCharacterId = id
	return b
}

// SetControllerHasAggro sets whether the controlling character has aggro.
func (b *Builder) SetControllerHasAggro(v bool) *Builder {
	b.controllerHasAggro = v
	return b
}

// SetMp sets the current mana points.
func (b *Builder) SetMp(mp uint32) *Builder {
	b.mp = mp
	return b
}

// SetNextSkillDecision sets the picker's chosen next skill (or sentinel zero
// for "no skill"). Picker-only API; not used by gameplay code.
func (b *Builder) SetNextSkillDecision(d nextSkillDecision) *Builder {
	b.nextSkillDecision = d
	return b
}

// SetLastDamageTakenMs sets the most-recent damage timestamp. Used by the
// recovery task's HP-regen idle gate.
func (b *Builder) SetLastDamageTakenMs(v int64) *Builder {
	b.lastDamageTakenMs = v
	return b
}

// SetAggroRefreshedMs sets the aggro lease stamp.
func (b *Builder) SetAggroRefreshedMs(v int64) *Builder {
	b.aggroRefreshedMs = v
	return b
}

// SetSpawnSource sets the opaque spawn provenance pair.
func (b *Builder) SetSpawnSource(sourceType string, sourceId string) *Builder {
	b.spawnSourceType = sourceType
	b.spawnSourceId = sourceId
	return b
}

// AddDamageEntry credits damage to a character's damage entry, aggregating by
// characterId. A repeat call for the same character sums into the existing
// entry and leaves its LastHitMs alone (this signature carries no timestamp);
// a first call appends, so slice order records first contact. This mirrors
// Registry.ApplyDamage (registry.go:436-495) so both write paths agree, which
// is what makes Model.DamageSummary()'s "pre-aggregated" contract true.
func (b *Builder) AddDamageEntry(characterId uint32, damage uint32) *Builder {
	b.damageEntries = creditModelEntry(b.damageEntries, characterId, damage)
	b.experienceEntries = creditModelEntry(b.experienceEntries, characterId, damage)
	return b
}

// creditModelEntry accumulates damage onto a character's entry, appending on
// first contact. Shared by the aggro and experience ledgers so the builder
// write path credits both exactly as Registry.ApplyDamage does.
func creditModelEntry(es []entry, characterId uint32, damage uint32) []entry {
	for i := range es {
		if es[i].CharacterId == characterId {
			es[i].Damage += damage
			return es
		}
	}
	return append(es, entry{
		CharacterId: characterId,
		Damage:      damage,
	})
}

// AddStatusEffect adds a status effect, replacing any existing effect with overlapping status types.
// Exception: VENOM stacks up to 3 times. When the cap is reached, the
// VENOM-bearing effect with the earliest ExpiresAt is evicted (not the
// first-inserted), per design D3 / PRD FR-4.4.2.
func (b *Builder) AddStatusEffect(effect StatusEffect) *Builder {
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
func (b *Builder) RemoveStatusEffect(effectId uuid.UUID) *Builder {
	for i, se := range b.statusEffects {
		if se.EffectId() == effectId {
			b.statusEffects = append(b.statusEffects[:i], b.statusEffects[i+1:]...)
			return b
		}
	}
	return b
}

// RemoveStatusEffectByType removes all status effects that contain the given status type.
func (b *Builder) RemoveStatusEffectByType(statusType string) *Builder {
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
func (b *Builder) ClearStatusEffects() *Builder {
	b.statusEffects = make([]StatusEffect, 0)
	return b
}

// Build creates an immutable Model from the builder state.
func (b *Builder) Build() Model {
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
		experienceEntries:  b.experienceEntries,
		statusEffects:      b.statusEffects,
		nextSkillDecision:  b.nextSkillDecision,
		lastDamageTakenMs:  b.lastDamageTakenMs,
		aggroRefreshedMs:   b.aggroRefreshedMs,
		spawnSourceType:    b.spawnSourceType,
		spawnSourceId:      b.spawnSourceId,
	}
}
