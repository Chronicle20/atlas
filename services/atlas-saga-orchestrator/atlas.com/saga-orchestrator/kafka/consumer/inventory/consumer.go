package inventory

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	inventory2 "atlas-saga-orchestrator/kafka/message/inventory"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("inventory_status_event")(inventory2.EnvEventTopicInventoryStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(inventory2.EnvEventTopicInventoryStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryCreatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryCreationFailedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleInventoryCreatedEvent(l logrus.FieldLogger, ctx context.Context, e inventory2.StatusEvent[inventory2.CreatedStatusEventBody]) {
	if e.Type != inventory2.StatusEventTypeCreated {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindInventoryCreated); !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
	}).Debug("Inventory created, advancing AwaitInventoryCreated step.")
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleInventoryCreationFailedEvent(l logrus.FieldLogger, ctx context.Context, e inventory2.StatusEvent[inventory2.CreationFailedStatusEventBody]) {
	if e.Type != inventory2.StatusEventTypeCreationFailed {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindInventoryCreationFailed); !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
		"reason":         e.Body.Reason,
	}).Error("Inventory creation failed, failing AwaitInventoryCreated step.")
	_ = p.StepCompleted(e.TransactionId, false)
}
