package drop

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// SpawnDropCommandProvider creates a Kafka message for spawning a drop
func SpawnDropCommandProvider(
	transactionId uuid.UUID,
	worldId world.Id,
	channelId channel.Id,
	mapId _map.Id,
	itemId uint32,
	quantity uint32,
	mesos uint32,
	dropType byte,
	x int16,
	y int16,
	ownerId uint32,
	ownerPartyId uint32,
	dropperId uint32,
	dropperX int16,
	dropperY int16,
	playerDrop bool,
	mod byte,
) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &Command[CommandSpawnBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		ChannelId:     channelId,
		MapId:         mapId,
		Type:          CommandTypeSpawn,
		Body: CommandSpawnBody{
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
