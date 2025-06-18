package account

import (
	"atlas-cashshop/cashshop/inventory"
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

		// Create wallet
		_, err := wallet.NewProcessor(l, ctx, db).CreateAndEmit(e.AccountId, 0, 0, 0)
		if err != nil {
			l.WithError(err).Errorf("Could not create wallet for account [%d].", e.AccountId)
			return
		}

		// Create inventory with default compartments
		// Default capacity is 55, but this can be changed as needed
		_, err = inventory.NewProcessor(l, ctx, db).CreateAndEmit(e.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Could not create inventory for account [%d].", e.AccountId)
			return
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[account.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e account.StatusEvent) {
		if e.Status != account.EventStatusDeleted {
			return
		}

		// Delete wallet
		err := wallet.NewProcessor(l, ctx, db).DeleteAndEmit(e.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Could not delete wallet for account [%d].", e.AccountId)
			return
		}

		// Delete inventory, compartments, and assets
		err = inventory.NewProcessor(l, ctx, db).DeleteAndEmit(e.AccountId)
		if err != nil {
			l.WithError(err).Errorf("Could not delete inventory for account [%d].", e.AccountId)
			return
		}
	}
}
