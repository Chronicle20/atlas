package account

import (
	"atlas-ban/history"
	consumer2 "atlas-ban/kafka/consumer"
	account2 "atlas-ban/kafka/message/account"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("account_session_status_event")(account2.EnvEventSessionStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(account2.EnvEventSessionStatusTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedSessionEvent(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorSessionEvent(db))))
		}
	}
}

func handleCreatedSessionEvent(db *gorm.DB) message.Handler[account2.SessionStatusEvent[account2.CreatedSessionStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e account2.SessionStatusEvent[account2.CreatedSessionStatusEventBody]) {
		if e.Type != account2.SessionEventStatusTypeCreated {
			return
		}
		l.Debugf("Recording successful login for account [%d] ip [%s] hwid [%s].", e.AccountId, e.Body.IPAddress, e.Body.HWID)
		_, err := history.NewProcessor(l, ctx, db).Record(e.AccountId, e.AccountName, e.Body.IPAddress, e.Body.HWID, true, "")
		if err != nil {
			l.WithError(err).Errorf("Unable to record login history for account [%d].", e.AccountId)
		}
	}
}

func handleErrorSessionEvent(db *gorm.DB) message.Handler[account2.SessionStatusEvent[account2.ErrorSessionStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e account2.SessionStatusEvent[account2.ErrorSessionStatusEventBody]) {
		if e.Type != account2.SessionEventStatusTypeError {
			return
		}
		l.Debugf("Recording failed login for account [%d] ip [%s] hwid [%s] code [%s].", e.AccountId, e.Body.IPAddress, e.Body.HWID, e.Body.Code)
		_, err := history.NewProcessor(l, ctx, db).Record(e.AccountId, e.AccountName, e.Body.IPAddress, e.Body.HWID, false, e.Body.Code)
		if err != nil {
			l.WithError(err).Errorf("Unable to record login history for account [%d].", e.AccountId)
		}
	}
}
