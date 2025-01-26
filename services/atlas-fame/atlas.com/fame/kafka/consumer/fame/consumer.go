package fame

import (
	"atlas-fame/fame"
	consumer2 "atlas-fame/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("fame_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestChangeCommand(db))))
		}
	}
}

func handleRequestChangeCommand(db *gorm.DB) message.Handler[command[requestChangeCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[requestChangeCommandBody]) {
		if c.Type != CommandTypeRequestChange {
			return
		}
		_ = fame.RequestChange(l)(ctx)(db)(c.WorldId, c.Body.ChannelId, c.CharacterId, c.Body.MapId, c.Body.TargetId, c.Body.Amount)
	}
}
