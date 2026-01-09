package consumable

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	consumable2 "atlas-saga-orchestrator/kafka/message/consumable"
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
			rf(consumer2.NewConfig(l)("consumable_status_event")(consumable2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(consumable2.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEffectAppliedEvent)))
	}
}

func handleEffectAppliedEvent(l logrus.FieldLogger, ctx context.Context, e consumable2.StatusEvent[consumable2.EffectAppliedStatusEventBody]) {
	if e.Type != consumable2.StatusEventTypeEffectApplied {
		return
	}

	// Skip events without a transaction ID (non-saga effect applications)
	if e.Body.TransactionId == uuid.Nil {
		l.Debugf("Effect applied event for character [%d] has no transaction ID, skipping saga completion", e.CharacterId)
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"character_id":   e.CharacterId,
		"item_id":        e.Body.ItemId,
	}).Debug("Consumable effect applied successfully, marking saga step as completed")

	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}
