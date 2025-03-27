package inventory

import (
	inventory2 "atlas-consumables/kafka/message/inventory"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func requestReserveCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, reserves []Reserves) model.Provider[[]kafka.Message] {
	items := make([]inventory2.ItemBody, 0)
	for _, res := range reserves {
		items = append(items, inventory2.ItemBody{
			Source:   res.Slot,
			ItemId:   res.ItemId,
			Quantity: res.Quantity,
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.RequestReserveCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          inventory2.CommandRequestReserve,
		Body: inventory2.RequestReserveCommandBody{
			TransactionId: transactionId,
			Items:         items,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func consumeCommandProvider(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.ConsumeCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          inventory2.CommandConsume,
		Body: inventory2.ConsumeCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func destroyCommandProvider(characterId uint32, inventoryType inventory.Type, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.DestroyCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          inventory2.CommandDestroy,
		Body: inventory2.DestroyCommandBody{
			Slot: slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelReservationCommandProvider(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.CancelReservationCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          inventory2.CommandCancelReservation,
		Body: inventory2.CancelReservationCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
