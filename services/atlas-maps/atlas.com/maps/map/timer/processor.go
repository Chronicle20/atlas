package timer

import (
	"context"
	"time"

	"atlas-maps/kafka/message"
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type Processor interface {
	Register(transactionId uuid.UUID, characterId uint32, f field.Model, forcedReturnMapId _map.Id, seconds uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	p   producer.Provider
	r   *Registry
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider) Processor {
	return NewProcessorWithRegistry(l, ctx, p, GetRegistry())
}

func NewProcessorWithRegistry(l logrus.FieldLogger, ctx context.Context, p producer.Provider, r *Registry) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		p:   p,
		r:   r,
	}
}

// Register inserts (or replaces) the timer entry for (tenant, characterId),
// schedules a per-entry time.Timer, and publishes MAP_TIMER_STARTED so
// atlas-channel can render the countdown.
func (p *ProcessorImpl) Register(transactionId uuid.UUID, characterId uint32, f field.Model, forcedReturnMapId _map.Id, seconds uint32) error {
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "MapTimer.Start")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.Int("world.id", int(f.WorldId())),
		attribute.Int("map.id", int(f.MapId())),
		attribute.Int("forced.return.map.id", int(forcedReturnMapId)),
	)
	defer span.End()
	if prior, ok := p.r.Cancel(p.t, characterId); ok {
		if prior.Timer() != nil {
			prior.Timer().Stop()
		}
	}

	tok := uuid.New()
	duration := time.Duration(seconds) * time.Second
	expiresAt := time.Now().Add(duration)
	t := time.AfterFunc(duration, func() {
		p.handleExpire(p.t, characterId, tok)
	})

	entry := NewEntryBuilder().
		SetTenant(p.t).
		SetCharacterId(characterId).
		SetField(f).
		SetForcedReturnMapId(forcedReturnMapId).
		SetSeconds(seconds).
		SetToken(tok).
		SetExpiresAt(expiresAt).
		SetTimer(t).
		Build()
	if err := p.r.Add(entry); err != nil {
		t.Stop()
		return err
	}

	if err := message.Emit(p.p)(func(buf *message.Buffer) error {
		return buf.Put(mapKafka.EnvEventTopicMapStatus, mapTimerStartedProvider(transactionId, f, characterId, seconds))
	}); err != nil {
		p.l.WithError(err).Warnf("MapTimer.Register: failed to emit MAP_TIMER_STARTED for character [%d] map [%d].", characterId, f.MapId())
	}
	p.l.Infof("MapTimer.Start: tenant=[%s] character=[%d] map=[%d] forcedReturn=[%d] seconds=[%d].", p.t.Id(), characterId, f.MapId(), forcedReturnMapId, seconds)
	return nil
}

// handleExpire is the time.Timer callback. Stub for now — real impl in Task 13.
func (p *ProcessorImpl) handleExpire(t tenant.Model, characterId uint32, token uuid.UUID) {
	_, _ = p.r.Claim(t, characterId, token)
}
