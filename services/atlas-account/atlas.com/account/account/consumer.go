package account

import (
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerNameCreate = "create_account_command"
)

func CreateAccountCommandConsumer(l logrus.FieldLogger, db *gorm.DB) func(groupId string) consumer.Config {
	t := lookupTopic(l)(EnvCommandTopicCreateAccount)
	return func(groupId string) consumer.Config {
		return consumer.NewConfig[createCommand](consumerNameCreate, t, groupId, handleCreateAccountCommand(db))
	}
}

func handleCreateAccountCommand(db *gorm.DB) consumer.HandlerFunc[createCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command createCommand) {
		l.Debugf("Received create account command name [%s] password [%s].", command.Name, command.Password)
		_, err := Create(l, db, span, command.Tenant)(command.Name, command.Password)
		if err != nil {
			l.WithError(err).Errorf("Error processing command to create account [%s].", command.Name)
			return
		}
	}
}
