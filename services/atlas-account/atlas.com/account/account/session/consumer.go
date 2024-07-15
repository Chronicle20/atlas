package session

import (
	"atlas-account/account"
	"atlas-account/kafka"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerNameLogout = "logout_account_command"
)

func CreateAccountSessionCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerNameLogout)(EnvCommandTopicAccountLogout)(groupId)
	}
}

func handleLogoutAccountCommand(db *gorm.DB) message.Handler[logoutCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command logoutCommand) {
		l.Debugf("Received logout account command account [%d] from [%s].", command.AccountId, command.Issuer)
		a, err := account.GetById(l, db, command.Tenant)(command.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate account [%d].", command.AccountId)
			return
		}

		if command.Issuer == "login" && a.State() <= 1 {
			err = account.SetLoggedOut(db)(command.Tenant, command.AccountId)
			if err != nil {
				l.WithError(err).Errorf("Error processing command to logout account [%d].", command.AccountId)
				return
			}
			emitLoggedOutEvent(l, span, command.Tenant)
			return
		}

		if command.Issuer == "channel" && a.State() >= 2 {
			err = account.SetLoggedOut(db)(command.Tenant, command.AccountId)
			if err != nil {
				l.WithError(err).Errorf("Error processing command to logout account [%d].", command.AccountId)
				return
			}
			emitLoggedOutEvent(l, span, command.Tenant)
			return
		}

		l.Debugf("Ignoring logout command from [%s] as account [%d] is not in correct state.", command.Issuer, command.AccountId)
	}
}

func CreateAccountSessionRegister(l *logrus.Logger, db *gorm.DB) (string, handler.Handler) {
	return kafka.LookupTopic(l)(EnvCommandTopicAccountLogout), message.AdaptHandler(message.PersistentConfig(handleLogoutAccountCommand(db)))
}
