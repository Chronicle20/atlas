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
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("account_session_command")(EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(EnvCommandTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateAccountSessionCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleProgressStateAccountSessionCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLogoutAccountSessionCommand(db))))
		}
	}
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

func handleProgressStateAccountSessionCommand(db *gorm.DB) message.Handler[command[progressStateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[progressStateCommandBody]) {
		if c.Type != CommandTypeProgressState {
			return
		}
		_ = session.ProgressState(l)(ctx)(db)(c.SessionId, c.Issuer, c.AccountId, account.State(c.Body.State), c.Body.Params)
	}
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
