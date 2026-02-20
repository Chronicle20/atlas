package pet

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	pet2 "atlas-saga-orchestrator/kafka/message/pet"
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
			rf(consumer2.NewConfig(l)("pet_status_event")(pet2.EnvEventTopicPetStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(pet2.EnvEventTopicPetStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleClosenessChangedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleClosenessChangedEvent(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.ClosenessChangedStatusEventBody]) {
	if e.Type != pet2.StatusEventTypeClosenessChanged {
		return
	}

	// Skip events without a transaction ID (non-saga closeness changes)
	if e.Body.TransactionId == uuid.Nil {
		l.Debugf("Pet closeness changed event for pet [%d] has no transaction ID, skipping saga completion", e.PetId)
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"pet_id":         e.PetId,
		"owner_id":       e.OwnerId,
		"new_closeness":  e.Body.Closeness,
		"amount":         e.Body.Amount,
	}).Debug("Pet closeness changed successfully, marking saga step as completed")

	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}
