package summon

import (
	consumer2 "atlas-channel/kafka/consumer"
	summon2 "atlas-channel/kafka/message/summon"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	summonpkt "github.com/Chronicle20/atlas/libs/atlas-packet/summon/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("summon_status_event")(summon2.EnvEventTopicSummonStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(summon2.EnvEventTopicSummonStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[summon2.StatusEvent[summon2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon2.StatusEvent[summon2.StatusEventCreatedBody]) {
		if e.Type != summon2.EventSummonStatusCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(summonpkt.SummonSpawnWriter)(
				writer.SummonSpawnBody(e.OwnerCharacterId, e.SummonId, e.SkillId, e.Body.SkillLevel, e.Body.X, e.Body.Y, e.Body.Stance, e.Body.MovementType, e.Body.Puppet, e.Body.Animated)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn summon [%d] for characters in map [%d].", e.SummonId, e.MapId)
		}
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[summon2.StatusEvent[summon2.StatusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon2.StatusEvent[summon2.StatusEventDestroyedBody]) {
		if e.Type != summon2.EventSummonStatusDestroyed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(summonpkt.SummonRemoveWriter)(
				writer.SummonRemoveBody(e.OwnerCharacterId, e.SummonId, e.Body.Animated)))
		if err != nil {
			l.WithError(err).Errorf("Unable to remove summon [%d] for characters in map [%d].", e.SummonId, e.MapId)
		}
	}
}
