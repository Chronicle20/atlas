package compartment

import (
	"atlas-channel/kafka/message/compartment"

	"github.com/Chronicle20/atlas-constants/inventory"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func EquipAssetCommandProvider(characterId uint32, inventoryType inventory.Type, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.EquipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandEquip,
		Body: compartment.EquipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UnequipAssetCommandProvider(characterId uint32, inventoryType inventory.Type, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.UnequipCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandUnequip,
		Body: compartment.UnequipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func MoveAssetCommandProvider(characterId uint32, inventoryType inventory.Type, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.MoveCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandMove,
		Body: compartment.MoveCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DropAssetCommandProvider(m _map.Model, characterId uint32, inventoryType inventory.Type, source int16, quantity int16, x int16, y int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.DropCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandDrop,
		Body: compartment.DropCommandBody{
			WorldId:   m.WorldId(),
			ChannelId: m.ChannelId(),
			MapId:     m.MapId(),
			Source:    source,
			Quantity:  quantity,
			X:         x,
			Y:         y,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func MergeCommandProvider(characterId uint32, inventoryType inventory.Type) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.MergeCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandMerge,
		Body:          compartment.MergeCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func SortCommandProvider(characterId uint32, inventoryType inventory.Type) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.SortCommandBody]{
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandSort,
		Body:          compartment.SortCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
