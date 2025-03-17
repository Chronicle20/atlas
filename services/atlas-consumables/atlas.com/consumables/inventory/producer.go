package inventory

import (
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func consumeCommandProvider(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[consumeCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          CommandConsume,
		Body: consumeCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelReservationCommandProvider(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[cancelReservationCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          CommandCancelReservation,
		Body: cancelReservationCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
