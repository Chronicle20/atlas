package inventory

import (
	"atlas-inventory/kafka/message/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(transactionId uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.CreatedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          inventory.StatusEventTypeCreated,
		Body:          inventory.CreatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func CreationFailedEventStatusProvider(transactionId uuid.UUID, characterId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.CreationFailedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          inventory.StatusEventTypeCreationFailed,
		Body:          inventory.CreationFailedStatusEventBody{Reason: reason},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedEventStatusProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.DeletedStatusEventBody]{
		CharacterId: characterId,
		Type:        inventory.StatusEventTypeDeleted,
		Body:        inventory.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
