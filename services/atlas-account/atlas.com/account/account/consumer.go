package account

import (
	"atlas-account/kafka"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerNameCreate = "create_account_command"
)

func CreateAccountCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerNameCreate)(EnvCommandTopicCreateAccount)(groupId)
	}
}

func handleCreateAccountCommand(db *gorm.DB) message.Handler[createCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command createCommand) {
		l.Debugf("Received create account command name [%s] password [%s].", command.Name, command.Password)
		_, err := Create(l, db, span, command.Tenant)(command.Name, command.Password)
		if err != nil {
			l.WithError(err).Errorf("Error processing command to create account [%s].", command.Name)
			return
		}
	}
}

func CreateAccountRegister(l *logrus.Logger, db *gorm.DB) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvCommandTopicCreateAccount), message.AdaptHandler(message.PersistentConfig(handleCreateAccountCommand(db)))
}
