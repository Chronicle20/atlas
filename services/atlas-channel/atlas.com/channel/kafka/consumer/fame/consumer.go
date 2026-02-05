package fame

import (
	consumer2 "atlas-channel/kafka/consumer"
	fame2 "atlas-channel/kafka/message/fame"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
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
			rf(consumer2.NewConfig(l)("fame_event_status")(fame2.EnvEventTopicFameStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(fame2.EnvEventTopicFameStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleFameEventStatusError(sc, wp))))
			}
		}
	}
}

func handleFameEventStatusError(sc server.Model, wp writer.Producer) message.Handler[fame2.StatusEvent[fame2.StatusEventErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e fame2.StatusEvent[fame2.StatusEventErrorBody]) {
		if e.Type != fame2.StatusEventTypeError {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.Body.ChannelId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, fameResponseError(l)(ctx)(wp)(e.Body.Error))
		if err != nil {
			l.WithError(err).Errorf("Unable to fame error [%s] response to character [%d].", e.Body.Error, e.CharacterId)
		}
	}
}

func fameResponseError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(errCode string) model.Operator[session.Model] {
			return func(errCode string) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.FameResponse)(writer.FameResponseErrorBody(l)(errCode))
			}
		}
	}
}
