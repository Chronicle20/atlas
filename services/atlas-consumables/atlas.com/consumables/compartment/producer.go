package compartment

import (
	"atlas-consumables/kafka/message/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func requestReserveCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, reserves []Reserves) model.Provider[[]kafka.Message] {
	items := make([]compartment.ItemBody, 0)
	for _, res := range reserves {
		items = append(items, compartment.ItemBody{
			Source:   res.Slot,
			ItemId:   res.ItemId,
			Quantity: res.Quantity,
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.RequestReserveCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandRequestReserve,
		Body: compartment.RequestReserveCommandBody{
			TransactionId: transactionId,
			Items:         items,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func consumeCommandProvider(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.ConsumeCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandConsume,
		Body: compartment.ConsumeCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func destroyCommandProvider(characterId uint32, inventoryType inventory.Type, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.DestroyCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandDestroy,
		Body: compartment.DestroyCommandBody{
			Slot: slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelReservationCommandProvider(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.CancelReservationCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandCancelReservation,
		Body: compartment.CancelReservationCommandBody{
			TransactionId: transactionId,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
