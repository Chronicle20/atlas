package account

import (
	consumer2 "atlas-account/kafka/consumer"
	"strings"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerNameCreate = "create_account_command"
	consumerNameLogout = "logout_account_command"
)

func CreateAccountCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameCreate)(EnvCommandTopicCreateAccount)(groupId)
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
	t, _ := topic.EnvProvider(l)(EnvCommandTopicCreateAccount)()
	return t, message.AdaptHandler(message.PersistentConfig(handleCreateAccountCommand(db)))
}

func CreateAccountSessionCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameLogout)(EnvCommandTopicAccountLogout)(groupId)
	}
}

func handleLogoutAccountCommand(db *gorm.DB) func(l logrus.FieldLogger, span opentracing.Span, command logoutCommand) {
	return func(l logrus.FieldLogger, span opentracing.Span, command logoutCommand) {
		l.Debugf("Received logout account command account [%d] from [%s].", command.AccountId, command.Issuer)
		Logout(l, db, span, command.Tenant)(command.SessionId, command.AccountId, strings.ToUpper(command.Issuer))
		// l.Debugf("Ignoring logout command from [%s] as account [%d] is not in correct state.", command.Issuer, command.AccountId)
	}
}

func CreateAccountSessionRegister(l *logrus.Logger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicAccountLogout)()
	return t, message.AdaptHandler(message.PersistentConfig(handleLogoutAccountCommand(db)))
}
