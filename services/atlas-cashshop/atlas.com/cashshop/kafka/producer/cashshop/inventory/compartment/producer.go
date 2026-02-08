package compartment

import (
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/kafka/message/cashshop/compartment"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// CreateStatusEventProvider creates a provider for compartment creation events
// According to the requirements, it should always include compartmentId and type, and include capacity in the body
func CreateStatusEventProvider(compartmentId uuid.UUID, compartmentType byte, capacity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Using 0 as the key since we don't have a numeric ID to use
	value := &compartment.StatusEvent[compartment.StatusEventCreatedBody]{
		CompartmentId:   compartmentId,
		CompartmentType: compartmentType,
		Type:            compartment.StatusEventTypeCreated,
		Body: compartment.StatusEventCreatedBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// UpdateStatusEventProvider creates a provider for compartment update events
func UpdateStatusEventProvider(compartmentId uuid.UUID, compartmentType byte, capacity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Using 0 as the key since we don't have a numeric ID to use
	value := &compartment.StatusEvent[compartment.StatusEventUpdatedBody]{
		CompartmentId:   compartmentId,
		CompartmentType: compartmentType,
		Type:            compartment.StatusEventTypeUpdated,
		Body: compartment.StatusEventUpdatedBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// DeleteStatusEventProvider creates a provider for compartment deletion events
// According to the requirements, it should always include compartmentId and type, and have an empty body
func DeleteStatusEventProvider(compartmentId uuid.UUID, compartmentType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Using 0 as the key since we don't have a numeric ID to use
	value := &compartment.StatusEvent[compartment.StatusEventDeletedBody]{
		CompartmentId:   compartmentId,
		CompartmentType: compartmentType,
		Type:            compartment.StatusEventTypeDeleted,
		Body:            compartment.StatusEventDeletedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func ErrorStatusEventProvider(compartmentId uuid.UUID, compartmentType byte, error string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Using 0 as the key since we don't have a numeric ID to use
	value := &compartment.StatusEvent[compartment.StatusEventErrorBody]{
		CompartmentId:   compartmentId,
		CompartmentType: compartmentType,
		Type:            cashshop.StatusEventTypeError,
		Body: compartment.StatusEventErrorBody{
			ErrorCode:     error,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AcceptedStatusEventProvider(accountId uint32, characterId uint32, compartmentId uuid.UUID, compartmentType byte, transactionId uuid.UUID, assetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.StatusEventAcceptedBody]{
		AccountId:       accountId,
		CharacterId:     characterId,
		CompartmentId:   compartmentId,
		CompartmentType: compartmentType,
		Type:            compartment.StatusEventTypeAccepted,
		Body: compartment.StatusEventAcceptedBody{
			TransactionId: transactionId,
			AssetId:       assetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReleasedStatusEventProvider(accountId uint32, characterId uint32, compartmentId uuid.UUID, compartmentType byte, transactionId uuid.UUID, assetId uint32, cashId int64, templateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.StatusEventReleasedBody]{
		AccountId:       accountId,
		CharacterId:     characterId,
		CompartmentId:   compartmentId,
		CompartmentType: compartmentType,
		Type:            compartment.StatusEventTypeReleased,
		Body: compartment.StatusEventReleasedBody{
			TransactionId: transactionId,
			AssetId:       assetId,
			CashId:        cashId,
			TemplateId:    templateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
