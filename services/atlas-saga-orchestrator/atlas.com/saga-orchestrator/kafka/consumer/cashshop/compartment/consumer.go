package compartment

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	cashshopCompartment "atlas-saga-orchestrator/kafka/message/cashshop/compartment"
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
			rf(consumer2.NewConfig(l)("cashshop_compartment_status_event")(cashshopCompartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(cashshopCompartment.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleReleasedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorEvent)))
	}
}

func handleAcceptedEvent(l logrus.FieldLogger, ctx context.Context, e cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventAcceptedBody]) {
	if e.Type != cashshopCompartment.StatusEventTypeAccepted {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":   e.Body.TransactionId.String(),
		"compartment_id":   e.CompartmentId.String(),
		"compartment_type": e.CompartmentType,
	}).Debug("Cash shop accepted asset successfully")

	// Mark the saga step as completed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}

func handleReleasedEvent(l logrus.FieldLogger, ctx context.Context, e cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventReleasedBody]) {
	if e.Type != cashshopCompartment.StatusEventTypeReleased {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":   e.Body.TransactionId.String(),
		"compartment_id":   e.CompartmentId.String(),
		"compartment_type": e.CompartmentType,
	}).Debug("Cash shop released asset successfully")

	// Mark the saga step as completed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, true)
}

func handleErrorEvent(l logrus.FieldLogger, ctx context.Context, e cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventErrorBody]) {
	if e.Type != cashshopCompartment.StatusEventTypeError {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id":   e.Body.TransactionId.String(),
		"compartment_id":   e.CompartmentId.String(),
		"compartment_type": e.CompartmentType,
		"error_code":       e.Body.ErrorCode,
	}).Error("Cash shop compartment operation failed")

	// Mark the saga step as failed
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.Body.TransactionId, false)
}
