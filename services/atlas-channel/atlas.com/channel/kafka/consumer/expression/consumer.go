package expression

import (
	consumer2 "atlas-channel/kafka/consumer"
	expression2 "atlas-channel/kafka/message/expression"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

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
			rf(consumer2.NewConfig(l)("expression_event")(expression2.EnvExpressionEvent)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(expression2.EnvExpressionEvent)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEvent(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleEvent(sc server.Model, wp writer.Producer) message.Handler[expression2.Event] {
	return func(l logrus.FieldLogger, ctx context.Context, e expression2.Event) {
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.CharacterId, session.Announce(l)(ctx)(wp)(writer.CharacterExpression)(writer.CharacterExpressionBody(e.CharacterId, e.Expression)))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] expression [%d] change to characters in map [%d].", e.CharacterId, e.Expression, e.MapId)
		}
	}
}
