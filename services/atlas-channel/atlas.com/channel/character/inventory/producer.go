package inventory

import (
	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func equipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[equipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          CommandEquip,
		Body: equipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func unequipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[unequipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          CommandUnequip,
		Body: unequipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func moveItemCommandProvider(characterId uint32, inventoryType byte, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[moveCommandBody]{
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          CommandMove,
		Body: moveCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func dropItemCommandProvider(m _map.Model, characterId uint32, inventoryType byte, source int16, quantity int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[dropCommandBody]{
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          CommandDrop,
		Body: dropCommandBody{
			WorldId:   byte(m.WorldId()),
			ChannelId: byte(m.ChannelId()),
			MapId:     uint32(m.MapId()),
			Source:    source,
			Quantity:  quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
