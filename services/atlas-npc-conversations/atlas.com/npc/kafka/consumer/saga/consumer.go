package saga

import (
	"atlas-npc-conversations/conversation"
	consumer2 "atlas-npc-conversations/kafka/consumer"
	"atlas-npc-conversations/kafka/message/saga"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
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

		// Get tenant from context
		t := tenant.MustFromContext(ctx)

		// Find the conversation context by saga ID
		conversationCtx, err := conversation.GetRegistry().GetContextBySagaId(t, e.TransactionId)
		if err != nil {
			l.WithError(err).WithField("transaction_id", e.TransactionId.String()).Warn("No conversation found for completed saga")
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":  conversationCtx.CharacterId(),
			"npc_id":        conversationCtx.NpcId(),
			"current_state": conversationCtx.CurrentState(),
		}).Info("Resuming conversation after saga completion")

		// Get the success state from context
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

		// Clean up temporary context values
		delete(conversationCtx.Context(), "craftAction_successState")
		delete(conversationCtx.Context(), "craftAction_failureState")
		delete(conversationCtx.Context(), "craftAction_missingMaterialsState")

		// Update the context in registry
		conversation.GetRegistry().UpdateContext(t, conversationCtx.CharacterId(), conversationCtx)

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
			"reason":         e.Body.Reason,
			"failed_step":    e.Body.FailedStep,
		}).Debug("Received saga failure event")

		// Get tenant from context
		t := tenant.MustFromContext(ctx)

		// Find the conversation context by saga ID
		conversationCtx, err := conversation.GetRegistry().GetContextBySagaId(t, e.TransactionId)
		if err != nil {
			l.WithError(err).WithField("transaction_id", e.TransactionId.String()).Warn("No conversation found for failed saga")
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":  conversationCtx.CharacterId(),
			"npc_id":        conversationCtx.NpcId(),
			"current_state": conversationCtx.CurrentState(),
			"reason":        e.Body.Reason,
		}).Info("Resuming conversation after saga failure")

		// Determine which failure state to use based on the failure reason
		var failureState string
		if e.Body.Reason == "Validation failed" {
			// Use missingMaterialsState for validation failures
			if state, exists := conversationCtx.Context()["craftAction_missingMaterialsState"]; exists && state != "" {
				failureState = state
			}
		}

		// Fall back to general failureState if missingMaterialsState not set or not a validation failure
		if failureState == "" {
			if state, exists := conversationCtx.Context()["craftAction_failureState"]; exists && state != "" {
				failureState = state
			}
		}

		if failureState == "" {
			l.WithField("character_id", conversationCtx.CharacterId()).Error("No failure state stored in conversation context")
			// End the conversation as we can't proceed
			_ = conversation.NewProcessor(l, ctx, db).End(conversationCtx.CharacterId())
			return
		}

		// Clear the pending saga and update to failure state
		conversationCtx = conversationCtx.ClearPendingSaga()
		conversationCtx = conversationCtx.SetCurrentState(failureState)

		// Clean up temporary context values
		delete(conversationCtx.Context(), "craftAction_successState")
		delete(conversationCtx.Context(), "craftAction_failureState")
		delete(conversationCtx.Context(), "craftAction_missingMaterialsState")

		// Update the context in registry
		conversation.GetRegistry().UpdateContext(t, conversationCtx.CharacterId(), conversationCtx)

		// Process the failure state using ProcessorImpl directly
		processor := conversation.NewProcessor(l, ctx, db).(*conversation.ProcessorImpl)
		_, err = processor.ProcessState(conversationCtx)
		if err != nil {
			l.WithError(err).Error("Failed to process failure state after saga failure")
			_ = processor.End(conversationCtx.CharacterId())
		}
	}
}
