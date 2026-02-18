package saga

import (
	"atlas-npc-conversations/conversation"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	"atlas-npc-conversations/kafka/message/saga"
	npcSender "atlas-npc-conversations/npc"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("saga_status_event")(saga.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(saga.EnvStatusEventTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCompleted(l, db))))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventFailed(l, db))))
	}
}

func handleStatusEventCompleted(l logrus.FieldLogger, db *gorm.DB) message.Handler[saga.StatusEvent[saga.StatusEventCompletedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventCompletedBody]) {
		if e.Type != saga.StatusEventTypeCompleted {
			return
		}

		l.WithField("transaction_id", e.TransactionId.String()).Debug("Received saga completion event")

		// Find the conversation context by saga ID
		conversationCtx, err := conversation.GetRegistry().GetContextBySagaId(ctx, e.TransactionId)
		if err != nil {
			l.WithError(err).WithField("transaction_id", e.TransactionId.String()).Warn("No conversation found for completed saga")
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":  conversationCtx.CharacterId(),
			"npc_id":        conversationCtx.NpcId(),
			"current_state": conversationCtx.CurrentState(),
		}).Info("Resuming conversation after saga completion")

		// Check if this is a transport action (no success state needed - player is warped)
		if _, isTransport := conversationCtx.Context()["transportAction_failureState"]; isTransport {
			l.WithField("character_id", conversationCtx.CharacterId()).Debug("Transport action completed - player was warped, ending conversation")
			// Clean up transport context values and end conversation
			delete(conversationCtx.Context(), "transportAction_failureState")
			delete(conversationCtx.Context(), "transportAction_capacityFullState")
			delete(conversationCtx.Context(), "transportAction_alreadyInTransitState")
			delete(conversationCtx.Context(), "transportAction_routeNotFoundState")
			delete(conversationCtx.Context(), "transportAction_serviceErrorState")
			conversationCtx = conversationCtx.ClearPendingSaga()
			conversation.GetRegistry().UpdateContext(ctx, conversationCtx.CharacterId(), conversationCtx)
			_ = conversation.NewProcessor(l, ctx, db).End(conversationCtx.CharacterId())
			return
		}

		// Check if this is a gachapon action (item awarded, end conversation)
		if _, isGachapon := conversationCtx.Context()["gachaponAction_failureState"]; isGachapon {
			l.WithField("character_id", conversationCtx.CharacterId()).Debug("Gachapon action completed - item awarded, ending conversation")
			delete(conversationCtx.Context(), "gachaponAction_failureState")
			conversationCtx = conversationCtx.ClearPendingSaga()
			conversation.GetRegistry().UpdateContext(ctx, conversationCtx.CharacterId(), conversationCtx)
			_ = conversation.NewProcessor(l, ctx, db).End(conversationCtx.CharacterId())
			npcSender.NewProcessor(l, ctx).Dispose(conversationCtx.Field().Channel(), conversationCtx.CharacterId())
			return
		}

		// Get the success state from context (craft actions)
		successState, exists := conversationCtx.Context()["craftAction_successState"]
		if !exists || successState == "" {
			l.WithField("character_id", conversationCtx.CharacterId()).Error("No success state stored in conversation context")
			// End the conversation as we can't proceed
			_ = conversation.NewProcessor(l, ctx, db).End(conversationCtx.CharacterId())
			return
		}

		// Clear the pending saga and update to success state
		conversationCtx = conversationCtx.ClearPendingSaga()
		conversationCtx = conversationCtx.SetCurrentState(successState)

		// Clean up temporary context values (craft action)
		delete(conversationCtx.Context(), "craftAction_successState")
		delete(conversationCtx.Context(), "craftAction_failureState")
		delete(conversationCtx.Context(), "craftAction_missingMaterialsState")

		// Update the context in registry
		conversation.GetRegistry().UpdateContext(ctx, conversationCtx.CharacterId(), conversationCtx)

		// Process the success state using ProcessorImpl directly
		processor := conversation.NewProcessor(l, ctx, db).(*conversation.ProcessorImpl)
		_, err = processor.ProcessState(conversationCtx)
		if err != nil {
			l.WithError(err).Error("Failed to process success state after saga completion")
			_ = processor.End(conversationCtx.CharacterId())
		}
	}
}

func handleStatusEventFailed(l logrus.FieldLogger, db *gorm.DB) message.Handler[saga.StatusEvent[saga.StatusEventFailedBody]] {
	return func(logger logrus.FieldLogger, ctx context.Context, e saga.StatusEvent[saga.StatusEventFailedBody]) {
		if e.Type != saga.StatusEventTypeFailed {
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"error_code":     e.Body.ErrorCode,
			"reason":         e.Body.Reason,
			"failed_step":    e.Body.FailedStep,
		}).Debug("Received saga failure event")

		// Find the conversation context by saga ID
		conversationCtx, err := conversation.GetRegistry().GetContextBySagaId(ctx, e.TransactionId)
		if err != nil {
			l.WithError(err).WithField("transaction_id", e.TransactionId.String()).Warn("No conversation found for failed saga")
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":  conversationCtx.CharacterId(),
			"npc_id":        conversationCtx.NpcId(),
			"current_state": conversationCtx.CurrentState(),
			"error_code":    e.Body.ErrorCode,
			"reason":        e.Body.Reason,
		}).Info("Resuming conversation after saga failure")

		// Determine which failure state to use based on error code and reason
		failureState := resolveFailureState(conversationCtx, e.Body.ErrorCode, e.Body.Reason)

		if failureState == "" {
			l.WithField("character_id", conversationCtx.CharacterId()).Error("No failure state stored in conversation context")
			// End the conversation as we can't proceed
			_ = conversation.NewProcessor(l, ctx, db).End(conversationCtx.CharacterId())
			return
		}

		// Clear the pending saga and update to failure state
		conversationCtx = conversationCtx.ClearPendingSaga()
		conversationCtx = conversationCtx.SetCurrentState(failureState)

		// Clean up temporary context values (craft action)
		delete(conversationCtx.Context(), "craftAction_successState")
		delete(conversationCtx.Context(), "craftAction_failureState")
		delete(conversationCtx.Context(), "craftAction_missingMaterialsState")

		// Clean up temporary context values (transport action)
		delete(conversationCtx.Context(), "transportAction_successState")
		delete(conversationCtx.Context(), "transportAction_failureState")
		delete(conversationCtx.Context(), "transportAction_capacityFullState")
		delete(conversationCtx.Context(), "transportAction_alreadyInTransitState")
		delete(conversationCtx.Context(), "transportAction_routeNotFoundState")
		delete(conversationCtx.Context(), "transportAction_serviceErrorState")

		// Clean up temporary context values (gachapon action)
		delete(conversationCtx.Context(), "gachaponAction_failureState")

		// Update the context in registry
		conversation.GetRegistry().UpdateContext(ctx, conversationCtx.CharacterId(), conversationCtx)

		// Process the failure state using ProcessorImpl directly
		processor := conversation.NewProcessor(l, ctx, db).(*conversation.ProcessorImpl)
		_, err = processor.ProcessState(conversationCtx)
		if err != nil {
			l.WithError(err).Error("Failed to process failure state after saga failure")
			_ = processor.End(conversationCtx.CharacterId())
		}
	}
}

// resolveFailureState determines which failure state to use based on error code and reason
func resolveFailureState(ctx conversation.ConversationContext, errorCode string, reason string) string {
	// Check for transport-specific error codes first
	switch errorCode {
	case "TRANSPORT_CAPACITY_FULL":
		if state, exists := ctx.Context()["transportAction_capacityFullState"]; exists && state != "" {
			return state
		}
	case "TRANSPORT_ALREADY_IN_TRANSIT":
		if state, exists := ctx.Context()["transportAction_alreadyInTransitState"]; exists && state != "" {
			return state
		}
	case "TRANSPORT_ROUTE_NOT_FOUND":
		if state, exists := ctx.Context()["transportAction_routeNotFoundState"]; exists && state != "" {
			return state
		}
	case "TRANSPORT_SERVICE_ERROR":
		if state, exists := ctx.Context()["transportAction_serviceErrorState"]; exists && state != "" {
			return state
		}
	}

	// Check for transport general failure state
	if errorCode != "" && len(errorCode) > 10 && errorCode[:10] == "TRANSPORT_" {
		if state, exists := ctx.Context()["transportAction_failureState"]; exists && state != "" {
			return state
		}
	}

	// Check for craft action validation failures
	if reason == "Validation failed" {
		if state, exists := ctx.Context()["craftAction_missingMaterialsState"]; exists && state != "" {
			return state
		}
	}

	// Fall back to general craft action failure state
	if state, exists := ctx.Context()["craftAction_failureState"]; exists && state != "" {
		return state
	}

	// Fall back to gachapon action failure state
	if state, exists := ctx.Context()["gachaponAction_failureState"]; exists && state != "" {
		return state
	}

	return ""
}
