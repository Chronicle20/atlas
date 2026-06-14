package summon

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type SummonType string

const (
	SummonTypePuppet   SummonType = "PUPPET"
	SummonTypeAttacker SummonType = "ATTACKER"
	SummonTypeBuffAura SummonType = "BUFF_AURA"
)

type MovementType byte

const (
	MovementStationary   MovementType = 0
	MovementFollow       MovementType = 1
	MovementCircleFollow MovementType = 3
)

type Model struct {
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

	// Beholder-only aura snapshot (zero-valued for all other summons)
	nextHealAt   time.Time
	nextBuffAt   time.Time
	healAmount   int16
	healInterval time.Duration
	buffInterval time.Duration
	buffSourceId int32
	buffLevel    byte
	buffDuration int32
	buffChanges  []StatChange
}

// StatChange mirrors the buff command's change element (see context.md §5).
type StatChange struct {
	Type   string
	Amount int32
}

func (m Model) Id() uint32                   { return m.id }
func (m Model) OwnerCharacterId() uint32     { return m.ownerCharacterId }
func (m Model) SkillId() uint32              { return m.skillId }
func (m Model) SkillLevel() byte             { return m.skillLevel }
func (m Model) SummonType() SummonType       { return m.summonType }
func (m Model) MovementType() MovementType   { return m.movementType }
func (m Model) Field() field.Model           { return m.fld }
func (m Model) X() int16                     { return m.x }
func (m Model) Y() int16                     { return m.y }
func (m Model) Stance() byte                 { return m.stance }
func (m Model) Hp() int32                    { return m.hp }
func (m Model) MaxHp() int32                 { return m.maxHp }
func (m Model) Animated() bool               { return m.animated }
func (m Model) SpawnTime() time.Time         { return m.spawnTime }
func (m Model) ExpiresAt() time.Time         { return m.expiresAt }
func (m Model) IsPuppet() bool               { return m.summonType == SummonTypePuppet }
func (m Model) IsBeholder() bool             { return m.summonType == SummonTypeBuffAura }
func (m Model) NextHealAt() time.Time        { return m.nextHealAt }
func (m Model) NextBuffAt() time.Time        { return m.nextBuffAt }
func (m Model) HealAmount() int16            { return m.healAmount }
func (m Model) HealInterval() time.Duration  { return m.healInterval }
func (m Model) BuffInterval() time.Duration  { return m.buffInterval }
func (m Model) BuffSourceId() int32          { return m.buffSourceId }
func (m Model) BuffLevel() byte              { return m.buffLevel }
func (m Model) BuffDuration() int32          { return m.buffDuration }
func (m Model) BuffChanges() []StatChange    { return m.buffChanges }

// Move returns a copy at the new position/stance (non-stationary summons only).
func (m Model) Move(x int16, y int16, stance byte) Model {
	return Clone(m).SetX(x).SetY(y).SetStance(stance).Build()
}

// AddHP returns a copy with hp adjusted by delta, clamped to [0, maxHp].
func (m Model) AddHP(delta int32) Model {
	hp := m.hp + delta
	if hp < 0 {
		hp = 0
	}
	if m.maxHp > 0 && hp > m.maxHp {
		hp = m.maxHp
	}
	return Clone(m).SetHp(hp).Build()
}
