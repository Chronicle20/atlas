package buddylist

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	buddylist2 "atlas-saga-orchestrator/kafka/message/buddylist"
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
			rf(consumer2.NewConfig(l)("buddylist_status_event")(buddylist2.EnvEventTopicBuddyListStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(buddylist2.EnvEventTopicBuddyListStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleBuddyCapacityChangedEvent)))
	}
}

func handleBuddyCapacityChangedEvent(l logrus.FieldLogger, ctx context.Context, e buddylist2.StatusEvent[buddylist2.BuddyCapacityChangeStatusEventBody]) {
	if e.Type != buddylist2.StatusEventTypeBuddyCapacityUpdate {
		return
	}

	// Skip events without a transaction ID (non-saga capacity changes)
	if e.Body.TransactionId == uuid.Nil {
		l.Debugf("Buddy capacity changed event for character [%d] has no transaction ID, skipping saga completion", e.CharacterId)
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"character_id":   e.CharacterId,
		"new_capacity":   e.Body.Capacity,
	}).Debug("Buddy capacity changed successfully, marking saga step as completed")

	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}
