package transfer

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	"atlas-saga-orchestrator/kafka/message/transfer"
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
			rf(consumer2.NewConfig(l)("compartment_transfer_status_event")(transfer.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(transfer.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleTransferCompletedEvent)))
	}
}

func handleTransferCompletedEvent(l logrus.FieldLogger, ctx context.Context, e transfer.StatusEvent[transfer.StatusEventCompletedBody]) {
	if e.Type != transfer.StatusEventTypeCompleted {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"character_id":   e.CharacterId,
		"asset_id":       e.Body.AssetId,
		"inventory_type": e.Body.InventoryType,
	}).Debug("Compartment transfer completed successfully")

	// Mark the saga step as completed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}
