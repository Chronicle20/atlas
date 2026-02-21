package saga

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/kafka/message/saga"
	"atlas-channel/server"
	"atlas-channel/session"
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
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCompletedEvent(sc)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleFailedEvent(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

// handleCompletedEvent handles saga completion events
func handleCompletedEvent(sc server.Model) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
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

		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"saga_type":      e.Body.SagaType,
			"error_code":     e.Body.ErrorCode,
			"character_id":   e.Body.CharacterId,
			"failed_step":    e.Body.FailedStep,
		}).Debugf("Saga transaction failed. Reason: [%s]", e.Body.Reason)

		// Look up the session for the character
		s, ok := session.NewProcessor(l, ctx).GetSessionByCharacterId(e.Body.CharacterId)
		if !ok {
			l.WithField("character_id", e.Body.CharacterId).Debug("Character not connected, skipping error notification.")
			return
		}

		if s.ChannelId() != sc.ChannelId() {
			return
		}

		// Handle storage operation failures by sending appropriate error packets
		if e.Body.SagaType == saga.SagaTypeStorageOperation {
			// Get the appropriate error body producer based on the error code
			errorBody := getStorageErrorBodyProducer(l, e.Body.ErrorCode)
			if errorBody == nil {
				l.WithField("error_code", e.Body.ErrorCode).Warn("No error body producer for error code, skipping notification.")
				return
			}

			// Send the error packet to the client
			err := session.Announce(l)(ctx)(wp)(writer.StorageOperation)(errorBody)(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send storage error packet to client.")
				return
			}

			l.WithFields(logrus.Fields{
				"character_id": e.Body.CharacterId,
				"error_code":   e.Body.ErrorCode,
			}).Debug("Sent storage operation error packet to client.")
		}
	}
}

// getStorageErrorBodyProducer returns the appropriate BodyProducer for the given error code
func getStorageErrorBodyProducer(l logrus.FieldLogger, errorCode string) writer.BodyProducer {
	switch errorCode {
	case saga.ErrorCodeNotEnoughMesos:
		return writer.StorageOperationErrorNotEnoughMesoBody(l)
	case saga.ErrorCodeInventoryFull, saga.ErrorCodeStorageFull:
		return writer.StorageOperationErrorInventoryFullBody(l)
	default:
		return nil
	}
}
