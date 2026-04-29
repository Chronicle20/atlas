package mist

import (
	consumer2 "atlas-channel/kafka/consumer"
	mist2 "atlas-channel/kafka/message/mist"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mist_event")(mist2.EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(mist2.EnvEventTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMistCreated(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMistDestroyed(sc, wp)))); err != nil {
					return err
				}
				return nil
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
			e.Body.OriginX, e.Body.OriginY,
			e.Body.LtX, e.Body.LtY,
			e.Body.RbX, e.Body.RbY,
			e.Body.Duration,
			0,
		)
		affectedAreaCreatedBroadcaster(l, ctx, wp, f, body)
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
	}
}
