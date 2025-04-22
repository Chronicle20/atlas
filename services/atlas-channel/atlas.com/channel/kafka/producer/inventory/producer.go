package inventory

import (
	inventory2 "atlas-channel/kafka/message/inventory"
	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func EquipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.EquipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          inventory2.CommandEquip,
		Body: inventory2.EquipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UnequipItemCommandProvider(characterId uint32, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.UnequipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          inventory2.CommandUnequip,
		Body: inventory2.UnequipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func MoveItemCommandProvider(characterId uint32, inventoryType byte, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.MoveCommandBody]{
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          inventory2.CommandMove,
		Body: inventory2.MoveCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DropItemCommandProvider(m _map.Model, characterId uint32, inventoryType byte, source int16, quantity int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory2.Command[inventory2.DropCommandBody]{
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          inventory2.CommandDrop,
		Body: inventory2.DropCommandBody{
			WorldId:   byte(m.WorldId()),
			ChannelId: byte(m.ChannelId()),
			MapId:     uint32(m.MapId()),
			Source:    source,
			Quantity:  quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
