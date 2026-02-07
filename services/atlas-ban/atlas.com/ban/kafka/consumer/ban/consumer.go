package ban

import (
	"atlas-ban/ban"
	consumer2 "atlas-ban/kafka/consumer"
	ban2 "atlas-ban/kafka/message/ban"
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
			rf(consumer2.NewConfig(l)("ban_command")(ban2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(ban2.EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateBanCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDeleteBanCommand(db))))
		}
	}
}

func handleCreateBanCommand(db *gorm.DB) message.Handler[ban2.Command[ban2.CreateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c ban2.Command[ban2.CreateCommandBody]) {
		if c.Type != ban2.CommandTypeCreate {
			return
		}
		l.Debugf("Received create ban command type [%d] value [%s].", c.Body.BanType, c.Body.Value)
		_, err := ban.NewProcessor(l, ctx, db).CreateAndEmit(
			ban.BanType(c.Body.BanType),
			c.Body.Value,
			c.Body.Reason,
			c.Body.ReasonCode,
			c.Body.Permanent,
			c.Body.ExpiresAt,
			c.Body.IssuedBy,
		)
		if err != nil {
			l.WithError(err).Errorf("Error processing command to create ban for value [%s].", c.Body.Value)
			return
		}
	}
}

func handleDeleteBanCommand(db *gorm.DB) message.Handler[ban2.Command[ban2.DeleteCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c ban2.Command[ban2.DeleteCommandBody]) {
		if c.Type != ban2.CommandTypeDelete {
			return
		}
		l.Debugf("Received delete ban command for ban [%d].", c.Body.BanId)
		err := ban.NewProcessor(l, ctx, db).DeleteAndEmit(c.Body.BanId)
		if err != nil {
			l.WithError(err).Errorf("Error processing command to delete ban [%d].", c.Body.BanId)
			return
		}
	}
}
