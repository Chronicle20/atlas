package compartment

import (
	"atlas-cashshop/kafka/message/character/compartment"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func IncreaseCapacityCommandProvider(characterId uint32, inventoryType byte, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.IncreaseCapacityCommandBody]{
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandIncreaseCapacity,
		Body: compartment.IncreaseCapacityCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
