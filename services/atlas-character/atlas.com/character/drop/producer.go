package drop

import (
	drop2 "atlas-character/kafka/message/drop"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func dropMesoProvider(field field.Model, mesos uint32, dropType byte, x int16, y int16, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(field.MapId()))
	value := &drop2.Command[drop2.SpawnFromCharacterCommandBody]{
		WorldId:   field.WorldId(),
		ChannelId: field.ChannelId(),
		MapId:     field.MapId(),
		Instance:  field.Instance(),
		Type:      drop2.CommandTypeSpawnFromCharacter,
		Body: drop2.SpawnFromCharacterCommandBody{
			Mesos:      mesos,
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

func cancelReservationCommandProvider(field field.Model, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(field.MapId()))
	value := &drop2.Command[drop2.CancelReservationCommandBody]{
		WorldId:   field.WorldId(),
		ChannelId: field.ChannelId(),
		MapId:     field.MapId(),
		Instance:  field.Instance(),
		Type:      drop2.CommandTypeCancelReservation,
		Body: drop2.CancelReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestPickUpCommandProvider(field field.Model, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(field.MapId()))
	value := &drop2.Command[drop2.RequestPickUpCommandBody]{
		WorldId:   field.WorldId(),
		ChannelId: field.ChannelId(),
		MapId:     field.MapId(),
		Instance:  field.Instance(),
		Type:      drop2.CommandTypeRequestPickUp,
		Body: drop2.RequestPickUpCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
