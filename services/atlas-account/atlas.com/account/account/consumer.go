package account

import (
	consumer2 "atlas-account/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerNameCreate = "create_account_command"
)

func CreateCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameCreate)(EnvCommandTopicCreateAccount)(groupId)
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

func CreateAccountRegister(l *logrus.Logger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicCreateAccount)()
	return t, message.AdaptHandler(message.PersistentConfig(handleCreateAccountCommand(db)))
}
