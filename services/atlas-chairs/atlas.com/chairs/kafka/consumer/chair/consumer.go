package chair

import (
	"atlas-chairs/chair"
	consumer2 "atlas-chairs/kafka/consumer"
	chair2 "atlas-chairs/kafka/message/chair"
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
			rf(consumer2.NewConfig(l)("chair_command")(chair2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(chair2.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandUseChair)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandCancelChair)))
	}
}

func handleCommandUseChair(l logrus.FieldLogger, ctx context.Context, c chair2.Command[chair2.UseChairCommandBody]) {
	if c.Type != chair2.CommandUseChair {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).Build()
	_ = chair.NewProcessor(l, ctx).Set(f, c.Body.ChairType, c.Body.ChairId, c.Body.CharacterId)
}

func handleCommandCancelChair(l logrus.FieldLogger, ctx context.Context, c chair2.Command[chair2.CancelChairCommandBody]) {
	if c.Type != chair2.CommandCancelChair {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).Build()
	_ = chair.NewProcessor(l, ctx).Clear(f, c.Body.CharacterId)
}
