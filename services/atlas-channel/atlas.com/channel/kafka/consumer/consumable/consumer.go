package consumable

import (
	consumer2 "atlas-channel/kafka/consumer"
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
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("consumable_command")(EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleScrollConsumableEvent(sc, wp))))
			}
		}
	}
}

func handleScrollConsumableEvent(sc server.Model, wp writer.Producer) message.Handler[Event[ScrollBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e Event[ScrollBody]) {
		err := session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			return _map.ForSessionsInMap(l)(ctx)(s.Map(), session.Announce(l)(ctx)(wp)(writer.CharacterItemUpgrade)(writer.CharacterItemUpgradeBody(e.CharacterId, e.Body.Success, e.Body.Cursed, e.Body.LegendarySpirit, e.Body.WhiteScroll)))
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to process scroll event for character [%d].", e.CharacterId)
		}
	}
}
