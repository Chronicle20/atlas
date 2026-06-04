package mist

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// Mist represents an area-of-effect mist field placed on a map. It carries a
// disease (status effect) that is applied to characters whose position falls
// within its axis-aligned bounding box on each tick.
type Mist struct {
	id              uuid.UUID
	f               field.Model
	ownerType        string
	ownerId          uint32
	sourceSkillId    uint32
	sourceSkillLevel uint32
	mistType         int32
	originX          int16
	originY          int16
	ltX              int16
	ltY              int16
	rbX              int16
	rbY              int16
	disease          string
	diseaseValue     int32
	diseaseDuration  time.Duration
	duration         time.Duration
	tickInterval     time.Duration
	createdAt        time.Time
	expiresAt        time.Time
	lastTick         time.Time
}

// Id returns the unique identifier for this mist.
func (m Mist) Id() uuid.UUID {
	return m.id
}

// Field returns the field this mist belongs to.
func (m Mist) Field() field.Model {
	return m.f
}

// WorldId returns the world id of the mist's field.
func (m Mist) WorldId() world.Id {
	return m.f.WorldId()
}

// ChannelId returns the channel id of the mist's field.
func (m Mist) ChannelId() channel.Id {
	return m.f.ChannelId()
}

// MapId returns the map id of the mist's field.
func (m Mist) MapId() _map.Id {
	return m.f.MapId()
}

// OwnerType returns the type of entity that owns this mist (e.g. MONSTER, CHARACTER).
func (m Mist) OwnerType() string {
	return m.ownerType
}

// OwnerId returns the id of the entity that owns this mist.
func (m Mist) OwnerId() uint32 {
	return m.ownerId
}

// SourceSkillId returns the id of the skill that produced this mist.
func (m Mist) SourceSkillId() uint32 {
	return m.sourceSkillId
}

// SourceSkillLevel returns the level of the skill that produced this mist.
func (m Mist) SourceSkillLevel() uint32 {
	return m.sourceSkillLevel
}

// Type returns the mist/affected-area type discriminator. Defaults to 0.
func (m Mist) Type() int32 {
	return m.mistType
}

// OriginX returns the x coordinate of the mist's origin (anchor).
func (m Mist) OriginX() int16 {
	return m.originX
}

// OriginY returns the y coordinate of the mist's origin (anchor).
func (m Mist) OriginY() int16 {
	return m.originY
}

// LtX returns the left-top x offset relative to the origin.
func (m Mist) LtX() int16 {
	return m.ltX
}

// LtY returns the left-top y offset relative to the origin.
func (m Mist) LtY() int16 {
	return m.ltY
}

// RbX returns the right-bottom x offset relative to the origin.
func (m Mist) RbX() int16 {
	return m.rbX
}

// RbY returns the right-bottom y offset relative to the origin.
func (m Mist) RbY() int16 {
	return m.rbY
}

// Disease returns the name of the disease applied by this mist.
func (m Mist) Disease() string {
	return m.disease
}

// DiseaseValue returns the magnitude (level/damage) of the applied disease.
func (m Mist) DiseaseValue() int32 {
	return m.diseaseValue
}

// DiseaseDuration returns how long the applied disease lasts on a target.
func (m Mist) DiseaseDuration() time.Duration {
	return m.diseaseDuration
}

// Duration returns the total lifetime of this mist.
func (m Mist) Duration() time.Duration {
	return m.duration
}

// TickInterval returns the interval between disease application ticks.
func (m Mist) TickInterval() time.Duration {
	return m.tickInterval
}

// CreatedAt returns the time the mist was constructed.
func (m Mist) CreatedAt() time.Time {
	return m.createdAt
}

// ExpiresAt returns the absolute time when the mist expires.
func (m Mist) ExpiresAt() time.Time {
	return m.expiresAt
}

// LastTick returns the time of the most recent disease application tick.
func (m Mist) LastTick() time.Time {
	return m.lastTick
}

// Contains reports whether the given world coordinates fall within the mist's
// axis-aligned bounding box (inclusive of edges).
func (m Mist) Contains(x, y int16) bool {
	minX := m.originX + m.ltX
	maxX := m.originX + m.rbX
	minY := m.originY + m.ltY
	maxY := m.originY + m.rbY
	return x >= minX && x <= maxX && y >= minY && y <= maxY
}

// Expired returns true when the current time is past the mist's expiration.
func (m Mist) Expired() bool {
	return time.Now().After(m.expiresAt)
}

// ShouldTick returns true when enough time has elapsed since lastTick for
// another disease application tick to fire.
func (m Mist) ShouldTick() bool {
	if m.tickInterval <= 0 {
		return false
	}
	return time.Since(m.lastTick) >= m.tickInterval
}

// WithLastTick returns a copy of the mist with lastTick advanced to t.
func (m Mist) WithLastTick(t time.Time) Mist {
	m.lastTick = t
	return m
}

// Builder constructs a Mist value via fluent setters.
type Builder struct {
	id               uuid.UUID
	f                field.Model
	ownerType        string
	ownerId          uint32
	sourceSkillId    uint32
	sourceSkillLevel uint32
	mistType         int32
	originX          int16
	originY          int16
	ltX              int16
	ltY              int16
	rbX              int16
	rbY              int16
	disease          string
	diseaseValue     int32
	diseaseDuration  time.Duration
	duration         time.Duration
	tickInterval     time.Duration
	createdAt        time.Time
	expiresAt        time.Time
	lastTick         time.Time
}

// NewBuilder constructs a Builder anchored to the given mist id and field.
// lastTick is initialized far enough in the past that the first ShouldTick
// call (after a SetTickInterval) returns true.
func NewBuilder(id uuid.UUID, f field.Model) *Builder {
	now := time.Now()
	return &Builder{
		id:        id,
		f:         f,
		createdAt: now,
		expiresAt: now,
		// Set lastTick far in the past so the first tick fires immediately.
		lastTick: now.Add(-24 * time.Hour),
	}
}

// SetOwner sets the owner type and id.
func (b *Builder) SetOwner(ownerType string, ownerId uint32) *Builder {
	b.ownerType = ownerType
	b.ownerId = ownerId
	return b
}

// SetSource sets the skill id and level responsible for the mist.
func (b *Builder) SetSource(skillId, skillLevel uint32) *Builder {
	b.sourceSkillId = skillId
	b.sourceSkillLevel = skillLevel
	return b
}

// SetType sets the mist/affected-area type discriminator. Defaults to 0 if unset.
func (b *Builder) SetType(t int32) *Builder {
	b.mistType = t
	return b
}

// SetOrigin sets the world-space anchor coordinates.
func (b *Builder) SetOrigin(x, y int16) *Builder {
	b.originX = x
	b.originY = y
	return b
}

// SetBounds sets the left-top and right-bottom offsets relative to the origin.
func (b *Builder) SetBounds(ltX, ltY, rbX, rbY int16) *Builder {
	b.ltX = ltX
	b.ltY = ltY
	b.rbX = rbX
	b.rbY = rbY
	return b
}

// SetDisease sets the disease name, magnitude, and per-target duration.
func (b *Builder) SetDisease(disease string, value int32, duration time.Duration) *Builder {
	b.disease = disease
	b.diseaseValue = value
	b.diseaseDuration = duration
	return b
}

// SetDuration sets the total mist lifetime and recomputes expiresAt from createdAt.
func (b *Builder) SetDuration(d time.Duration) *Builder {
	b.duration = d
	b.expiresAt = b.createdAt.Add(d)
	return b
}

// SetTickInterval sets the per-tick interval for disease application.
func (b *Builder) SetTickInterval(d time.Duration) *Builder {
	b.tickInterval = d
	return b
}

// Build returns a value-immutable Mist.
func (b *Builder) Build() Mist {
	return Mist{
		id:               b.id,
		f:                b.f,
		ownerType:        b.ownerType,
		ownerId:          b.ownerId,
		sourceSkillId:    b.sourceSkillId,
		sourceSkillLevel: b.sourceSkillLevel,
		mistType:         b.mistType,
		originX:          b.originX,
		originY:          b.originY,
		ltX:              b.ltX,
		ltY:              b.ltY,
		rbX:              b.rbX,
		rbY:              b.rbY,
		disease:          b.disease,
		diseaseValue:     b.diseaseValue,
		diseaseDuration:  b.diseaseDuration,
		duration:         b.duration,
		tickInterval:     b.tickInterval,
		createdAt:        b.createdAt,
		expiresAt:        b.expiresAt,
		lastTick:         b.lastTick,
	}
}
