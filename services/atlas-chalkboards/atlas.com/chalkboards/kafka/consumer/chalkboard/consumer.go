package character

import (
	"atlas-chalkboards/chalkboard"
	consumer2 "atlas-chalkboards/kafka/consumer"
	chalkboard2 "atlas-chalkboards/kafka/message/chalkboard"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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
			rf(consumer2.NewConfig(l)("chalkboard_command")(chalkboard2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(chalkboard2.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSetCommand))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClearCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleSetCommand(l logrus.FieldLogger, ctx context.Context, c chalkboard2.Command[chalkboard2.SetCommandBody]) {
	if c.Type != chalkboard2.CommandChalkboardSet {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).Build()
	_ = chalkboard.NewProcessor(l, ctx).Set(c.TransactionId, f, c.CharacterId, c.Body.Message)
}

func handleClearCommand(l logrus.FieldLogger, ctx context.Context, c chalkboard2.Command[chalkboard2.ClearCommandBody]) {
	if c.Type != chalkboard2.CommandChalkboardClear {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).Build()
	_ = chalkboard.NewProcessor(l, ctx).Clear(c.TransactionId, f, c.CharacterId)
}
