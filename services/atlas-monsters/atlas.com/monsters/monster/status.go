package monster

import (
	"time"

	"github.com/google/uuid"
)

const (
	SourceTypeMonsterSkill = "MONSTER_SKILL"
	SourceTypePlayerSkill  = "PLAYER_SKILL"
)

const (
	// StatusPoison is the damage-over-time status whose magnitude is the
	// per-tick damage. Unlike most statuses its value is NOT supplied by the
	// caster: it is derived per-monster at apply time (see
	// ResolvePoisonDamage / ApplyStatusEffect) because it depends on the
	// target's max HP.
	StatusPoison = "POISON"
	// StatusVenom carries its per-tick damage directly in the magnitude the
	// caster sends.
	StatusVenom = "VENOM"
)

// MaxPoisonDamage caps a single poison tick. The client renders the tick from
// the signed 16-bit magnitude carried in the monster temporary-stat packet, so
// a larger value cannot be transmitted faithfully.
const MaxPoisonDamage int32 = 32767

// ResolvePoisonDamage returns the per-tick poison magnitude for a monster with
// maxHp poisoned by a skill at skillLevel: ceil(maxHp / (70 - skillLevel)),
// capped at MaxPoisonDamage.
//
// The magnitude is resolved ONCE, at apply time, and stored in the status
// effect's statuses map. That single value then serves both consumers: the
// damage the tick applies (StatusExpirationTask.calculatePoisonDamage) and the
// number the client renders for itself from the temporary-stat packet. Those
// two must not be computed independently or they will drift.
func ResolvePoisonDamage(maxHp uint32, skillLevel uint32) int32 {
	divisor := int64(70) - int64(skillLevel)
	if divisor <= 0 {
		divisor = 1
	}
	// Integer ceiling division.
	d := (int64(maxHp) + divisor - 1) / divisor
	if d > int64(MaxPoisonDamage) {
		return MaxPoisonDamage
	}
	return int32(d)
}

type StatusEffect struct {
	effectId          uuid.UUID
	sourceType        string
	sourceCharacterId uint32
	sourceSkillId     uint32
	sourceSkillLevel  uint32
	statuses          map[string]int32
	duration          time.Duration
	tickInterval      time.Duration
	lastTick          time.Time
	createdAt         time.Time
	expiresAt         time.Time
	reflectKind       string
	reflectPercent    int32
	reflectLtX        int16
	reflectLtY        int16
	reflectRbX        int16
	reflectRbY        int16
	reflectMaxDamage  int32
}

func NewStatusEffect(sourceType string, sourceCharacterId uint32, sourceSkillId uint32, sourceSkillLevel uint32, statuses map[string]int32, duration time.Duration, tickInterval time.Duration) StatusEffect {
	now := time.Now()
	return StatusEffect{
		effectId:          uuid.New(),
		sourceType:        sourceType,
		sourceCharacterId: sourceCharacterId,
		sourceSkillId:     sourceSkillId,
		sourceSkillLevel:  sourceSkillLevel,
		statuses:          statuses,
		duration:          duration,
		tickInterval:      tickInterval,
		lastTick:          now,
		createdAt:         now,
		expiresAt:         now.Add(duration),
	}
}

func NewReflectStatusEffect(sourceType string, sourceCharacterId uint32, sourceSkillId uint32, sourceSkillLevel uint32, statuses map[string]int32, duration time.Duration, reflectKind string, reflectPercent int32, reflectLtX int16, reflectLtY int16, reflectRbX int16, reflectRbY int16, reflectMaxDamage int32) StatusEffect {
	now := time.Now()
	return StatusEffect{
		effectId:          uuid.New(),
		sourceType:        sourceType,
		sourceCharacterId: sourceCharacterId,
		sourceSkillId:     sourceSkillId,
		sourceSkillLevel:  sourceSkillLevel,
		statuses:          statuses,
		duration:          duration,
		tickInterval:      0,
		lastTick:          now,
		createdAt:         now,
		expiresAt:         now.Add(duration),
		reflectKind:       reflectKind,
		reflectPercent:    reflectPercent,
		reflectLtX:        reflectLtX,
		reflectLtY:        reflectLtY,
		reflectRbX:        reflectRbX,
		reflectRbY:        reflectRbY,
		reflectMaxDamage:  reflectMaxDamage,
	}
}

func (s StatusEffect) EffectId() uuid.UUID {
	return s.effectId
}

func (s StatusEffect) SourceType() string {
	return s.sourceType
}

func (s StatusEffect) SourceCharacterId() uint32 {
	return s.sourceCharacterId
}

func (s StatusEffect) SourceSkillId() uint32 {
	return s.sourceSkillId
}

func (s StatusEffect) SourceSkillLevel() uint32 {
	return s.sourceSkillLevel
}

func (s StatusEffect) Statuses() map[string]int32 {
	return s.statuses
}

func (s StatusEffect) Duration() time.Duration {
	return s.duration
}

func (s StatusEffect) TickInterval() time.Duration {
	return s.tickInterval
}

func (s StatusEffect) LastTick() time.Time {
	return s.lastTick
}

func (s StatusEffect) CreatedAt() time.Time {
	return s.createdAt
}

func (s StatusEffect) ExpiresAt() time.Time {
	return s.expiresAt
}

func (s StatusEffect) Expired() bool {
	return time.Now().After(s.expiresAt)
}

func (s StatusEffect) HasStatus(statusType string) bool {
	_, ok := s.statuses[statusType]
	return ok
}

func (s StatusEffect) ShouldTick() bool {
	if s.tickInterval <= 0 {
		return false
	}
	return time.Since(s.lastTick) >= s.tickInterval
}

func (s StatusEffect) WithLastTick(t time.Time) StatusEffect {
	s.lastTick = t
	return s
}

// WithStatus returns a copy carrying statusType at the given magnitude. The
// statuses map is copied rather than mutated: the receiver is shared with
// whatever built it, and StatusEffect is treated as immutable everywhere else.
func (s StatusEffect) WithStatus(statusType string, amount int32) StatusEffect {
	next := make(map[string]int32, len(s.statuses)+1)
	for k, v := range s.statuses {
		next[k] = v
	}
	next[statusType] = amount
	s.statuses = next
	return s
}

func (s StatusEffect) ReflectKind() string {
	return s.reflectKind
}

func (s StatusEffect) ReflectPercent() int32 {
	return s.reflectPercent
}

func (s StatusEffect) ReflectLtX() int16 {
	return s.reflectLtX
}

func (s StatusEffect) ReflectLtY() int16 {
	return s.reflectLtY
}

func (s StatusEffect) ReflectRbX() int16 {
	return s.reflectRbX
}

func (s StatusEffect) ReflectRbY() int16 {
	return s.reflectRbY
}

func (s StatusEffect) ReflectMaxDamage() int32 {
	return s.reflectMaxDamage
}

func (s StatusEffect) IsReflect() bool {
	return s.reflectKind != ""
}
