package asset

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/saga"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"time"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status_event")(asset2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(asset2.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetCreatedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetQuantityUpdatedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMovedEvent)))
	}
}

func handleAssetCreatedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody[any]]) {
	if e.Type != asset2.StatusEventTypeCreated {
		return
	}

	sagaProcessor := saga.NewProcessor(l, ctx)

	// Get the saga to check if this is a CreateAndEquipAsset step
	s, err := sagaProcessor.GetById(e.TransactionId)
	if err != nil {
		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   e.CharacterId,
		}).Debug("Unable to locate saga for asset created event.")
		_ = sagaProcessor.StepCompleted(e.TransactionId, true)
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
	if currentStep.Action == saga.CreateAndEquipAsset {
		// Extract the payload to get the character ID and inventory type
		createPayload, ok := currentStep.Payload.(saga.CreateAndEquipAssetPayload)
		if !ok {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"character_id":   e.CharacterId,
				"step_id":        currentStep.StepId,
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
				"step_id":               currentStep.StepId,
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

		equipStep := saga.Step[any]{
			StepId:    autoEquipStepId,
			Status:    saga.Pending,
			Action:    saga.EquipAsset,
			Payload:   equipPayload,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Add the equip step to the saga after the current step (should be executed next)
		err = sagaProcessor.AddStepAfterCurrent(e.TransactionId, equipStep)
		if err != nil {
			l.WithFields(logrus.Fields{
				"transaction_id":     e.TransactionId.String(),
				"character_id":       e.CharacterId,
				"auto_equip_step_id": autoEquipStepId,
				"step_id":            currentStep.StepId,
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
			"original_step_id":   currentStep.StepId,
		}).Info("Successfully added auto-equip step for CreateAndEquipAsset action to be executed next.")
	}

	// Complete the current step (either regular creation or CreateAndEquipAsset)
	_ = sagaProcessor.StepCompleted(e.TransactionId, true)
}

func handleAssetQuantityUpdatedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.QuantityChangedEventBody]) {
	if e.Type != asset2.StatusEventTypeQuantityChanged {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleAssetMovedEvent(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) {
	if e.Type != asset2.StatusEventTypeMoved {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}
