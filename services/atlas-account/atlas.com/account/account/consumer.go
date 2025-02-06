package account

import (
	consumer2 "atlas-account/kafka/consumer"
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
			rf(consumer2.NewConfig(l)("create_account_command")(EnvCommandTopicCreateAccount)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopicCreateAccount)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateAccountCommand(db))))
		}
	}
}

func handleCreateAccountCommand(db *gorm.DB) message.Handler[createCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, command createCommand) {
		l.Debugf("Received create account command name [%s] password [%s].", command.Name, command.Password)
		_, err := Create(l, db, ctx)(command.Name, command.Password)
		if err != nil {
			l.WithError(err).Errorf("Error processing command to create account [%s].", command.Name)
			return
		}
	}
}
