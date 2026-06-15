package summon

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ModelBuilder struct {
	id               uint32
	ownerCharacterId uint32
	skillId          uint32
	skillLevel       byte
	summonType       SummonType
	movementType     MovementType
	fld              field.Model
	x                int16
	y                int16
	stance           byte
	hp               int32
	maxHp            int32
	animated         bool
	spawnTime        time.Time
	expiresAt        time.Time
	nextHealAt       time.Time
	nextBuffAt       time.Time
	healAmount       int16
	healInterval     time.Duration
	buffInterval     time.Duration
	buffSourceId     int32
	buffLevel        byte
	buffDuration     int32
	buffChanges      []StatChange
}

func NewBuilder() *ModelBuilder { return &ModelBuilder{animated: true} }

func Clone(m Model) *ModelBuilder {
	changes := make([]StatChange, len(m.buffChanges))
	copy(changes, m.buffChanges)
	return &ModelBuilder{
		id: m.id, ownerCharacterId: m.ownerCharacterId, skillId: m.skillId,
		skillLevel: m.skillLevel, summonType: m.summonType, movementType: m.movementType,
		fld: m.fld, x: m.x, y: m.y, stance: m.stance, hp: m.hp, maxHp: m.maxHp,
		animated: m.animated, spawnTime: m.spawnTime, expiresAt: m.expiresAt,
		nextHealAt: m.nextHealAt, nextBuffAt: m.nextBuffAt, healAmount: m.healAmount,
		healInterval: m.healInterval, buffInterval: m.buffInterval,
		buffSourceId: m.buffSourceId, buffLevel: m.buffLevel, buffDuration: m.buffDuration,
		buffChanges: changes,
	}
}

func (b *ModelBuilder) SetId(v uint32) *ModelBuilder                  { b.id = v; return b }
func (b *ModelBuilder) SetOwnerCharacterId(v uint32) *ModelBuilder    { b.ownerCharacterId = v; return b }
func (b *ModelBuilder) SetSkillId(v uint32) *ModelBuilder             { b.skillId = v; return b }
func (b *ModelBuilder) SetSkillLevel(v byte) *ModelBuilder            { b.skillLevel = v; return b }
func (b *ModelBuilder) SetSummonType(v SummonType) *ModelBuilder      { b.summonType = v; return b }
func (b *ModelBuilder) SetMovementType(v MovementType) *ModelBuilder  { b.movementType = v; return b }
func (b *ModelBuilder) SetField(v field.Model) *ModelBuilder          { b.fld = v; return b }
func (b *ModelBuilder) SetX(v int16) *ModelBuilder                    { b.x = v; return b }
func (b *ModelBuilder) SetY(v int16) *ModelBuilder                    { b.y = v; return b }
func (b *ModelBuilder) SetStance(v byte) *ModelBuilder                { b.stance = v; return b }
func (b *ModelBuilder) SetHp(v int32) *ModelBuilder                   { b.hp = v; return b }
func (b *ModelBuilder) SetMaxHp(v int32) *ModelBuilder                { b.maxHp = v; return b }
func (b *ModelBuilder) SetAnimated(v bool) *ModelBuilder              { b.animated = v; return b }
func (b *ModelBuilder) SetSpawnTime(v time.Time) *ModelBuilder        { b.spawnTime = v; return b }
func (b *ModelBuilder) SetExpiresAt(v time.Time) *ModelBuilder        { b.expiresAt = v; return b }
func (b *ModelBuilder) SetNextHealAt(v time.Time) *ModelBuilder       { b.nextHealAt = v; return b }
func (b *ModelBuilder) SetNextBuffAt(v time.Time) *ModelBuilder       { b.nextBuffAt = v; return b }
func (b *ModelBuilder) SetHealAmount(v int16) *ModelBuilder           { b.healAmount = v; return b }
func (b *ModelBuilder) SetHealInterval(v time.Duration) *ModelBuilder { b.healInterval = v; return b }
func (b *ModelBuilder) SetBuffInterval(v time.Duration) *ModelBuilder { b.buffInterval = v; return b }
func (b *ModelBuilder) SetBuffSourceId(v int32) *ModelBuilder         { b.buffSourceId = v; return b }
func (b *ModelBuilder) SetBuffLevel(v byte) *ModelBuilder             { b.buffLevel = v; return b }
func (b *ModelBuilder) SetBuffDuration(v int32) *ModelBuilder         { b.buffDuration = v; return b }
func (b *ModelBuilder) SetBuffChanges(v []StatChange) *ModelBuilder   { b.buffChanges = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{
		id: b.id, ownerCharacterId: b.ownerCharacterId, skillId: b.skillId,
		skillLevel: b.skillLevel, summonType: b.summonType, movementType: b.movementType,
		fld: b.fld, x: b.x, y: b.y, stance: b.stance, hp: b.hp, maxHp: b.maxHp,
		animated: b.animated, spawnTime: b.spawnTime, expiresAt: b.expiresAt,
		nextHealAt: b.nextHealAt, nextBuffAt: b.nextBuffAt, healAmount: b.healAmount,
		healInterval: b.healInterval, buffInterval: b.buffInterval,
		buffSourceId: b.buffSourceId, buffLevel: b.buffLevel, buffDuration: b.buffDuration,
		buffChanges: b.buffChanges,
	}
}
