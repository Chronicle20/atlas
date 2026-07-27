package wallet

import (
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/wallet"
	wallet2 "atlas-cashshop/wallet"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("wallet_command")(wallet.EnvCommandTopicWallet)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(wallet.EnvCommandTopicWallet)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAdjustCurrencyCommand(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleAdjustCurrencyCommand(db *gorm.DB) message.Handler[wallet.AdjustCurrencyCommand] {
	return func(l logrus.FieldLogger, ctx context.Context, c wallet.AdjustCurrencyCommand) {
		if c.Type != wallet.CommandTypeAdjustCurrency {
			return
		}
		l.Debugf("Received adjust currency command for account [%d]. Currency type: %d, Amount: %d, Transaction: %s",
			c.AccountId, c.CurrencyType, c.Amount, c.TransactionId.String())

		proc := wallet2.NewProcessor(l, ctx, db)
		_, err := proc.AdjustCurrencyWithTransaction(c.TransactionId, c.AccountId, c.CurrencyType, c.Amount)
		if err != nil {
			l.WithError(err).Errorf("Could not adjust currency for account [%d].", c.AccountId)
			// Fail the waiting saga step fast rather than letting it time out. Only a
			// transactional adjust has a saga waiter; a nil transaction id is a
			// non-saga adjust with nobody to notify.
			if c.TransactionId != uuid.Nil {
				if emitErr := proc.EmitAdjustFailure(c.TransactionId, c.AccountId, err.Error()); emitErr != nil {
					l.WithError(emitErr).Errorf("Could not emit wallet adjust-failure event for account [%d].", c.AccountId)
				}
			}
			return
		}

		l.Debugf("Successfully adjusted currency for account [%d]. Transaction: %s", c.AccountId, c.TransactionId.String())
	}
}
