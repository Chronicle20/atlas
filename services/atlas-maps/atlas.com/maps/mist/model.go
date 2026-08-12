package mist

import (
	mistKafka "atlas-maps/kafka/message/mist"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Mist represents an area-of-effect mist field placed on a map. It carries a
// status effect that is applied on each tick to whatever its targetKind names
// -- characters (the monster AREA_POISON path) or monsters (player-cast
// mists). The disease* fields are the generic status name / magnitude /
// per-target duration triple; the names are historical.
type Mist struct {
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
	targetKind       string
	effectKind       string
	recoveryMp       int32
	partyMemberIds   []uint32
	elemAttr         int32
	skillDelay       int16
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

// Type returns the mist/affected-area type discriminator (the client's
// AFFECTEDAREA::nType, +0x4). Defaults to 0 -- see AffectedAreaTypeFor for why
// that default is dangerous for a player-cast mist.
func (m Mist) Type() int32 {
	return m.mistType
}

// AFFECTEDAREA::nType wire values. The client has no enum symbol for this
// field even in the PDB-backed GMS v95 IDB (AFFECTEDAREA::nType is a bare
// `int`), so these names are ours; the VALUES are read from the client.
//
// Only three values are load-bearing. Everything the client does with nType:
//
//   - == 0  CAffectedAreaPool::GetAffectedAreaByPoint (v83 sub_431783
//     @0x4317b6, v95 @0x434cc0 PDB-named) selects an area for the LOCAL USER
//     iff `!nType && tCur >= tStart && PtInRect(rcArea, ptUser)`, and returns
//     `nSLV | (nSkillID << 8)` -- a MOB-skill disease descriptor.
//     CUserLocal::Update (v83 @0x94b7ba) calls it every frame and, on a hit,
//     computes `AFFECTEDAREA.nDamage (+0x34) * (100 - resist) / 100`
//     (@0x94b801) and damages the local user.
//   - == 2  CAffectedAreaPool::IsSmokeAreaByPoint (v95 @0x434f40) -- Smoke
//     Screen. v83 CAffectedAreaPool::Update (@0x43109f) also gates the
//     fade-out animation on `nType == 2`.
//   - == 3  CAffectedAreaPool::OnAffectedAreaCreated (v83 @0x431ade, v95
//     @0x437ec0) routes to CItemInfo::GetAreaBuffItem -- an area-buff ITEM,
//     not a skill.
//
// Nothing reads any other value: the per-skill construction path
// (CAffectedAreaPool::AffectedAreaAnimationCreated, v95 @0x4372c0) dispatches
// purely on nSkillID, and the user/party aura lookups
// (GetAffectAreaByPoint @0x4350f0, GetAr01AreaPAD/MAD) filter on
// nSkillID + dwOwnerID and ignore nType entirely.
//
// This is why a player-cast mist MUST NOT go out as 0. `nDamage` is written
// ONLY on the mob-skill arms of AffectedAreaAnimationCreated
// (`pa.p->nDamage = a[nSLV-1].nX` under `nSkillID == 130` / `== 131`); the
// 2111003 (Poison Mist) arm never touches it, so the field holds whatever was
// in the freshly-allocated AFFECTEDAREA. Sending nType 0 for Poison Mist made
// the v83 client find the caster standing inside their own cloud and bill them
// that uninitialized value -- observed live as a 1,434,803-damage self-hit
// (clamped to 999999) roughly one second after every cast.
//
// AffectedAreaTypeUserSkill is 1 because 1 is inert: the only requirement the
// client imposes is "not 0, not 2, not 3".
const (
	AffectedAreaTypeMobSkill  = int32(0)
	AffectedAreaTypeUserSkill = int32(1)
	// AffectedAreaTypeSmoke is 2 because the client's smoke lookup keys on it:
	// CAffectedAreaPool::IsSmokeAreaByPoint (v95 @0x434f40) rejects any area
	// whose nType != 2, and v83 CAffectedAreaPool::Update (@0x43109f) gates the
	// fade-out animation on the same value. A Smokescreen mist sent as 1 is
	// invisible to the client's own protection check.
	AffectedAreaTypeSmoke = int32(2)
)

// AffectedAreaTypeFor maps a mist's owner and effect to its nType. A
// monster-owned mist IS a mob disease cloud and must stay 0 -- that is what
// makes the client apply it to players standing in it (the pre-task-200
// AREA_POISON behaviour, which must not change). A character-owned
// PROTECTION mist is Smoke Screen (2); every other character-owned mist is a
// generic user skill area (1).
//
// Derived here rather than carried on COMMAND_TOPIC_MIST on purpose: nType is
// a client wire detail, and no producer should have to know the client's
// value table to create a mist.
func AffectedAreaTypeFor(ownerType string, effectKind string) int32 {
	if ownerType != OwnerTypeCharacter {
		return AffectedAreaTypeMobSkill
	}
	if effectKind == mistKafka.EffectKindProtection {
		return AffectedAreaTypeSmoke
	}
	return AffectedAreaTypeUserSkill
}

// Mist owner types, as carried on COMMAND_TOPIC_MIST CreateCommandBody.
const (
	OwnerTypeCharacter = "CHARACTER"
	OwnerTypeMonster   = "MONSTER"
)

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

// TargetKind reports who this mist's per-tick effect applies to: CHARACTER or
// MONSTER. Never empty on a mist built through Processor.Create, which
// normalizes an absent value to CHARACTER.
func (m Mist) TargetKind() string {
	return m.targetKind
}

// EffectKind reports what this mist's per-tick effect does: DISEASE or
// DAMAGE_OVER_TIME. Never empty on a mist built through Processor.Create.
func (m Mist) EffectKind() string {
	return m.effectKind
}

// RecoveryMp returns the per-tick MP a RECOVERY mist restores. 0 for every
// other effect kind.
func (m Mist) RecoveryMp() int32 {
	return m.recoveryMp
}

// PartyMemberIds returns the cast-time party snapshot a RECOVERY mist heals.
// A copy is returned: the tick fans out across goroutines (tasks.processTenant),
// so callers must not share this slice's backing array.
func (m Mist) PartyMemberIds() []uint32 {
	if len(m.partyMemberIds) == 0 {
		return nil
	}
	return append([]uint32(nil), m.partyMemberIds...)
}

// InPartySnapshot reports whether the character is in this mist's cast-time
// party snapshot. Always false for a mist with no snapshot, which is the
// correct answer for every non-RECOVERY kind.
func (m Mist) InPartySnapshot(characterId uint32) bool {
	for _, id := range m.partyMemberIds {
		if id == characterId {
			return true
		}
	}
	return false
}

// ElemAttr returns the AffectedAreaCreated `nElemAttr` wire value. The client
// stores it raw at AFFECTEDAREA+0x30 (v83 @0x431b3b, v95 @0x437fd9) and never
// reads it on any rendering path -- it takes the skill's element from its own
// Skill.wz. Atlas models no mist element, so this is 0 for every mist.
func (m Mist) ElemAttr() int32 {
	return m.elemAttr
}

// SkillDelay returns the AffectedAreaCreated `skillDelay` wire value: a
// DRAW DELAY in units of 100 ms, not a lifetime. The client computes
// tStart = get_update_time() + 100*skillDelay (v83 @0x431b50, v95 @0x437fa3)
// and gates the mist's first draw on it, so any non-zero value hides the mist
// for that long. Atlas has no per-mist cast delay to express: 0 = draw now.
func (m Mist) SkillDelay() int16 {
	return m.skillDelay
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

// Rect returns the mist's absolute axis-aligned bounding box in world
// coordinates: (x1, y1) top-left, (x2, y2) bottom-right. Bounds are inclusive,
// matching Contains and the atlas-monsters in-rect endpoint.
func (m Mist) Rect() (int16, int16, int16, int16) {
	return m.originX + m.ltX, m.originY + m.ltY, m.originX + m.rbX, m.originY + m.rbY
}

// Contains reports whether the given world coordinates fall within the mist's
// axis-aligned bounding box (inclusive of edges).
func (m Mist) Contains(x, y int16) bool {
	minX, minY, maxX, maxY := m.Rect()
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
	targetKind       string
	effectKind       string
	recoveryMp       int32
	partyMemberIds   []uint32
	elemAttr         int32
	skillDelay       int16
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

// SetKinds sets the target and effect descriptors. Grouped rather than split
// into two single-field setters because the pair is meaningless apart.
func (b *Builder) SetKinds(targetKind, effectKind string) *Builder {
	b.targetKind = targetKind
	b.effectKind = effectKind
	return b
}

// SetRecovery sets the RECOVERY magnitude and its cast-time party snapshot.
// Grouped rather than split because the pair is meaningless apart: a
// magnitude with no scope would heal the whole map.
func (b *Builder) SetRecovery(mp int32, partyMemberIds []uint32) *Builder {
	b.recoveryMp = mp
	b.partyMemberIds = append([]uint32(nil), partyMemberIds...)
	return b
}

// SetRender sets the client-render wire values carried on MIST_CREATED. Both
// are 0 for every mist Atlas creates; see the getters for why.
func (b *Builder) SetRender(elemAttr int32, skillDelay int16) *Builder {
	b.elemAttr = elemAttr
	b.skillDelay = skillDelay
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
		targetKind:       b.targetKind,
		effectKind:       b.effectKind,
		recoveryMp:       b.recoveryMp,
		partyMemberIds:   b.partyMemberIds,
		elemAttr:         b.elemAttr,
		skillDelay:       b.skillDelay,
		createdAt:        b.createdAt,
		expiresAt:        b.expiresAt,
		lastTick:         b.lastTick,
	}
}
