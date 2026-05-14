package timer

import (
	"context"
	"time"

	"atlas-maps/kafka/message"
	characterKafka "atlas-maps/kafka/message/character"
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
	CancelIfTracked(characterId uint32) bool
	ForceReturnIfTracked(characterId uint32) bool
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

	// Block the AfterFunc callback until p.r.Add has installed the entry into
	// the registry. Without this gate, time.AfterFunc(small) can schedule the
	// callback fast enough that handleExpire's Claim races p.r.Add for the
	// registry mutex, finds nothing, and silently no-ops — leaving the timer
	// permanently inert. Forced returns then never emit CHANGE_MAP.
	ready := make(chan struct{})
	t := time.AfterFunc(duration, func() {
		<-ready
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
		close(ready)
		return err
	}
	close(ready)

	if err := message.Emit(p.p)(func(buf *message.Buffer) error {
		return buf.Put(mapKafka.EnvEventTopicMapStatus, mapTimerStartedProvider(transactionId, f, characterId, seconds))
	}); err != nil {
		p.l.WithError(err).Warnf("MapTimer.Register: failed to emit MAP_TIMER_STARTED for character [%d] map [%d].", characterId, f.MapId())
	}
	p.l.Infof("MapTimer.Start: tenant=[%s] character=[%d] map=[%d] forcedReturn=[%d] seconds=[%d].", p.t.Id(), characterId, f.MapId(), forcedReturnMapId, seconds)
	return nil
}

// handleExpire is the time.Timer callback fired when the per-entry timer
// elapses. It atomically claims the entry (no-op when the token is stale due
// to a Register/Cancel race) and emits CHANGE_MAP to the forced-return map.
func (p *ProcessorImpl) handleExpire(tt tenant.Model, characterId uint32, token uuid.UUID) {
	entry, claimed := p.r.Claim(tt, characterId, token)
	if !claimed {
		return
	}
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), "MapTimer.Expire")
	span.SetAttributes(
		attribute.String("tenant.id", tt.Id().String()),
		attribute.Int("world.id", int(entry.Field().WorldId())),
		attribute.Int("map.id", int(entry.Field().MapId())),
		attribute.Int("forced.return.map.id", int(entry.ForcedReturnMapId())),
	)
	defer span.End()
	if err := p.emitChangeMap(entry); err != nil {
		p.l.WithError(err).Errorf("MapTimer.Expire: failed to emit CHANGE_MAP for character [%d].", characterId)
		return
	}
	p.l.Warnf("MapTimer.Expire: tenant=[%s] character=[%d] map=[%d] forcedReturn=[%d].", tt.Id(), characterId, entry.Field().MapId(), entry.ForcedReturnMapId())
}

func (p *ProcessorImpl) CancelIfTracked(characterId uint32) bool {
	prior, ok := p.r.Cancel(p.t, characterId)
	if !ok {
		return false
	}
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "MapTimer.Cancel")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.Int("world.id", int(prior.Field().WorldId())),
		attribute.Int("map.id", int(prior.Field().MapId())),
	)
	defer span.End()
	if prior.Timer() != nil {
		prior.Timer().Stop()
	}
	p.l.Infof("MapTimer.Cancel: tenant=[%s] character=[%d] map=[%d].", p.t.Id(), characterId, prior.Field().MapId())
	return true
}

// ForceReturnIfTracked is invoked on disconnect: if the character has a
// tracked entry it is removed unconditionally so the per-entry timer stops
// firing. Forced-return persistence is handled by location.Resolve at next
// login, so no CHANGE_MAP is emitted here.
func (p *ProcessorImpl) ForceReturnIfTracked(characterId uint32) bool {
	entry, ok := p.r.ClaimAny(p.t, characterId)
	if !ok {
		return false
	}
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "MapTimer.Disconnect")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.Int("world.id", int(entry.Field().WorldId())),
		attribute.Int("map.id", int(entry.Field().MapId())),
	)
	defer span.End()
	if entry.Timer() != nil {
		entry.Timer().Stop()
	}
	p.l.Warnf("MapTimer.Disconnect: tenant=[%s] character=[%d] map=[%d] (forced-return persistence handled by location.Resolve).",
		p.t.Id(), characterId, entry.Field().MapId())
	return true
}

func (p *ProcessorImpl) emitChangeMap(entry Entry) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return buf.Put(characterKafka.EnvCommandTopic, changeMapProvider(uuid.New(), entry.CharacterId(), entry.Field().WorldId(), entry.Field().ChannelId(), entry.ForcedReturnMapId()))
	})
}
