package mist

import (
	"atlas-maps/kafka/message"
	mistKafka "atlas-maps/kafka/message/mist"
	"atlas-maps/kafka/producer"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   p,
		r:   GetRegistry(),
	}
}

// Create materialises a Mist from body, registers it under the resolved
// tenant, and emits MIST_CREATED. On emit failure the registry insert is
// rolled back so the registry stays in lockstep with downstream observers.
func (p *ProcessorImpl) Create(body mistKafka.CreateCommandBody) (Mist, error) {
	id := uuid.New()
	f := field.NewBuilder(body.WorldId, body.ChannelId, body.MapId).SetInstance(body.Instance).Build()
	m := NewBuilder(id, f).
		SetOwner(body.OwnerType, body.OwnerId).
		SetOrigin(body.OriginX, body.OriginY).
		SetBounds(body.LtX, body.LtY, body.RbX, body.RbY).
		SetDisease(body.Disease, body.DiseaseValue, time.Duration(body.DiseaseDuration)*time.Millisecond).
		SetDuration(time.Duration(body.Duration) * time.Millisecond).
		SetTickInterval(time.Duration(body.TickIntervalMs) * time.Millisecond).
		SetSource(body.SourceSkillId, body.SourceSkillLevel).
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
