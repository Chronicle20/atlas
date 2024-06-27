package session

import (
	"atlas-account/account"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	consumerNameLogout = "logout_account_command"
)

func CreateAccountSessionCommandConsumer(l logrus.FieldLogger, db *gorm.DB) func(groupId string) consumer.Config {
	t := lookupTopic(l)(EnvCommandTopicAccountLogout)
	return func(groupId string) consumer.Config {
		return consumer.NewConfig[logoutCommand](consumerNameLogout, t, groupId, handleLogoutAccountCommand(db))
	}
}

func handleLogoutAccountCommand(db *gorm.DB) consumer.HandlerFunc[logoutCommand] {
	return func(l logrus.FieldLogger, span opentracing.Span, command logoutCommand) {
		l.Debugf("Received logout account command account [%d].", command.AccountId)
		err := account.SetLoggedOut(db)(command.Tenant, command.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Error processing command to logout account [%d].", command.AccountId)
			return
		}
		emitLoggedOutEvent(l, span, command.Tenant)
	}
}
