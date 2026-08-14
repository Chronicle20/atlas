package mist

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

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
//
// Build intentionally performs NO invariant validation (positive duration,
// non-degenerate rectangle, non-zero nType for a character-owned mist, etc.)
// even though file-responsibilities.md's builder.go contract calls for
// "Build() enforces invariants". Every one of those candidate invariants is
// contradicted by an existing, currently-passing test that constructs a Mist
// through this exact Builder expecting Build() to succeed:
//
//   - mist/model_test.go TestMist_Expired_AfterDuration calls
//     SetDuration(0).Build() and asserts Expired() is true — a zero-duration
//     mist is a deliberately exercised construction, not corrupt input.
//   - mist/model_test.go TestMist_Kinds_RoundTrip and
//     TestMist_Rect_AgreesWithContains never call SetDuration or SetBounds
//     with a non-degenerate rect, relying on the zero values.
//   - tasks/mist_tick_monster_test.go's mkMonsterMist and the inline builder
//     in TestMistTick_MonsterTarget_DotTickIntervalStrictlyLessThanReapplyInterval
//     both build a CHARACTER-owned Mist without ever calling SetType, so
//     mistType stays 0 -- the exact "nType 0 for a character-owned mist"
//     shape the model.go doc comment (AffectedAreaTypeFor) calls dangerous
//     on the wire, but which this in-memory construction path must keep
//     accepting.
//
// Rejecting any of these at Build() would require changing Build()'s
// signature to return an error and updating every call site (production and
// test) that currently treats it as infallible -- an API and behaviour
// change, not the pure move this file exists to make. The kinds that DO have
// a validated invariant (TargetKind/EffectKind) are already enforced one
// layer up, in Processor.Create's knownTargetKind/knownEffectKind gate,
// before NewBuilder is ever reached -- Build() itself is never the last line
// of defense for those. See docs/tasks/task-218-player-cast-mists/context.md
// §5 and the final-fix-report for the full evidence trail.
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
