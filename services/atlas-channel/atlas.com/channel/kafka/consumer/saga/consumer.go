package saga

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/kafka/message/saga"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// InitConsumers initializes saga status event consumers
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

// InitHandlers initializes saga status event handlers
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompletedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleFailedEvent(sc, wp))))
			}
		}
	}
}

// handleCompletedEvent handles saga completion events
func handleCompletedEvent(sc server.Model, wp writer.Producer) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Debugf("Saga transaction [%s] completed successfully.", e.TransactionId.String())
		// Storage mesos update is handled by storage consumer
		// Character sees the result through character meso changed event
		// No additional action needed here
	}
}

// handleFailedEvent handles saga failure events
func handleFailedEvent(sc server.Model, wp writer.Producer) message.Handler[saga.StatusEvent[saga.StatusEventFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
		if e.Type != saga.StatusEventTypeFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Warnf("Saga transaction [%s] failed. Reason: [%s], Failed Step: [%s]", e.TransactionId.String(), e.Body.Reason, e.Body.FailedStep)

		// TODO: Notify the client about the failure
		// This might involve looking up the session associated with this transaction
		// and sending an error message to the client
	}
}
