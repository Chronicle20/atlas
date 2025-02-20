package inventory

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func equipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &equipItemCommand{
		CharacterId: characterId,
		Source:      source,
		Destination: destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func unequipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &unequipItemCommand{
		CharacterId: characterId,
		Source:      source,
		Destination: destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func moveItemCommandProvider(characterId uint32, inventoryType byte, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &moveItemCommand{
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Source:        source,
		Destination:   destination,
	}
	return producer.SingleMessageProvider(key, value)
}

func dropItemCommandProvider(m _map.Model, characterId uint32, inventoryType byte, source int16, quantity int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &dropItemCommand{
		WorldId:       byte(m.WorldId()),
		ChannelId:     byte(m.ChannelId()),
		MapId:         uint32(m.MapId()),
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Source:        source,
		Quantity:      quantity,
	}
	return producer.SingleMessageProvider(key, value)
}
