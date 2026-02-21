package reactor

import (
	consumer2 "atlas-channel/kafka/consumer"
	reactor2 "atlas-channel/kafka/message/reactor"
	_map "atlas-channel/map"
	"atlas-channel/reactor"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("reactor_status_event")(reactor2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(reactor2.EnvEventStatusTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyed(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleHit(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[reactor2.StatusEvent[reactor2.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e reactor2.StatusEvent[reactor2.CreatedStatusEventBody]) {
		if e.Type != reactor2.EventStatusTypeCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		r := reactor.NewModelBuilder(f, e.Body.Classification, e.Body.Name).
			SetId(e.ReactorId).
			SetState(e.Body.State).
			SetEventState(e.Body.EventState).
			SetPosition(e.Body.X, e.Body.Y).
			SetDelay(e.Body.Delay).
			SetDirection(e.Body.Direction).
			MustBuild()

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), session.Announce(l)(ctx)(wp)(writer.ReactorSpawn)(writer.ReactorSpawnBody()(r)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn reactor [%d] for characters in map [%d].", r.Id(), e.MapId)
		}
	}
}

func handleDestroyed(sc server.Model, wp writer.Producer) message.Handler[reactor2.StatusEvent[reactor2.DestroyedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e reactor2.StatusEvent[reactor2.DestroyedStatusEventBody]) {
		if e.Type != reactor2.EventStatusTypeDestroyed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), session.Announce(l)(ctx)(wp)(writer.ReactorDestroy)(writer.ReactorDestroyBody()(e.ReactorId, e.Body.State, e.Body.X, e.Body.Y)))
		if err != nil {
			l.WithError(err).Errorf("Unable to destroy reactor [%d] for characters in map [%d].", e.ReactorId, e.MapId)
		}
	}
}

func handleHit(sc server.Model, wp writer.Producer) message.Handler[reactor2.StatusEvent[reactor2.HitStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e reactor2.StatusEvent[reactor2.HitStatusEventBody]) {
		if e.Type != reactor2.EventStatusTypeHit {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		r := reactor.NewModelBuilder(f, e.Body.Classification, "").
			SetId(e.ReactorId).
			SetState(e.Body.State).
			SetPosition(e.Body.X, e.Body.Y).
			SetDirection(e.Body.Direction).
			MustBuild()

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), session.Announce(l)(ctx)(wp)(writer.ReactorHit)(writer.ReactorHitBody()(r)))
		if err != nil {
			l.WithError(err).Errorf("Unable to send reactor hit [%d] to characters in map [%d].", r.Id(), e.MapId)
		}
	}
}
