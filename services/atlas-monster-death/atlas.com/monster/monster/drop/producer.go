package drop

import (
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func spawnDropCommandProvider(f field.Model, itemId uint32, quantity uint32, mesos uint32, dropType byte, x int16, y int16, ownerId uint32, ownerPartyId uint32, dropperId uint32, dropperX int16, dropperY int16, playerDrop bool, mod byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	cmd := commandFromField(f, CommandTypeSpawn, spawnCommandBody{
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
	})
	return producer.SingleMessageProvider(key, &cmd)
}
