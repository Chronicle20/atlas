package cashshop

import (
	"atlas-cashshop/kafka/message/cashshop"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ErrorStatusEventProvider(characterId uint32, error string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.ErrorEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeError,
		Body: cashshop.ErrorEventBody{
			Error: error,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func InventoryCapacityIncreasedStatusEventProvider(characterId uint32, inventoryType byte, capacity uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.InventoryCapacityIncreasedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeInventoryCapacityIncreased,
		Body: cashshop.InventoryCapacityIncreasedBody{
			InventoryType: inventoryType,
			Capacity:      capacity,
			Amount:        amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func PurchaseStatusEventProvider(characterId uint32, templateId, price uint32, compartmentId uuid.UUID, assetId uuid.UUID, itemId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.PurchaseEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypePurchase,
		Body: cashshop.PurchaseEventBody{
			TemplateId:    templateId,
			Price:         price,
			CompartmentId: compartmentId,
			AssetId:       assetId,
			ItemId:        itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CashItemMovedToInventoryStatusEventProvider(characterId uint32, compartmentId uuid.UUID, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.CashItemMovedToInventoryEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeCashItemMovedToInventory,
		Body: cashshop.CashItemMovedToInventoryEventBody{
			CompartmentId: compartmentId,
			Slot:          slot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
