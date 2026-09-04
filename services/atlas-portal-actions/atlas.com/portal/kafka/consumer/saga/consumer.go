package saga

import (
	"atlas-portal-actions/action"
	"atlas-portal-actions/character"
	"atlas-portal-actions/kafka/message/saga"
	"context"
	"fmt"

	consumer2 "atlas-portal-actions/kafka/consumer"

	portalsaga "atlas-portal-actions/saga"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// InitConsumers initializes Kafka consumers for saga status events
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser),
			)
		}
	}
}

// InitHandlers initializes Kafka message handlers for saga status events
func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, err := topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
		if err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCompleted(l)))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFailed(l)))); err != nil {
			return err
		}
		return nil
	}
}

// handleStatusEventCompleted handles saga completion events
// For portal action sagas (warp or transport), the move already happened - just cleanup
func handleStatusEventCompleted(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}

		// Try to find and remove pending action
		pendingAction, found := action.GetRegistry().Get(l, ctx, e.TransactionId)
		if !found {
			// Not a portal action saga, ignore
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   pendingAction.CharacterId,
		}).Debug("Portal action saga completed, cleaning up pending action")

		// Cleanup - warp already happened via saga orchestrator
		action.GetRegistry().Remove(l, ctx, e.TransactionId)
	}
}

// handleStatusEventFailed handles saga failure events
// Sends failure message to character and enables their actions
func handleStatusEventFailed(l logrus.FieldLogger) message.Handler[saga.StatusEvent[saga.StatusEventFailedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
		if e.Type != saga.StatusEventTypeFailed {
			return
		}

		// Try to find pending action
		pendingAction, found := action.GetRegistry().Get(l, ctx, e.TransactionId)
		if !found {
			// Not a portal action saga, ignore
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   pendingAction.CharacterId,
			"kind":           pendingAction.Kind,
			"error_code":     e.Body.ErrorCode,
			"reason":         e.Body.Reason,
			"failed_step":    e.Body.FailedStep,
		}).Info("Portal action saga failed, sending failure message to character")

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
		action.GetRegistry().Remove(l, ctx, e.TransactionId)
	}
}

// resolveFailureMessage determines the appropriate failure message based on
// error code, falling back to a default chosen by the kind of portal action
// that failed (task-184 FR-2.7).
func resolveFailureMessage(pendingAction action.PendingAction, errorCode string) string {
	// Use custom failure message if provided
	if pendingAction.FailureMessage != "" {
		return pendingAction.FailureMessage
	}

	// Default messages based on error code. These codes are emitted only on the
	// transport path; a warp saga never produces them.
	switch errorCode {
	case "TRANSPORT_CAPACITY_FULL":
		return "The transport is currently full. Please try again later."
	case "TRANSPORT_ALREADY_IN_TRANSIT":
		return "You are already on a transport."
	case "TRANSPORT_ROUTE_NOT_FOUND":
		return "Transport service is currently unavailable."
	case "TRANSPORT_SERVICE_ERROR":
		return "Transport service is currently unavailable."
	}

	// No specific code: pick by what actually failed. An empty Kind means the
	// entry was written by a replica predating the field, so it keeps the
	// pre-existing transport text.
	switch pendingAction.Kind {
	case action.KindWarp:
		return "You cannot move there right now."
	default:
		return "Unable to board transport at this time."
	}
}

// sendFailureMessage creates a saga to send a message to the character
func sendFailureMessage(l logrus.FieldLogger, ctx context.Context, characterId uint32, ch channel.Model, message string) {
	s := sharedsaga.NewBuilder().
		SetSagaType(sharedsaga.InventoryTransaction).
		SetInitiatedBy("portal-action-failure").
		AddStep(
			fmt.Sprintf("message-%d", characterId),
			sharedsaga.Pending,
			sharedsaga.SendMessage,
			sharedsaga.SendMessagePayload{ // script-ops-guard:allow — internal failure notice, not driven by script params; ops.SendMessage needs a script param map this call site doesn't have.
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
