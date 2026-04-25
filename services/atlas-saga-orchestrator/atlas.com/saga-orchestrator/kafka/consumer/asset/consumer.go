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

	decision, ok := sagaProcessor.AcceptEvent(e.TransactionId, saga.EventKindAssetCreated)
	if !ok {
		return
	}

	// Template-id guard for AwardAsset and CreateAndEquipAsset.
	switch decision.Step.Action() {
	case saga.AwardAsset:
		payload, cast := decision.Step.Payload().(saga.AwardItemActionPayload)
		if !cast {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"step_id":        decision.Step.StepId(),
			}).Error("Invalid payload for AwardAsset step.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}
		if payload.Item.TemplateId != e.TemplateId {
			saga.LogSkip(l, logrus.Fields{
				"transaction_id":       e.TransactionId.String(),
				"step_id":              decision.Step.StepId(),
				"step_action":          decision.Step.Action(),
				"event_kind":           saga.EventKindAssetCreated,
				"event_template_id":    e.TemplateId,
				"expected_template_id": payload.Item.TemplateId,
			}, saga.SkipReasonTemplateIdMismatch)
			return
		}
	case saga.CreateAndEquipAsset:
		payload, cast := decision.Step.Payload().(saga.CreateAndEquipAssetPayload)
		if !cast {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"character_id":   e.CharacterId,
				"step_id":        decision.Step.StepId(),
			}).Error("Invalid payload for CreateAndEquipAsset step - expected CreateAndEquipAssetPayload.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}
		if payload.CharacterId != e.CharacterId {
			l.WithFields(logrus.Fields{
				"transaction_id":        e.TransactionId.String(),
				"expected_character_id": payload.CharacterId,
				"actual_character_id":   e.CharacterId,
				"step_id":               decision.Step.StepId(),
			}).Error("Character ID mismatch in CreateAndEquipAsset creation event.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}
		if payload.Item.TemplateId != e.TemplateId {
			saga.LogSkip(l, logrus.Fields{
				"transaction_id":       e.TransactionId.String(),
				"step_id":              decision.Step.StepId(),
				"step_action":          decision.Step.Action(),
				"event_kind":           saga.EventKindAssetCreated,
				"event_template_id":    e.TemplateId,
				"expected_template_id": payload.Item.TemplateId,
			}, saga.SkipReasonTemplateIdMismatch)
			return
		}

		// Generate a unique step ID for the auto-equip step with proper timestamp format.
		autoEquipStepId := fmt.Sprintf("auto_equip_step_%d", time.Now().UnixNano())

		// Create the EquipAsset step.
		// Note: Using reasonable defaults for slot information since asset event doesn't provide it.
		// The item is typically created in the first available slot (assumption: slot 5).
		// Equipment slot -1 is typically used for equipment.
		it, _ := inventory.TypeFromItemId(item.Id(e.TemplateId))
		equipPayload := saga.EquipAssetPayload{
			CharacterId:   payload.CharacterId,
			InventoryType: uint32(it),
			Source:        e.Slot,
			Destination:   -1, // Assumption: equip to slot -1
		}

		equipStep := saga.NewStep[any](autoEquipStepId, saga.Pending, saga.EquipAsset, equipPayload)

		// Add the equip step to the saga after the current step (should be executed next).
		if err := sagaProcessor.AddStepAfterCurrent(e.TransactionId, equipStep); err != nil {
			l.WithFields(logrus.Fields{
				"transaction_id":     e.TransactionId.String(),
				"character_id":       e.CharacterId,
				"auto_equip_step_id": autoEquipStepId,
				"step_id":            decision.Step.StepId(),
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
			"original_step_id":   decision.Step.StepId(),
		}).Info("Successfully added auto-equip step for CreateAndEquipAsset action to be executed next.")
	}

	// Emit the notice using event's templateId + quantity (payload.CharacterId
	// and ShowEffect still come from the step — preserves task-014 behaviour).
	emitRewardNoticeForCurrentStep(l, ctx, e.TransactionId, e.TemplateId, e.Body.Quantity)

	// Complete the current step (either regular creation or CreateAndEquipAsset).
	_ = sagaProcessor.StepCompletedWithResult(e.TransactionId, true, map[string]any{"assetId": e.AssetId})
}

func handleAssetDeletedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.DeletedStatusEventBody]) {
	if e.Type != asset2.StatusEventTypeDeleted {
		return
	}
	p := saga.NewProcessor(l, ctx)
	decision, ok := p.AcceptEvent(e.TransactionId, saga.EventKindAssetDeleted)
	if !ok {
		return
	}
	// Template-id guard for DestroyAsset / DestroyAssetFromSlot.
	switch decision.Step.Action() {
	case saga.DestroyAsset:
		payload, cast := decision.Step.Payload().(saga.DestroyAssetPayload)
		if !cast {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"step_id":        decision.Step.StepId(),
			}).Error("Invalid payload for DestroyAsset step.")
			_ = p.StepCompleted(e.TransactionId, false)
			return
		}
		if payload.TemplateId != e.TemplateId {
			saga.LogSkip(l, logrus.Fields{
				"transaction_id":       e.TransactionId.String(),
				"step_id":              decision.Step.StepId(),
				"step_action":          decision.Step.Action(),
				"event_kind":           saga.EventKindAssetDeleted,
				"event_template_id":    e.TemplateId,
				"expected_template_id": payload.TemplateId,
			}, saga.SkipReasonTemplateIdMismatch)
			return
		}
	case saga.DestroyAssetFromSlot:
		// No templateId on the payload — the event's templateId is authoritative.
	}
	// DeletedStatusEventBody carries no quantity — pass 0. DestroyAssetFromSlot
	// steps use their payload Quantity internally; DestroyAsset is driven by
	// payload semantics (RemoveAll / Quantity) and the full-delete case is the
	// common path here.
	emitRewardNoticeForCurrentStep(l, ctx, e.TransactionId, e.TemplateId, 0)
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleAssetQuantityUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.QuantityChangedEventBody]) {
	if e.Type != asset2.StatusEventTypeQuantityChanged {
		return
	}
	p := saga.NewProcessor(l, ctx)
	decision, ok := p.AcceptEvent(e.TransactionId, saga.EventKindAssetQuantityChanged)
	if !ok {
		return
	}
	// Template-id guard matches on the step's payload templateId.
	var expectedTemplateId uint32
	switch decision.Step.Action() {
	case saga.AwardAsset:
		pl, cast := decision.Step.Payload().(saga.AwardItemActionPayload)
		if !cast {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"step_id":        decision.Step.StepId(),
			}).Error("Invalid payload for AwardAsset step.")
			_ = p.StepCompleted(e.TransactionId, false)
			return
		}
		expectedTemplateId = pl.Item.TemplateId
	case saga.DestroyAsset:
		pl, cast := decision.Step.Payload().(saga.DestroyAssetPayload)
		if !cast {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"step_id":        decision.Step.StepId(),
			}).Error("Invalid payload for DestroyAsset step.")
			_ = p.StepCompleted(e.TransactionId, false)
			return
		}
		expectedTemplateId = pl.TemplateId
	case saga.DestroyAssetFromSlot:
		// No templateId on payload.
		expectedTemplateId = e.TemplateId
	}
	if expectedTemplateId != 0 && expectedTemplateId != e.TemplateId {
		saga.LogSkip(l, logrus.Fields{
			"transaction_id":       e.TransactionId.String(),
			"step_id":              decision.Step.StepId(),
			"step_action":          decision.Step.Action(),
			"event_kind":           saga.EventKindAssetQuantityChanged,
			"event_template_id":    e.TemplateId,
			"expected_template_id": expectedTemplateId,
		}, saga.SkipReasonTemplateIdMismatch)
		return
	}
	emitRewardNoticeForCurrentStep(l, ctx, e.TransactionId, e.TemplateId, e.Body.Quantity)
	_ = p.StepCompletedWithResult(e.TransactionId, true, map[string]any{"assetId": e.AssetId})
}

func handleAssetMovedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) {
	if e.Type != asset2.StatusEventTypeMoved {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindAssetMoved); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}
