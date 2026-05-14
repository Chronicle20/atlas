package expression

import (
	consumer2 "atlas-channel/kafka/consumer"
	expression2 "atlas-channel/kafka/message/expression"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

		// TODO(task-028 follow-up): Kafka expression.Event doesn't carry duration
		// or byItemOption, so v95+ and JMS clients always observe duration=0 and
		// byItemOption=false. Extend kafka/message/expression/kafka.go to carry the
		// fields end-to-end (producer side too). Tracked in
		// docs/tasks/task-028-character-domain-audit/post-phase-b.md "Remaining work".
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.CharacterId, session.Announce(l)(ctx)(wp)(charpkt.CharacterExpressionWriter)(charpkt.NewCharacterExpression(e.CharacterId, e.Expression, 0).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] expression [%d] change to characters in map [%d].", e.CharacterId, e.Expression, e.MapId)
		}
	}
}
