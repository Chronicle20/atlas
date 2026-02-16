package monster

import (
	"time"

	"github.com/google/uuid"
)

const (
	SourceTypeMonsterSkill = "MONSTER_SKILL"
	SourceTypePlayerSkill  = "PLAYER_SKILL"
)

type StatusEffect struct {
	effectId           uuid.UUID
	sourceType         string
	sourceCharacterId  uint32
	sourceSkillId      uint32
	sourceSkillLevel   uint32
	statuses           map[string]int32
	duration           time.Duration
	tickInterval       time.Duration
	lastTick           time.Time
	createdAt          time.Time
	expiresAt          time.Time
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
