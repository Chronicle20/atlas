package drop

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func dropMesoProvider(worldId byte, channelId byte, mapId uint32, mesos uint32, dropType byte, x int16, y int16, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &command[spawnFromCharacterCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandTypeSpawnFromCharacter,
		Body: spawnFromCharacterCommandBody{
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

func cancelReservationCommandProvider(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &command[cancelReservationCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandTypeCancelReservation,
		Body: cancelReservationCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestPickUpCommandProvider(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &command[requestPickUpCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandTypeRequestPickUp,
		Body: requestPickUpCommandBody{
			DropId:      dropId,
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
