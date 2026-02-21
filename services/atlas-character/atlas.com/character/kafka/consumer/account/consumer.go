package account

import (
	"atlas-character/character"
	consumer2 "atlas-character/kafka/consumer"
	"atlas-character/kafka/message/account"
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

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(account.EnvEventTopicStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[account.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e account.StatusEvent) {
		if e.Status != account.EventStatusDeleted {
			return
		}

		l.Infof("Account [%d] was deleted. Deleting all characters...", e.AccountId)
		err := character.NewProcessor(l, ctx, db).DeleteByAccountIdAndEmit(e.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Could not delete characters for account [%d].", e.AccountId)
			return
		}
	}
}
