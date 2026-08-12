package mist

import (
	"atlas-maps/kafka/message"
	mistKafka "atlas-maps/kafka/message/mist"
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor exposes the lifecycle operations for tenant-scoped mists. Create
// inserts a new mist and emits MIST_CREATED; Destroy removes a mist and emits
// MIST_DESTROYED with the supplied reason.
type Processor interface {
	Create(body mistKafka.CreateCommandBody) (Mist, error)
	Destroy(id uuid.UUID, reason string) (Mist, error)
}

// ProcessorImpl is the default Processor backed by the singleton registry and
// the project's standard producer.Provider seam (so tests can inject a
// recording provider).
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	r   *Registry
}

// NewProcessor constructs the canonical Processor wired to the singleton
// registry and the supplied producer.Provider. Tenant is resolved from ctx.
func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor {
	return NewProcessorWithRegistry(l, ctx, p, GetRegistry())
}

var _ Processor = (*ProcessorImpl)(nil)

// NewProcessorWithRegistry constructs a Processor backed by the supplied
// registry instead of the singleton. Used by tick tasks and tests that need
// to operate on a non-singleton registry while reusing the lifecycle
// emission logic.
func NewProcessorWithRegistry(l logrus.FieldLogger, ctx context.Context, p producer.Provider, r *Registry) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   p,
		r:   r,
	}
}

// ErrUnknownKind is returned by Create when a command names a target or
// effect kind this service does not implement. FR-2.5: rejecting is the
// correct behaviour -- silently falling back to DISEASE would apply the
// wrong effect to the wrong targets, which is worse than creating no mist.
var ErrUnknownKind = errors.New("unknown mist target or effect kind")

func knownTargetKind(k string) bool {
	return k == mistKafka.TargetKindCharacter || k == mistKafka.TargetKindMonster
}

func knownEffectKind(k string) bool {
	switch k {
	case mistKafka.EffectKindDisease, mistKafka.EffectKindDamageOverTime,
		mistKafka.EffectKindProtection, mistKafka.EffectKindRecovery:
		return true
	}
	return false
}

// Create materialises a Mist from body, registers it under the resolved
// tenant, and emits MIST_CREATED. On emit failure the registry insert is
// rolled back so the registry stays in lockstep with downstream observers.
func (p *ProcessorImpl) Create(body mistKafka.CreateCommandBody) (Mist, error) {
	// Normalize the descriptors exactly once, here, so every Mist in the
	// registry has non-empty kinds and the tick task can switch on them
	// without an empty-string case. This is what gives the pre-task-200
	// atlas-monsters producer byte-for-byte unchanged behavior (FR-2.3).
	targetKind := body.TargetKind
	if targetKind == "" {
		targetKind = mistKafka.TargetKindCharacter
	}
	effectKind := body.EffectKind
	if effectKind == "" {
		effectKind = mistKafka.EffectKindDisease
	}

	if !knownTargetKind(targetKind) {
		p.l.Warnf("Mist create rejected: unknown targetKind [%s] from owner [%s:%d] on map [%d].", body.TargetKind, body.OwnerType, body.OwnerId, body.MapId)
		return Mist{}, ErrUnknownKind
	}
	if !knownEffectKind(effectKind) {
		p.l.Warnf("Mist create rejected: unknown effectKind [%s] from owner [%s:%d] on map [%d].", body.EffectKind, body.OwnerType, body.OwnerId, body.MapId)
		return Mist{}, ErrUnknownKind
	}

	id := uuid.New()
	f := field.NewBuilder(body.WorldId, body.ChannelId, body.MapId).SetInstance(body.Instance).Build()
	m := NewBuilder(id, f).
		SetOwner(body.OwnerType, body.OwnerId).
		SetOrigin(body.OriginX, body.OriginY).
		SetBounds(body.LtX, body.LtY, body.RbX, body.RbY).
		SetDisease(body.Disease, body.DiseaseValue, time.Duration(body.DiseaseDuration)*time.Millisecond).
		SetRecovery(body.RecoveryMp, body.PartyMemberIds).
		SetDuration(time.Duration(body.Duration)*time.Millisecond).
		SetTickInterval(time.Duration(body.TickIntervalMs)*time.Millisecond).
		SetSource(body.SourceSkillId, body.SourceSkillLevel).
		// nType is derived here rather than carried on the command -- see
		// AffectedAreaTypeFor. Leaving it at the zero value marks the mist as
		// a MOB disease cloud, which makes the client damage any player
		// standing in it, including the caster of a player-cast mist.
		SetType(AffectedAreaTypeFor(body.OwnerType)).
		SetKinds(targetKind, effectKind).
		Build()

	if err := p.r.Add(p.t, m); err != nil {
		return Mist{}, err
	}

	if err := message.Emit(p.p)(func(buf *message.Buffer) error {
		return buf.Put(mistKafka.EnvEventTopic, createdEventProvider(p.t, m))
	}); err != nil {
		// Roll back the registry insert so the registry never observes a
		// mist that downstream consumers will not see.
		_, _ = p.r.Remove(p.t, id)
		return Mist{}, err
	}
	return m, nil
}

// Destroy removes the mist with the given id from the tenant's bucket and
// emits MIST_DESTROYED with the supplied reason. Emit failures are logged
// but do not fail Destroy: the registry-side removal is authoritative.
func (p *ProcessorImpl) Destroy(id uuid.UUID, reason string) (Mist, error) {
	m, err := p.r.Remove(p.t, id)
	if err != nil {
		return Mist{}, err
	}
	if emitErr := message.Emit(p.p)(func(buf *message.Buffer) error {
		return buf.Put(mistKafka.EnvEventTopic, destroyedEventProvider(p.t, m, reason))
	}); emitErr != nil {
		p.l.WithError(emitErr).Errorf("Unable to emit MIST_DESTROYED for [%s].", id)
	}
	return m, nil
}
