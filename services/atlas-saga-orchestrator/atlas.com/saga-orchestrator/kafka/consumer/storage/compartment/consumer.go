package compartment

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	storageCompartment "atlas-saga-orchestrator/kafka/message/storage/compartment"
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
			rf(consumer2.NewConfig(l)("storage_compartment_status_event")(storageCompartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(storageCompartment.EnvEventTopicStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleAcceptedEvent(l logrus.FieldLogger, ctx context.Context, e storageCompartment.StatusEvent[storageCompartment.StatusEventAcceptedBody]) {
	if e.Type != storageCompartment.StatusEventTypeAccepted {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"account_id":     e.AccountId,
		"asset_id":       e.Body.AssetId,
		"slot":           e.Body.Slot,
	}).Debug("Storage accepted asset successfully")

	// Mark the saga step as completed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}

func handleReleasedEvent(l logrus.FieldLogger, ctx context.Context, e storageCompartment.StatusEvent[storageCompartment.StatusEventReleasedBody]) {
	if e.Type != storageCompartment.StatusEventTypeReleased {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"account_id":     e.AccountId,
		"asset_id":       e.Body.AssetId,
	}).Debug("Storage released asset successfully")

	// Mark the saga step as completed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}

func handleErrorEvent(l logrus.FieldLogger, ctx context.Context, e storageCompartment.StatusEvent[storageCompartment.StatusEventErrorBody]) {
	if e.Type != storageCompartment.StatusEventTypeError {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"account_id":     e.AccountId,
		"error_code":     e.Body.ErrorCode,
		"error_message":  e.Body.Message,
	}).Error("Storage compartment operation failed")

	// Mark the saga step as failed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, false)
}
