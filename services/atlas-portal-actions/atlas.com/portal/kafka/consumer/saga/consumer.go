package saga

import (
	"context"
	"fmt"

	"atlas-portal-actions/action"
	"atlas-portal-actions/character"
	consumer2 "atlas-portal-actions/kafka/consumer"
	"atlas-portal-actions/kafka/message/saga"
	portalsaga "atlas-portal-actions/saga"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	scriptsaga "github.com/Chronicle20/atlas-script-core/saga"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// InitConsumers initializes Kafka consumers for saga status events
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}

// InitHandlers initializes Kafka message handlers for saga status events
func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		t, _ := topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCompleted(l))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFailed(l))))
	}
}

// handleStatusEventCompleted handles saga completion events
// For transport sagas, the warp already happened - just cleanup
func handleStatusEventCompleted(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}

		t := tenant.MustFromContext(ctx)

		// Try to find and remove pending action
		pendingAction, found := action.GetRegistry().Get(t.Id(), e.TransactionId)
		if !found {
			// Not a portal action saga, ignore
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   pendingAction.CharacterId,
		}).Debug("Transport saga completed, cleaning up pending action")

		// Cleanup - warp already happened via saga orchestrator
		action.GetRegistry().Remove(t.Id(), e.TransactionId)
	}
}

// handleStatusEventFailed handles saga failure events
// Sends failure message to character and enables their actions
func handleStatusEventFailed(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventFailedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
		if e.Type != saga.StatusEventTypeFailed {
			return
		}

		t := tenant.MustFromContext(ctx)

		// Try to find pending action
		pendingAction, found := action.GetRegistry().Get(t.Id(), e.TransactionId)
		if !found {
			// Not a portal action saga, ignore
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   pendingAction.CharacterId,
			"error_code":     e.Body.ErrorCode,
			"reason":         e.Body.Reason,
			"failed_step":    e.Body.FailedStep,
		}).Info("Transport saga failed, sending failure message to character")

		// Determine the message to send
		failureMessage := resolveFailureMessage(pendingAction, e.Body.ErrorCode)

		// Send failure message via saga
		if failureMessage != "" {
			ch := channel.NewModel(pendingAction.WorldId, pendingAction.ChannelId)
			sendFailureMessage(l, ctx, pendingAction.CharacterId, ch, failureMessage)
		}

		// Enable character actions
		ch := channel.NewModel(pendingAction.WorldId, pendingAction.ChannelId)
		character.EnableActions(l)(ctx)(ch, pendingAction.CharacterId)

		// Cleanup
		action.GetRegistry().Remove(t.Id(), e.TransactionId)
	}
}

// resolveFailureMessage determines the appropriate failure message based on error code
func resolveFailureMessage(pendingAction action.PendingAction, errorCode string) string {
	// Use custom failure message if provided
	if pendingAction.FailureMessage != "" {
		return pendingAction.FailureMessage
	}

	// Default messages based on error code
	switch errorCode {
	case "TRANSPORT_CAPACITY_FULL":
		return "The transport is currently full. Please try again later."
	case "TRANSPORT_ALREADY_IN_TRANSIT":
		return "You are already on a transport."
	case "TRANSPORT_ROUTE_NOT_FOUND":
		return "Transport service is currently unavailable."
	case "TRANSPORT_SERVICE_ERROR":
		return "Transport service is currently unavailable."
	default:
		return "Unable to board transport at this time."
	}
}

// sendFailureMessage creates a saga to send a message to the character
func sendFailureMessage(l logrus.FieldLogger, ctx context.Context, characterId uint32, ch channel.Model, message string) {
	s := scriptsaga.NewBuilder().
		SetSagaType(scriptsaga.InventoryTransaction).
		SetInitiatedBy("portal-action-transport-failure").
		AddStep(
			fmt.Sprintf("message-%d", characterId),
			scriptsaga.Pending,
			scriptsaga.SendMessage,
			scriptsaga.SendMessagePayload{
				CharacterId: characterId,
				WorldId:     ch.WorldId(),
				ChannelId:   ch.Id(),
				MessageType: "PINK_TEXT",
				Message:     message,
			},
		).Build()

	err := portalsaga.NewProcessor(l, ctx).Create(s)
	if err != nil {
		l.WithError(err).Errorf("Failed to send failure message to character [%d]", characterId)
	}
}
