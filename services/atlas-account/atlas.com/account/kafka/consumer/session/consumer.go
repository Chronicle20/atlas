package session

import (
	"atlas-account/account"
	consumer2 "atlas-account/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
)

const (
	consumerAccountSessionCommand = "account_session_command"
)

func AccountSessionCommandConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerAccountSessionCommand)(EnvCommandTopic)(groupId)
	}
}

func LogoutAccountSessionCommandRegister(l *logrus.Logger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleLogoutAccountSessionCommand(db)))
}

func handleLogoutAccountSessionCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[logoutCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[logoutCommandBody]) {
		l.Debugf("Received logout account command account [%d] from [%s].", c.AccountId, c.Issuer)
		_ = account.Logout(l, db, ctx)(c.SessionId, c.AccountId, strings.ToUpper(c.Issuer))
		// l.Debugf("Ignoring logout command from [%s] as account [%d] is not in correct state.", command.Issuer, command.AccountId)
	}
}
