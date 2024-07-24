package session

import (
	"atlas-account/account"
	consumer2 "atlas-account/kafka/consumer"
	"atlas-account/kafka/producer"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerNameLogout = "logout_account_command"
)

func CreateAccountSessionCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameLogout)(EnvCommandTopicAccountLogout)(groupId)
	}
}

func handleLogoutAccountCommand(db *gorm.DB) message.Handler[logoutCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command logoutCommand) {
		pi := producer.ProviderImpl(l)(span)
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
			_ = pi(EnvEventTopicAccountStatus)(loggedOutEventProvider()(command.Tenant, command.AccountId, a.Name()))
			return
		}

		if command.Issuer == "channel" && a.State() >= 2 {
			err = account.SetLoggedOut(db)(command.Tenant, command.AccountId)
			if err != nil {
				l.WithError(err).Errorf("Error processing command to logout account [%d].", command.AccountId)
				return
			}
			_ = pi(EnvEventTopicAccountStatus)(loggedOutEventProvider()(command.Tenant, command.AccountId, a.Name()))
			return
		}

		l.Debugf("Ignoring logout command from [%s] as account [%d] is not in correct state.", command.Issuer, command.AccountId)
	}
}

func CreateAccountSessionRegister(l *logrus.Logger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopicAccountLogout)()
	return t, message.AdaptHandler(message.PersistentConfig(handleLogoutAccountCommand(db)))
}
