package storage

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	"atlas-saga-orchestrator/kafka/message/storage"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("storage_status_event")(storage.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(storage.EnvStatusEventTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMesosUpdatedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStorageErrorEvent)))
	}
}

func handleMesosUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e storage.StatusEvent[storage.MesosUpdatedEventBody]) {
	if e.Type != storage.StatusEventTypeMesosUpdate {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"account_id":     e.AccountId,
		"old_mesos":      e.Body.OldMesos,
		"new_mesos":      e.Body.NewMesos,
	}).Debug("Storage mesos updated successfully")

	// Mark the saga step as completed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleStorageErrorEvent(l logrus.FieldLogger, ctx context.Context, e storage.StatusEvent[storage.ErrorEventBody]) {
	if e.Type != storage.StatusEventTypeError {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"account_id":     e.AccountId,
		"error_code":     e.Body.ErrorCode,
		"error_message":  e.Body.Message,
	}).Error("Storage operation failed")

	// Mark the saga step as failed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, false)
}
