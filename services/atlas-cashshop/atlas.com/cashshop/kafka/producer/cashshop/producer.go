package cashshop

import (
	"atlas-cashshop/kafka/message/cashshop"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func ErrorStatusEventProvider(characterId uint32, error string, code byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.ErrorEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeError,
		Body: cashshop.ErrorEventBody{
			Error: error,
			Code:  code,
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
