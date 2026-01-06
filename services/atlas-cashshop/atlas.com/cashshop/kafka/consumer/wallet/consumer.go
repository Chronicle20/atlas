package wallet

import (
	consumer2 "atlas-cashshop/kafka/consumer"
	"atlas-cashshop/kafka/message/wallet"
	wallet2 "atlas-cashshop/wallet"
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
			rf(consumer2.NewConfig(l)("wallet_command")(wallet.EnvCommandTopicWallet)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(wallet.EnvCommandTopicWallet)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAdjustCurrencyCommand(db))))
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

		_, err := wallet2.NewProcessor(l, ctx, db).AdjustCurrency(c.AccountId, c.CurrencyType, c.Amount)
		if err != nil {
			l.WithError(err).Errorf("Could not adjust currency for account [%d].", c.AccountId)
			return
		}

		l.Debugf("Successfully adjusted currency for account [%d]. Transaction: %s", c.AccountId, c.TransactionId.String())
	}
}
