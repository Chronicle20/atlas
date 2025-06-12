package account

import (
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/account"
	"atlas-cashshop/wallet"
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
			rf(consumer2.NewConfig(l)("account_status_event")(account.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(account.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db))))
		}
	}
}

func handleStatusEventCreated(db *gorm.DB) message.Handler[account.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e account.StatusEvent) {
		if e.Status != account.EventStatusCreated {
			return
		}
		l.Debugf("Account [%d] was created. Initializing cash shop information...", e.AccountId)
		_, _ = wallet.NewProcessor(l, ctx, db).CreateAndEmit(e.AccountId, 0, 0, 0)
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[account.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e account.StatusEvent) {
		if e.Status != account.EventStatusDeleted {
			return
		}
		_ = wallet.NewProcessor(l, ctx, db).DeleteAndEmit(e.AccountId)
	}
}
