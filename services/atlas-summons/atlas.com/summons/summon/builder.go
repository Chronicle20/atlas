package summon

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Builder struct {
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

func NewBuilder() *Builder { return &Builder{animated: true} }

func Clone(m Model) *Builder {
	changes := make([]StatChange, len(m.buffChanges))
	copy(changes, m.buffChanges)
	return &Builder{
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

func (b *Builder) SetId(v uint32) *Builder                  { b.id = v; return b }
func (b *Builder) SetOwnerCharacterId(v uint32) *Builder    { b.ownerCharacterId = v; return b }
func (b *Builder) SetSkillId(v uint32) *Builder             { b.skillId = v; return b }
func (b *Builder) SetSkillLevel(v byte) *Builder            { b.skillLevel = v; return b }
func (b *Builder) SetSummonType(v SummonType) *Builder      { b.summonType = v; return b }
func (b *Builder) SetMovementType(v MovementType) *Builder  { b.movementType = v; return b }
func (b *Builder) SetField(v field.Model) *Builder          { b.fld = v; return b }
func (b *Builder) SetX(v int16) *Builder                    { b.x = v; return b }
func (b *Builder) SetY(v int16) *Builder                    { b.y = v; return b }
func (b *Builder) SetStance(v byte) *Builder                { b.stance = v; return b }
func (b *Builder) SetHp(v int32) *Builder                   { b.hp = v; return b }
func (b *Builder) SetMaxHp(v int32) *Builder                { b.maxHp = v; return b }
func (b *Builder) SetAnimated(v bool) *Builder              { b.animated = v; return b }
func (b *Builder) SetSpawnTime(v time.Time) *Builder        { b.spawnTime = v; return b }
func (b *Builder) SetExpiresAt(v time.Time) *Builder        { b.expiresAt = v; return b }
func (b *Builder) SetNextHealAt(v time.Time) *Builder       { b.nextHealAt = v; return b }
func (b *Builder) SetNextBuffAt(v time.Time) *Builder       { b.nextBuffAt = v; return b }
func (b *Builder) SetHealAmount(v int16) *Builder           { b.healAmount = v; return b }
func (b *Builder) SetHealInterval(v time.Duration) *Builder { b.healInterval = v; return b }

func (b *Builder) SetBuffInterval(v time.Duration) *Builder { b.buffInterval = v; return b }

func (b *Builder) SetBuffSourceId(v int32) *Builder { b.buffSourceId = v; return b }
func (b *Builder) SetBuffLevel(v byte) *Builder     { b.buffLevel = v; return b }
func (b *Builder) SetBuffDuration(v int32) *Builder { b.buffDuration = v; return b }

func (b *Builder) SetBuffChanges(v []StatChange) *Builder { b.buffChanges = v; return b }

func (b *Builder) Build() Model {
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
