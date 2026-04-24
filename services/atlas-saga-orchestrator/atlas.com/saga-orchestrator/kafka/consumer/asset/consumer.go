package asset

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	notice "atlas-saga-orchestrator/kafka/message/conversation_reward_notice"
	"atlas-saga-orchestrator/saga"
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// emitRewardNoticeForCurrentStep inspects the saga's current pending step (the
// one that's about to be marked Completed by the caller) and, if it's a
// conversation-sourced reward step with ShowEffect=true, emits a
// conversation_reward_notice for atlas-channel to render. Failures here are
// non-fatal — the notice is purely cosmetic. Call BEFORE StepCompleted so the
// pending step is still observable on the saga.
//
// templateId and quantity come from the asset event body so the notice reflects
// what atlas-compartment actually applied, not what the payload requested (see
// PRD §4.4). DestroyAssetFromSlot is the one exception — it uses the payload's
// Quantity because slot-based deletes emit zero/unreliable quantity on the event.
func emitRewardNoticeForCurrentStep(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, templateId uint32, quantity uint32) {
	s, err := saga.NewProcessor(l, ctx).GetById(transactionId)
	if err != nil {
		return
	}
	step, ok := s.GetCurrentStep()
	if !ok {
		return
	}
	switch step.Action() {
	case saga.AwardAsset:
		payload, ok := step.Payload().(saga.AwardItemActionPayload)
		if !ok || !payload.ShowEffect {
			return
		}
		if err := saga.EmitConversationRewardNotice(l, ctx, payload.CharacterId, notice.KindItemGain, templateId, quantity); err != nil {
			l.WithError(err).Debug("Unable to emit conversation_reward_notice for item gain.")
		}
	case saga.DestroyAsset:
		payload, ok := step.Payload().(saga.DestroyAssetPayload)
		if !ok || !payload.ShowEffect {
			return
		}
		if err := saga.EmitConversationRewardNotice(l, ctx, payload.CharacterId, notice.KindItemLoss, templateId, quantity); err != nil {
			l.WithError(err).Debug("Unable to emit conversation_reward_notice for item loss.")
		}
	case saga.DestroyAssetFromSlot:
		payload, ok := step.Payload().(saga.DestroyAssetFromSlotPayload)
		if !ok || !payload.ShowEffect {
			return
		}
		// Slot-based destroy doesn't carry the templateId on the payload — use
		// the asset event's templateId. Quantity comes from the payload because
		// slot-delete events don't reliably report it.
		if err := saga.EmitConversationRewardNotice(l, ctx, payload.CharacterId, notice.KindItemLoss, templateId, payload.Quantity); err != nil {
			l.WithError(err).Debug("Unable to emit conversation_reward_notice for item loss.")
		}
	}
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status_event")(asset2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(asset2.EnvEventTopicStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetCreatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeletedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetQuantityUpdatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMovedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleAssetCreatedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody]) {
	if e.Type != asset2.StatusEventTypeCreated {
		return
	}

	sagaProcessor := saga.NewProcessor(l, ctx)

	assetResult := map[string]any{"assetId": e.AssetId}

	emitRewardNoticeForCurrentStep(l, ctx, e.TransactionId, e.TemplateId, e.Body.Quantity)

	// Get the saga to check if this is a CreateAndEquipAsset step
	s, err := sagaProcessor.GetById(e.TransactionId)
	if err != nil {
		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   e.CharacterId,
		}).Debug("Unable to locate saga for asset created event.")
		_ = sagaProcessor.StepCompletedWithResult(e.TransactionId, true, assetResult)
		return
	}

	// Get the current step to check if it's a CreateAndEquipAsset action
	currentStep, ok := s.GetCurrentStep()
	if !ok {
		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   e.CharacterId,
		}).Debug("No current step found for asset created event.")
		_ = sagaProcessor.StepCompleted(e.TransactionId, true)
		return
	}

	// Check if this is a CreateAndEquipAsset step
	if currentStep.Action() == saga.CreateAndEquipAsset {
		// Extract the payload to get the character ID and inventory type
		createPayload, ok := currentStep.Payload().(saga.CreateAndEquipAssetPayload)
		if !ok {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"character_id":   e.CharacterId,
				"step_id":        currentStep.StepId(),
			}).Error("Invalid payload for CreateAndEquipAsset step - expected CreateAndEquipAssetPayload.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}

		// Validate that the created character matches the expected character
		if createPayload.CharacterId != e.CharacterId {
			l.WithFields(logrus.Fields{
				"transaction_id":        e.TransactionId.String(),
				"expected_character_id": createPayload.CharacterId,
				"actual_character_id":   e.CharacterId,
				"step_id":               currentStep.StepId(),
			}).Error("Character ID mismatch in CreateAndEquipAsset creation event.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}

		// Generate a unique step ID for the auto-equip step with proper timestamp format
		// Format: auto_equip_step_<timestamp> where timestamp is Unix nanoseconds
		autoEquipStepId := fmt.Sprintf("auto_equip_step_%d", time.Now().UnixNano())

		// Create the EquipAsset step
		// Note: Using reasonable defaults for slot information since asset event doesn't provide it
		// The item is typically created in the first available slot (assumption: slot 5)
		// Equipment slot -1 is typically used for equipment
		it, _ := inventory.TypeFromItemId(item.Id(e.TemplateId))
		equipPayload := saga.EquipAssetPayload{
			CharacterId:   createPayload.CharacterId,
			InventoryType: uint32(it),
			Source:        e.Slot,
			Destination:   -1, // Assumption: equip to slot -1
		}

		equipStep := saga.NewStep[any](autoEquipStepId, saga.Pending, saga.EquipAsset, equipPayload)

		// Add the equip step to the saga after the current step (should be executed next)
		err = sagaProcessor.AddStepAfterCurrent(e.TransactionId, equipStep)
		if err != nil {
			l.WithFields(logrus.Fields{
				"transaction_id":     e.TransactionId.String(),
				"character_id":       e.CharacterId,
				"auto_equip_step_id": autoEquipStepId,
				"step_id":            currentStep.StepId(),
				"error":              err.Error(),
			}).Error("Failed to add the equip step to saga for CreateAndEquipAsset - marking saga step as failed.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id":     e.TransactionId.String(),
			"character_id":       e.CharacterId,
			"auto_equip_step_id": autoEquipStepId,
			"inventory_type":     equipPayload.InventoryType,
			"source_slot":        equipPayload.Source,
			"destination_slot":   equipPayload.Destination,
			"original_step_id":   currentStep.StepId(),
		}).Info("Successfully added auto-equip step for CreateAndEquipAsset action to be executed next.")
	}

	// Complete the current step (either regular creation or CreateAndEquipAsset)
	_ = sagaProcessor.StepCompletedWithResult(e.TransactionId, true, assetResult)
}

func handleAssetDeletedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.DeletedStatusEventBody]) {
	if e.Type != asset2.StatusEventTypeDeleted {
		return
	}
	// DeletedStatusEventBody carries no quantity — pass 0. DestroyAssetFromSlot
	// steps use their payload Quantity internally; DestroyAsset is driven by
	// payload semantics (RemoveAll / Quantity) and the full-delete case is the
	// common path here.
	emitRewardNoticeForCurrentStep(l, ctx, e.TransactionId, e.TemplateId, 0)
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleAssetQuantityUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.QuantityChangedEventBody]) {
	if e.Type != asset2.StatusEventTypeQuantityChanged {
		return
	}
	emitRewardNoticeForCurrentStep(l, ctx, e.TransactionId, e.TemplateId, e.Body.Quantity)
	_ = saga.NewProcessor(l, ctx).StepCompletedWithResult(e.TransactionId, true, map[string]any{"assetId": e.AssetId})
}

func handleAssetMovedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) {
	if e.Type != asset2.StatusEventTypeMoved {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}
