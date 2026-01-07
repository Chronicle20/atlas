package cashshop

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	cashshop2 "atlas-saga-orchestrator/kafka/message/cashshop"
	"atlas-saga-orchestrator/saga"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("wallet_status_event")(cashshop2.EnvEventTopicWalletStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(cashshop2.EnvEventTopicWalletStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleWalletUpdatedEvent)))
	}
}

func handleWalletUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e cashshop2.StatusEvent[cashshop2.StatusEventUpdatedBody]) {
	if e.Type != cashshop2.StatusEventTypeUpdated {
		return
	}

	// Skip events without a transaction ID (non-saga wallet updates)
	if e.Body.TransactionId == uuid.Nil {
		l.Debugf("Wallet updated event for account [%d] has no transaction ID, skipping saga completion", e.AccountId)
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"account_id":     e.AccountId,
		"credit":         e.Body.Credit,
		"points":         e.Body.Points,
		"prepaid":        e.Body.Prepaid,
	}).Debug("Wallet updated successfully, marking saga step as completed")

	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}
