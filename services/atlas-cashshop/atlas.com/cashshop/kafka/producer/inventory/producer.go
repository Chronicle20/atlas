package inventory

import (
	"atlas-cashshop/kafka/message/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func IncreaseCapacityCommandProvider(characterId uint32, inventoryType byte, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.Command[inventory.IncreaseCapacityCommandBody]{
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          inventory.CommandIncreaseCapacity,
		Body: inventory.IncreaseCapacityCommandBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
