package drop

import (
	"atlas-inventory/kafka/message/drop"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func EquipmentProvider(f field.Model, itemId uint32, equipmentId uint32, dropType byte, x int16, y int16, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &drop.Command[drop.SpawnFromCharacterCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
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

func ItemProvider(f field.Model, itemId uint32, quantity uint32, dropType byte, x int16, y int16, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &drop.Command[drop.SpawnFromCharacterCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
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

func CancelReservationCommandProvider(f field.Model, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &drop.Command[drop.CancelReservationCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      drop.CommandTypeCancelReservation,
		Body: drop.CancelReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestPickUpCommandProvider(f field.Model, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &drop.Command[drop.RequestPickUpCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      drop.CommandTypeRequestPickUp,
		Body: drop.RequestPickUpCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
