package incubator

import (
	consumer2 "atlas-channel/kafka/consumer"
	incubator2 "atlas-channel/kafka/message/incubator"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	incubatorcb "github.com/Chronicle20/atlas/libs/atlas-packet/incubator/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("incubator_result")(incubator2.EnvEventTopicIncubatorResult)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(incubator2.EnvEventTopicIncubatorResult)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleResult(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleResult(sc server.Model, wp writer.Producer) message.Handler[incubator2.ResultEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, event incubator2.ResultEvent) {
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, world.Id(event.WorldId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(event.CharacterId,
			session.Announce(l)(ctx)(wp)(incubatorcb.IncubatorResultWriter)(incubatorcb.NewIncubatorResult(event.ItemId, uint16(event.Count), 0).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce incubator result to character [%d].", event.CharacterId)
		}
	}
}
