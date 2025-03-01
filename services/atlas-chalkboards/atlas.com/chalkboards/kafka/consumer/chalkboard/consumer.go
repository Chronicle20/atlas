package character

import (
	"atlas-chalkboards/chalkboard"
	consumer2 "atlas-chalkboards/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chalkboard_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSetCommand)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleClearCommand)))
	}
}

func handleSetCommand(l logrus.FieldLogger, ctx context.Context, c commandEvent[setCommandBody]) {
	if c.Type != CommandChalkboardSet {
		return
	}
	_ = chalkboard.Set(l)(ctx)(c.WorldId, c.ChannelId, c.MapId, c.CharacterId, c.Body.Message)
}

func handleClearCommand(l logrus.FieldLogger, ctx context.Context, c commandEvent[clearCommandBody]) {
	if c.Type != CommandChalkboardClear {
		return
	}
	_ = chalkboard.Clear(l)(ctx)(c.WorldId, c.ChannelId, c.MapId, c.CharacterId)
}
