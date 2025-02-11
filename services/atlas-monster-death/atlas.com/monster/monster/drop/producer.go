package drop

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func spawnDropCommandProvider(worldId byte, channelId byte, mapId uint32, itemId uint32, quantity uint32, mesos uint32, dropType byte, x int16, y int16, ownerId uint32, ownerPartyId uint32, dropperId uint32, dropperX int16, dropperY int16, playerDrop bool, mod byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &command[spawnCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandTypeSpawn,
		Body: spawnCommandBody{
			ItemId:       itemId,
			Quantity:     quantity,
			Mesos:        mesos,
			DropType:     dropType,
			X:            x,
			Y:            y,
			OwnerId:      ownerId,
			OwnerPartyId: ownerPartyId,
			DropperId:    dropperId,
			DropperX:     dropperX,
			DropperY:     dropperY,
			PlayerDrop:   playerDrop,
			Mod:          mod,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
