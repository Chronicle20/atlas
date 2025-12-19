package drop

import (
	"atlas-inventory/kafka/message/drop"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func EquipmentProvider(m _map.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.MapId()))
	value := &drop.Command[drop.SpawnFromCharacterCommandBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		Type:      drop.CommandTypeSpawnFromCharacter,
		Body: drop.SpawnFromCharacterCommandBody{
			ItemId:      itemId,
			EquipmentId: equipmentId,
			Quantity:    1,
			DropType:    dropType,
			X:           x,
			Y:           y,
			OwnerId:     ownerId,
			DropperId:   ownerId,
			DropperX:    x,
			DropperY:    y,
			PlayerDrop:  true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ItemProvider(m _map.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.MapId()))
	value := &drop.Command[drop.SpawnFromCharacterCommandBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		Type:      drop.CommandTypeSpawnFromCharacter,
		Body: drop.SpawnFromCharacterCommandBody{
			ItemId:     itemId,
			Quantity:   quantity,
			DropType:   dropType,
			X:          x,
			Y:          y,
			OwnerId:    ownerId,
			DropperId:  ownerId,
			DropperX:   x,
			DropperY:   y,
			PlayerDrop: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelReservationCommandProvider(m _map.Model, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.MapId()))
	value := &drop.Command[drop.CancelReservationCommandBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		Type:      drop.CommandTypeCancelReservation,
		Body: drop.CancelReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestPickUpCommandProvider(m _map.Model, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.MapId()))
	value := &drop.Command[drop.RequestPickUpCommandBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		Type:      drop.CommandTypeRequestPickUp,
		Body: drop.RequestPickUpCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
