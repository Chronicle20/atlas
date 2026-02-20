package account

import (
	consumer2 "atlas-storage/kafka/consumer"
	"atlas-storage/kafka/message/account"
	"atlas-storage/storage"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	kafkaMessage "github.com/Chronicle20/atlas-kafka/message"
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
			if _, err := rf(t, kafkaMessage.AdaptHandler(kafkaMessage.PersistentConfig(handleStatusEventDeleted(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) kafkaMessage.Handler[account.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e account.StatusEvent) {
		if e.Status != account.EventStatusDeleted {
			return
		}

		l.Infof("Account [%d] was deleted. Deleting all storage records...", e.AccountId)
		err := storage.NewProcessor(l, ctx, db).DeleteByAccountId(e.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Could not delete storage for account [%d].", e.AccountId)
			return
		}
	}
}
