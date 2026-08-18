package mist

import (
	consumer2 "atlas-channel/kafka/consumer"
	mist2 "atlas-channel/kafka/message/mist"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/mist"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mist_event")(mist2.EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(mist2.EnvEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMistCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleMistDestroyed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// affectedAreaCreatedBroadcaster is the channel-side broadcast seam for the
// MIST_CREATED -> AffectedAreaCreated translation. Held as a package-level
// var so tests can swap in a recording stub without standing up a REST mock
// for _map.ForSessionsInMap. The default preserves the production behaviour
// of announcing through wp + session.Announce via _map.ForSessionsInMap.
var affectedAreaCreatedBroadcaster = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, body fieldpkt.AffectedAreaCreated) {
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f,
		session.Announce(l)(ctx)(wp)(fieldpkt.AffectedAreaCreatedWriter)(body.Encode))
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast AffectedAreaCreated for mist [%s].", body.MistId())
	}
}

var affectedAreaRemovedBroadcaster = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, body fieldpkt.AffectedAreaRemoved) {
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f,
		session.Announce(l)(ctx)(wp)(fieldpkt.AffectedAreaRemovedWriter)(body.Encode))
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast AffectedAreaRemoved for mist [%s].", body.MistId())
	}
}

// The `skillDelay` and `nElemAttr` wire values now travel on MIST_CREATED
// (atlas-maps mist.Mist.SkillDelay() / .ElemAttr(), where the full IDB
// rationale lives). Both are 0 for every mist Atlas creates today.
//
// mistPhase is the GMS v92+ `nPhase` wire value (AFFECTEDAREA+0x48, v95
// @0x437fde). It is compared for equality only inside IsSmokeAreaByPoint /
// GetAffectAreaByPoint, neither of which any Atlas mist can reach (task-200
// design §3.3-3.4). Atlas does not model it; 0 matches the legacy versions,
// which omit the field entirely.
const mistPhase = int32(0)

// protectionRegistry is the channel-local index of live PROTECTION
// (Smokescreen) mists, consulted by the damage path. Held as a package var so
// tests can point it at an isolated registry.
//
// A protection mist is recognised by its EffectKind rather than by the
// client-facing `Type` (nType 2): nType is a render detail, and inferring the
// domain concept from it would couple channel logic to the client's value
// table -- exactly what AffectedAreaTypeFor's doc comment exists to prevent.
var protectionRegistry = mist.GetProtectionRegistry()

func handleMistCreated(sc server.Model, wp writer.Producer) message.Handler[mist2.Event[mist2.CreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mist2.Event[mist2.CreatedBody]) {
		if e.Type != mist2.EventTypeCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		body := fieldpkt.NewAffectedAreaCreated(
			e.MistId,
			e.Body.OwnerId,
			e.Body.Type,
			int32(e.Body.SourceSkillId),
			byte(e.Body.SourceSkillLevel),
			e.Body.SkillDelay,
			e.Body.OriginX, e.Body.OriginY,
			e.Body.LtX, e.Body.LtY,
			e.Body.RbX, e.Body.RbY,
			e.Body.ElemAttr,
			mistPhase,
		)
		affectedAreaCreatedBroadcaster(l, ctx, wp, f, body)

		if e.Body.EffectKind == mist2.EffectKindProtection {
			protectionRegistry.Add(tenant.MustFromContext(ctx),
				mist.NewProtectionBuilder(e.MistId, f).
					SetOwnerId(e.Body.OwnerId).
					// Absolute world rect: the event carries the origin
					// and the lt/rb OFFSETS, and the damage path tests a
					// character's absolute position.
					SetRect(e.Body.OriginX+e.Body.LtX, e.Body.OriginY+e.Body.LtY,
						e.Body.OriginX+e.Body.RbX, e.Body.OriginY+e.Body.RbY).
					SetExpiresAt(time.Now().Add(time.Duration(e.Body.Duration)*time.Millisecond)).
					Build())
		}
	}
}

func handleMistDestroyed(sc server.Model, wp writer.Producer) message.Handler[mist2.Event[mist2.DestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e mist2.Event[mist2.DestroyedBody]) {
		if e.Type != mist2.EventTypeDestroyed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		body := fieldpkt.NewAffectedAreaRemoved(e.MistId, 0)
		affectedAreaRemovedBroadcaster(l, ctx, wp, f, body)

		// Unconditional: removing an id that was never a protection mist
		// is a no-op, and this way a PROTECTION mist can never survive
		// its own destruction because of a kind check that drifted.
		protectionRegistry.Remove(tenant.MustFromContext(ctx), e.MistId)
	}
}
