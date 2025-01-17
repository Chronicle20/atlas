package session

import (
	"atlas-account/account"
	"atlas-account/account/session"
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

func CreateAccountSessionCommandRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleCreateAccountSessionCommand(db)))
}

func handleCreateAccountSessionCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[createCommandBody]) {
		if c.Type != CommandTypeCreate {
			return
		}

		l.Debugf("Received create account command account [%d] from [%s].", c.AccountId, c.Issuer)
		_ = session.AttemptLogin(l)(ctx)(db)(c.SessionId, c.Body.AccountName, c.Body.Password)
	}
}

func ProgressStateAccountSessionCommandRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleProgressStateAccountSessionCommand(db)))
}

func handleProgressStateAccountSessionCommand(db *gorm.DB) message.Handler[command[progressStateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[progressStateCommandBody]) {
		if c.Type != CommandTypeProgressState {
			return
		}
		_ = session.ProgressState(l)(ctx)(db)(c.SessionId, c.Issuer, c.AccountId, account.State(c.Body.State), c.Body.Params)
	}
}

func LogoutAccountSessionCommandRegister(l logrus.FieldLogger, db *gorm.DB) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
	return t, message.AdaptHandler(message.PersistentConfig(handleLogoutAccountSessionCommand(db)))
}

func handleLogoutAccountSessionCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c command[logoutCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c command[logoutCommandBody]) {
		if c.Type != CommandTypeLogout {
			return
		}

		l.Debugf("Received logout account command account [%d] from [%s].", c.AccountId, c.Issuer)
		_ = account.Logout(l, db, ctx)(c.SessionId, c.AccountId, strings.ToUpper(c.Issuer))
	}
}
