package account

import (
	"atlas-account/account"
	consumer2 "atlas-account/kafka/consumer"
	account2 "atlas-account/kafka/message/account"
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
			rf(consumer2.NewConfig(l)("create_account_command")(account2.EnvCommandTopicCreateAccount)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
			rf(consumer2.NewConfig(l)("account_session_command")(account2.EnvCommandSessionTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(account2.EnvCommandTopicCreateAccount)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateAccountCommand(db))))
			t, _ = topic.EnvProvider(l)(account2.EnvCommandSessionTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateAccountSessionCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleProgressStateAccountSessionCommand(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLogoutAccountSessionCommand(db))))
		}
	}
}

func handleCreateAccountCommand(db *gorm.DB) message.Handler[account2.CreateCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c account2.CreateCommand) {
		l.Debugf("Received create account command name [%s] password [%s].", c.Name, c.Password)
		_, err := account.NewProcessor(l, ctx, db).CreateAndEmit(c.Name, c.Password)
		if err != nil {
			l.WithError(err).Errorf("Error processing command to create account [%s].", c.Name)
			return
		}
	}
}

func handleCreateAccountSessionCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c account2.SessionCommand[account2.CreateSessionCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c account2.SessionCommand[account2.CreateSessionCommandBody]) {
		if c.Type != account2.SessionCommandTypeCreate {
			return
		}

		l.Debugf("Received create account command account [%d] from [%s].", c.AccountId, c.Issuer)
		_ = account.NewProcessor(l, ctx, db).AttemptLoginAndEmit(c.SessionId, c.Body.AccountName, c.Body.Password)
	}
}

func handleProgressStateAccountSessionCommand(db *gorm.DB) message.Handler[account2.SessionCommand[account2.ProgressStateSessionCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c account2.SessionCommand[account2.ProgressStateSessionCommandBody]) {
		if c.Type != account2.SessionCommandTypeProgressState {
			return
		}
		_ = account.NewProcessor(l, ctx, db).ProgressStateAndEmit(c.SessionId, c.Issuer, c.AccountId, account.State(c.Body.State), c.Body.Params)
	}
}

func handleLogoutAccountSessionCommand(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, c account2.SessionCommand[account2.LogoutSessionCommandBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, c account2.SessionCommand[account2.LogoutSessionCommandBody]) {
		if c.Type != account2.SessionCommandTypeLogout {
			return
		}

		l.Debugf("Received logout account command account [%d] from [%s].", c.AccountId, c.Issuer)
		_ = account.NewProcessor(l, ctx, db).LogoutAndEmit(c.SessionId, c.AccountId, strings.ToUpper(c.Issuer))
	}
}
