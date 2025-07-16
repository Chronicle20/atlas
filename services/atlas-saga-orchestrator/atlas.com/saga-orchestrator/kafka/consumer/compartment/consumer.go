package compartment

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	"atlas-saga-orchestrator/kafka/message/compartment"
	"atlas-saga-orchestrator/saga"
	"context"
	"fmt"
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
			rf(consumer2.NewConfig(l)("compartment_status_event")(compartment.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(compartment.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentCreatedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentDeletedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentEquippedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentUnequippedEvent)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCompartmentErrorEvent)))
	}
}

func handleCompartmentCreatedEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.CreatedStatusEventBody]) {
	if e.Type != compartment.StatusEventTypeCreated {
		return
	}

	sagaProcessor := saga.NewProcessor(l, ctx)
	
	// Get the saga to check if this is a CreateAndEquipAsset step
	s, err := sagaProcessor.GetById(e.TransactionId)
	if err != nil {
		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   e.CharacterId,
		}).Debug("Unable to locate saga for compartment created event.")
		_ = sagaProcessor.StepCompleted(e.TransactionId, true)
		return
	}

	// Get the current step to check if it's a CreateAndEquipAsset action
	currentStep, ok := s.GetCurrentStep()
	if !ok {
		l.WithFields(logrus.Fields{
			"transaction_id": e.TransactionId.String(),
			"character_id":   e.CharacterId,
		}).Debug("No current step found for compartment created event.")
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
			}).Error("Invalid payload for CreateAndEquipAsset step.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}

		// Generate a unique step ID for the auto-equip step
		autoEquipStepId := fmt.Sprintf("auto_equip_step_%d", time.Now().UnixNano())
		
		// Create the EquipAsset step
		// Note: Using reasonable defaults for slot information since compartment event doesn't provide it
		// The item is typically created in the first available slot (assumption: slot 5)
		// Equipment slot -1 is typically used for equipment
		equipPayload := saga.EquipAssetPayload{
			CharacterId:   createPayload.CharacterId,
			InventoryType: uint32(e.Body.Type), // Use the type from the created event
			Source:        5,                   // Assumption: created item is in slot 5
			Destination:   -1,                  // Assumption: equip to slot -1
		}

		equipStep := saga.Step[any]{
			StepId:    autoEquipStepId,
			Status:    saga.Pending,
			Action:    saga.EquipAsset,
			Payload:   equipPayload,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Add the equip step to the saga
		err = sagaProcessor.AddStep(e.TransactionId, equipStep)
		if err != nil {
			l.WithFields(logrus.Fields{
				"transaction_id": e.TransactionId.String(),
				"character_id":   e.CharacterId,
				"error":          err.Error(),
			}).Error("Failed to add equip step to saga.")
			_ = sagaProcessor.StepCompleted(e.TransactionId, false)
			return
		}

		l.WithFields(logrus.Fields{
			"transaction_id":      e.TransactionId.String(),
			"character_id":        e.CharacterId,
			"auto_equip_step_id":  autoEquipStepId,
			"inventory_type":      e.Body.Type,
		}).Info("Successfully added auto-equip step for CreateAndEquipAsset action.")
	}

	// Complete the current step (either regular creation or CreateAndEquipAsset)
	_ = sagaProcessor.StepCompleted(e.TransactionId, true)
}

func handleCompartmentDeletedEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.DeletedStatusEventBody]) {
	if e.Type != compartment.StatusEventTypeDeleted {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleCompartmentEquippedEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.EquippedEventBody]) {
	if e.Type != compartment.StatusEventTypeEquipped {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleCompartmentUnequippedEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.UnequippedEventBody]) {
	if e.Type != compartment.StatusEventTypeUnequipped {
		return
	}
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, true)
}

func handleCompartmentErrorEvent(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ErrorEventBody]) {
	if e.Type != compartment.StatusEventTypeError {
		return
	}
	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"error_code":     e.Body.ErrorCode,
		"character_id":   e.CharacterId,
	}).Error("Compartment operation failed")
	_ = saga.NewProcessor(l, ctx).StepCompleted(e.TransactionId, false)
}